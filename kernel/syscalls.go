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

func (k *Kernel) RecibirSyscallCPU(w http.ResponseWriter, r *http.Request) {
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

	mutex_CPUsConectadas.Lock()

	if err != nil {
		log.Printf("[ERROR] (GestionarSyscalls) ID de CPU no es un número: %v", syscall[IdCPU])
		return
	}

	pc, err := strconv.Atoi(syscall[PC])

	if err != nil {
		log.Printf("[ERROR] (GestionarSyscalls) PC invalido: %v", syscall[PC])
		return
	}

	mutex_CPUsConectadas.Lock() //pendiente
	cpu_ejecutando, existe := k.CPUsConectadas[id_cpu]
	mutex_CPUsConectadas.Unlock()
	if !existe {
		log.Printf("[ERROR] (GestionarSyscalls) ID de CPU inexistente: %d", id_cpu)
		return
	}

	cod_op := syscall[CodOp]

	utils.LoggerConFormato("## (%d) - Solicitó syscall: %s", cpu_ejecutando.Pid, cod_op)
	switch cod_op {
	case "IO":
		// 2 20 IO AURICULARES 9000

		nombre := syscall[3]
		tiempo, _ := strconv.Atoi(syscall[4])
		cpu_ejecutando.Pc = pc
		k.ManejarIO(nombre, cpu_ejecutando, tiempo)
		//manejarIO
		//validar que exista la io
		//enviar mensaje a io

	case "INIT_PROC":
		// 2 20 INIT_PROC proceso1 256
		nombre_arch := syscall[3]
		tamanio, _ := strconv.Atoi(syscall[4])
		k.GestionarINIT_PROC(nombre_arch, tamanio, pc, cpu_ejecutando)

	case "DUMP_MEMORY":
		// 2 30 DUMP_MEMORY

		mensaje_DUMP_MEMORY := fmt.Sprintf("DUMP_MEMORY %d", cpu_ejecutando.Pid)
		utils.EnviarSolicitudHTTPString("POST", cpu_ejecutando.Url, mensaje_DUMP_MEMORY)
		cpu_ejecutando.Esta_libre = true

	case "EXIT":
		// 2 30 EXIT
		//finalizarProc
		k.GestionarEXIT(cpu_ejecutando)
	}

	k.IntentarEnviarProcesoAExecute()

}

func (k *Kernel) GestionarINIT_PROC(nombre_arch string, tamanio int, pc int, cpu_ejecutando *CPU) {

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

func (k *Kernel) GestionarEXIT(cpu_ejecutando *CPU) {
	//saco de execute el proceso que esta ejecutando y lo obtengo
	pcb := k.QuitarYObtenerPCB(EstadoExecute, cpu_ejecutando.Pid, true)

	//marco la cpu como libre
	cpu_ejecutando.Esta_libre = true

	//envio solicitud para eliminar proceso
	k.EliminarProceso(pcb)

	//marcar cpu como libre

}

func (k *Kernel) EliminarProceso(procesoAEliminar *PCB) {
	respuesta, err := k.solicitudEliminarProceso(procesoAEliminar.Pid)
	if err != nil {
		fmt.Println("Error al eliminar proceso en Memoria", err)
		return
	}

	if respuesta != "OK" {
		fmt.Println("Memoria no acepto la eliminacion del proceso")
		return
	}

	utils.LoggerConFormato("## (%d) - Finaliza el proceso", procesoAEliminar.Pid)

	utils.LoggerConFormato("## (%d) - Métricas de estado: ", procesoAEliminar.Pid)

	for estado := 0; estado < cantEstados; estado++ {
		utils.LoggerConFormato(
			"%s (%d) (%s),",
			estados_proceso[estado],
			procesoAEliminar.Me[estado],
			procesoAEliminar.Mt[estado].String(),
		)

	}
	k.IntentarEnviarProcesosAReady()
}

func (k *Kernel) solicitudEliminarProceso(pid int) (string, error) {

	url := fmt.Sprintf("http://%s:%d/memoria/exit/", k.Configuracion.Ip_memoria, k.Configuracion.Puerto_Memoria)
	pid_string := strconv.Itoa(pid)
	respuestaMemo, err := utils.EnviarSolicitudHTTPString("POST", url, pid_string)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	//Deberia responder "OK"
	return respuestaMemo, err

}
