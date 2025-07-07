package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	utils "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (k *Kernel) llegaSyscallCPU(w http.ResponseWriter, r *http.Request) {
	var respuesta string

	if err := json.NewDecoder(r.Body).Decode(&respuesta); err != nil {
		fmt.Println("error creando la solicitud:", err)
		return
	}
	k.GestionarSyscalls(respuesta)
}

func (k *Kernel) GestionarSyscalls(respuesta string) {

	syscall := strings.Split(respuesta, " ")

	id_cpu, err := strconv.Atoi(syscall[IdCPU])

	if err != nil {
		log.Printf("[ERROR] (GestionarSyscalls) ID de CPU no es un número: %v", syscall[IdCPU])
		return
	}

	pc, err := strconv.Atoi(syscall[PC])

	if err != nil {
		log.Printf("[ERROR] (GestionarSyscalls) PC invalido: %v", syscall[PC])
		return
	}

	mutex_CPUsConectadas.Lock()
	cpu_ejecutando, existe := k.CPUsConectadas[id_cpu]
	if !existe {
		log.Printf("[ERROR] (GestionarSyscalls) ID de CPU inexistente: %d", id_cpu)
		mutex_CPUsConectadas.Unlock()
		return
	}
	pid := cpu_ejecutando.Pid
	mutex_CPUsConectadas.Unlock()

	cod_op := syscall[CodOp]

	utils.LoggerConFormato("## (%d) - Solicitó syscall: %s", pid, cod_op)
	switch cod_op {
	case "IO":
		// 2 20 IO AURICULARES 9000

		nombre := syscall[3]
		tiempo, _ := strconv.Atoi(syscall[4])
		k.GestionarIO(nombre, pid, pc, tiempo)
		//manejarIO
		//validar que exista la io
		//enviar mensaje a io

	case "INIT_PROC":
		// 2 20 INIT_PROC proceso1 256
		nombre_arch := syscall[3]
		tamanio, _ := strconv.Atoi(syscall[4])
		k.GestionarINIT_PROC(nombre_arch, tamanio, pc)

	case "DUMP_MEMORY":
		// 2 30 DUMP_MEMORY
		k.GestionarDUMP_MEMORY(pid)

	case "EXIT":
		// 2 30 EXIT
		//finalizarProc
		k.GestionarEXIT(pid)
	}

	mutex_CPUsConectadas.Lock()
	cpu_ejecutando, existe = k.CPUsConectadas[id_cpu]
	if !existe {
		log.Printf("[ERROR] (GestionarSyscalls) ID de CPU inexistente: %d", id_cpu)
		mutex_CPUsConectadas.Unlock()
		return
	}
	actualizarCPU(cpu_ejecutando, -1, 0, true)
	mutex_CPUsConectadas.Unlock()

	k.IntentarEnviarProcesoAExecute()

}

func (k *Kernel) GestionarIO(nombre_io string, pid, pc, duracion int) {

	//mutex IOs
	mutex_DispositivosIO.Lock()
	defer mutex_DispositivosIO.Unlock()

	iosMismoNombre, existeIO := k.DispositivosIO[nombre_io]

	if !existeIO {
		utils.LoggerConFormato("ERROR (IO) - No existe el dispositivo: %s", nombre_io)
		k.GestionarEXIT(pid)
		return
	}

	mutex_ProcesoPorEstado[EstadoBlock].Lock()
	mutex_ProcesoPorEstado[EstadoExecute].Lock()

	pcb := k.BuscarPorPidSinLock(EstadoExecute, pid)
	if pcb == nil {
		utils.LoggerConFormato("ERROR (IO) - No se encontró el PCB del proceso %d en EXECUTE", pid)
		mutex_ProcesoPorEstado[EstadoExecute].Unlock()
		mutex_ProcesoPorEstado[EstadoBlock].Unlock()
		return
	}

	pcb.Pc = pc

	k.MoverDeEstadoPorPid(EstadoExecute, EstadoBlock, pid, false)

	mutex_ProcesoPorEstado[EstadoExecute].Unlock()
	mutex_ProcesoPorEstado[EstadoBlock].Unlock()

	go k.temporizadorSuspension(pid)

	IO_seleccionada := k.buscarIOLibre(nombre_io)

	if IO_seleccionada == nil { //no hay io libre
		nuevo_proc_esperando := &ProcesoEsperandoIO{
			Pid:      pid,
			TiempoIO: duracion,
		}
		iosMismoNombre.ColaEspera = append(iosMismoNombre.ColaEspera, nuevo_proc_esperando)
		utils.LoggerConFormato("## (%d) - Encolado en espera de IO: %s", pid, nombre_io)
		return
	}
	// si hay io libre
	IO_seleccionada.PidOcupante = pid
	IO_seleccionada.Libre = false

	utils.LoggerConFormato("## (%d) - Bloqueado por IO: %s", pid, nombre_io)
	enviarProcesoAIO(IO_seleccionada, duracion)
}

func (k *Kernel) GestionarINIT_PROC(nombre_arch string, tamanio int, pc int) {

	nuevo_pcb := k.IniciarProceso(tamanio, nombre_arch)
	pid := nuevo_pcb.Pid
	k.AgregarAEstado(EstadoNew, nuevo_pcb, true)
	utils.LoggerConFormato(" (%d) Se crea el proceso - Estado: NEW", pid)

	unElemento, _ := k.UnicoEnNewYNadaEnSuspReady()

	if !unElemento {
		k.IntentarEnviarProcesoAReady(EstadoNew, pid)
	}
	// cpu_ejecutando.Pc = pc //Actualizar pc para cpu
}

func (k *Kernel) GestionarDUMP_MEMORY(pid int) {

	k.MoverDeEstadoPorPid(EstadoExecute, EstadoBlock, pid, true)
	go func() {
		fullURL := fmt.Sprintf("http://%s:%d/memoria/MEMORY_DUMP", k.Configuracion.Ip_memoria, k.Configuracion.Puerto_Memoria)
		respuesta, err := utils.EnviarStringConEspera("POST", fullURL, strconv.Itoa(pid))

		if err != nil || respuesta != "OK" {
			utils.LoggerConFormato("ERROR (GestionarDUMP_MEMORY) en la respuesta de Memoria")
			k.GestionarEXIT(pid)
			return
		}
		if !k.MoverDeEstadoPorPid(EstadoBlock, EstadoReady, pid, true) {
			utils.LoggerConFormato("ERROR (GestionarDUMP_MEMORY) no se encontó el proceso %d", pid)
		}
		utils.LoggerConFormato("## (%d) - DumpMemory finalizado correctamente", pid)
	}()

}

func (k *Kernel) GestionarEXIT(pid int) {
	//saco de execute el proceso que esta ejecutando y lo obtengo
	pcb := k.QuitarYObtenerPCB(EstadoExecute, pid, true)
	if pcb == nil {
		utils.LoggerConFormato("ERROR (GestionarEXIT) no se encontró el pid () asociado a la cpu () en Execute")
		return
	}
	//envio solicitud para eliminar proceso
	go k.EliminarProceso(pcb)
}
