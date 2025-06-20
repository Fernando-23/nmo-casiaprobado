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

	syscall := strings.Split(respuesta, " ")

	fmt.Println("PRUEBA, syscall haber como llegas: ", syscall)

	mutex_syscall.Lock()
	defer mutex_syscall.Unlock()
	k.GestionarSyscalls(syscall)
}

func (k *Kernel) GestionarSyscalls(syscall []string) {

	id_cpu, err := strconv.Atoi(syscall[IdCPU])

	if err != nil || id_cpu >= len(k.cpusLibres) { // id_cpu != k.cpusLibres[IdCPU].ID
		log.Printf("ID de CPU invalido: %v", syscall[IdCPU])
		return
	}

	pc, err := strconv.Atoi(syscall[PC])

	if err != nil {
		log.Printf("PC invalido: %v", syscall[PC])
		return
	}

	cpu_ejecutando := k.cpusLibres[id_cpu]
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

	new_pcb := k.IniciarProceso(tamanio, nombre_arch)
	k.AgregarAEstado(EstadoNew, new_pcb)
	utils.LoggerConFormato(" (%d) Se crea el proceso - Estado: NEW", cpu_ejecutando.Pid)

	unElemento, err := k.ListaNewSoloYo()
	if err != nil {
		return
	}
	if !unElemento {
		k.PlaniLargoPlazo()
	}
	// cpu_ejecutando.Pc = pc //Actualizar pc para cpu
}

func (k *Kernel) GestionarEXIT(cpu_ejecutando *CPU) {
	respuesta, err := k.solicitudEliminarProceso(cpu_ejecutando.Pid)
	if err != nil {
		fmt.Println("Error", err)
	}

	if respuesta == "OK" {

		utils.LoggerConFormato("## (%d) - Finaliza el proceso", cpu_ejecutando.Pid)
		// utils.LoggerConFormato(
		// 	"## (%d) - Métricas de estado:"+
		// 		"NEW (%d) (%d), READY (%d) (%d), EXECUTE (%d) (%d)"+
		// 		",  ", cpu_ejecutando.Pid, k.procesoPorEstado[EstadoNew][cpu_ejecutando.Pid].)
		k.MoverDeEstadoPorPid(EstadoExecute, EstadoExit, cpu_ejecutando.Pid)
		k.QuitarDeEstado(cpu_ejecutando.Pid, EstadoExit)
		//k.EliminarProceso(cpu_ejecutando.Pid)
		cpu_ejecutando.Esta_libre = true
		//k.IntentarEnviarProcesoAReady()
	}
}

func (k *Kernel) solicitudEliminarProceso(pid int) (string, error) {

	url := fmt.Sprintf("http://%s:%d/memoria/exit/", k.ConfigKernel.Ip_memoria, k.ConfigKernel.Puerto_Memoria)
	pid_string := strconv.Itoa(pid)
	respuestaMemo, err := utils.EnviarSolicitudHTTPString("POST", url, pid_string)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	//Deberia responder "OK"
	return respuestaMemo, err

}
