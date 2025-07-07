package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
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

	idCPU, err := strconv.Atoi(syscall[IdCPU])

	if err != nil {
		slog.Error("Error - (GestionarSyscalls) - ID de CPU no es un número",
			"id_cpu", syscall[IdCPU])
		return
	}

	pc, err := strconv.Atoi(syscall[PC])

	if err != nil {
		slog.Error("Error - (GestionarSyscalls) - PC invalido",
			"pc", syscall[PC])
		return
	}

	mutex_CPUsConectadas.Lock()
	cpu_ejecutando, existe := k.CPUsConectadas[idCPU]
	if !existe {
		slog.Error("Error - (GestionarSyscalls) - ID de CPU inexistente",
			"id_cpu", idCPU)
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
		k.GestionarIO(nombre, pid, pc, tiempo, idCPU)
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
		k.GestionarDUMP_MEMORY(pid, idCPU)

	case "EXIT":
		// 2 30 EXIT
		//finalizarProc
		k.GestionarEXIT(pid, idCPU)
		//return false
		//default:
		//	return fmt.Errorf("syscall no reconocida %s", cod_op)
	}

	// liberar CPU y actualizar
	mutex_CPUsConectadas.Lock()
	cpu_ejecutando, existe = k.CPUsConectadas[idCPU]
	if !existe {
		log.Printf("[ERROR] (GestionarSyscalls) ID de CPU inexistente: %d", idCPU)
		mutex_CPUsConectadas.Unlock()
		return
	}
	actualizarCPU(cpu_ejecutando, -1, 0, true)
	mutex_CPUsConectadas.Unlock()

	k.IntentarEnviarProcesoAExecute()

}

func (k *Kernel) GestionarIO(nombreIO string, pid, pc, duracion, idCPU int) {

	//mutex IOs
	mutex_DispositivosIO.Lock()
	defer mutex_DispositivosIO.Unlock()

	iosMismoNombre, existeIO := k.DispositivosIO[nombreIO]

	if !existeIO {
		slog.Error("Error - (GestionarIO) - No existe el dispositivo",
			"nombre_io", nombreIO)
		k.GestionarEXIT(pid, idCPU)
		return
	}

	mutex_ProcesoPorEstado[EstadoBlock].Lock()
	mutex_ProcesoPorEstado[EstadoExecute].Lock()

	pcb := k.BuscarPorPidSinLock(EstadoExecute, pid)
	if pcb == nil {
		slog.Error("Error - (GestionarIO) - No se encontró el PCB del proceso en EXECUTE",
			"pid", pid)
		mutex_ProcesoPorEstado[EstadoExecute].Unlock()
		mutex_ProcesoPorEstado[EstadoBlock].Unlock()
		return
	}

	pcb.Pc = pc

	k.MoverDeEstadoPorPid(EstadoExecute, EstadoBlock, pid, false)

	mutex_ProcesoPorEstado[EstadoExecute].Unlock()
	mutex_ProcesoPorEstado[EstadoBlock].Unlock()

	go k.temporizadorSuspension(pid)

	IO_seleccionada := k.buscarIOLibre(nombreIO)

	if IO_seleccionada == nil { //no hay io libre
		nuevo_proc_esperando := &ProcesoEsperandoIO{
			Pid:      pid,
			TiempoIO: duracion,
		}
		iosMismoNombre.ColaEspera = append(iosMismoNombre.ColaEspera, nuevo_proc_esperando)
		utils.LoggerConFormato("## (%d) - Encolado en espera de IO: %s", pid, nombreIO)
		return
	}
	// si hay io libre
	IO_seleccionada.PidOcupante = pid
	IO_seleccionada.Libre = false

	utils.LoggerConFormato("## (%d) - Bloqueado por IO: %s", pid, nombreIO)
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

func (k *Kernel) GestionarDUMP_MEMORY(pid int, idCpu int) {

	if !k.MoverDeEstadoPorPid(EstadoExecute, EstadoBlock, pid, true) {
		slog.Error("GestionarDUMP_MEMORY: no se pudo mover a BLOCK",
			"pid", pid)
		return
	}

	go func(pid int) {

		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic en GestionarDUMP_MEMORY goroutine", "pid", pid, "panic", r)
			}
		}()

		fullURL := fmt.Sprintf("http://%s:%d/memoria/MEMORY_DUMP", k.Configuracion.Ip_memoria, k.Configuracion.Puerto_Memoria)
		respuesta, err := utils.EnviarStringConEspera("POST", fullURL, strconv.Itoa(pid))

		if err != nil || respuesta != "OK" {
			slog.Error("Error - (GestionarDUMP_MEMORY) - Dump fallido o respuesta inesperada",
				"pid", pid,
				"error", err,
				"respuesta", respuesta,
			)
			k.GestionarEXIT(pid, idCpu)
			return
		}
		if !k.MoverDeEstadoPorPid(EstadoBlock, EstadoReady, pid, true) {
			slog.Error("Error - (GestionarDUMP_MEMORY) - No se encontó el proceso",
				"pid", pid)
			return
		}
		utils.LoggerConFormato("## (%d) - DumpMemory finalizado correctamente", pid)
	}(pid)

}

func (k *Kernel) GestionarEXIT(pid int, idCPU int) {
	//saco de execute el proceso que esta ejecutando y lo obtengo
	pcb := k.QuitarYObtenerPCB(EstadoExecute, pid, true)
	if pcb == nil {
		slog.Error("Error - (GestionarEXIT) - No se encontró el pid asociado a la cpu en Execute",
			"pid", pid,
			"cpu", idCPU,
		)
		return
	}
	//envio solicitud para eliminar proceso
	go k.EliminarProceso(pcb, true)
}
