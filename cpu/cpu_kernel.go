package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func esperarDatosKernel(w http.ResponseWriter, r *http.Request) {
	var respuesta string
	if err := json.NewDecoder(r.Body).Decode(&respuesta); err != nil {
		fmt.Println("Se produjo un error al deserializar: ", err)
		return
	}

	datos := strings.Split(respuesta, " ")
	if len(datos) != 2 {
		http.Error(w, "Se recibio una formato incorrecto", http.StatusBadRequest)
		return
	}

	pid_aux, _ := strconv.Atoi(datos[0])
	pc_aux, _ := strconv.Atoi(datos[1])

	*pid_ejecutando = pid_aux
	*pc_ejecutando = pc_aux

	sem_datos_kernel.Unlock()

}

func registrarCpu() {

	peticion := fmt.Sprintf("%s %d %s", id_cpu, config_CPU.Puerto_CPU, config_CPU.Ip_CPU)
	url := fmt.Sprintf("http://%s/cpu/registrar_cpu", url_kernel)

	if respuesta, err := utils.EnviarSolicitudHTTPString("POST", url, peticion); err != nil && respuesta != "OK" {
		log.Println("No se pudo registrar la cpu")
		return
	}

	log.Println("CPU registrada correctamente!")

}

func enviarSyscall(cod_op_syscall string, syscall string) {

	url := fmt.Sprintf("http://%s/cpu/syscall", url_kernel)

	utils.EnviarSolicitudHTTPString("POST", url, syscall)

	log.Println("Se envio correctamente la Syscall: ", cod_op_syscall)
}

func actualizarContexto() {

	url := fmt.Sprintf("http://%s/cpu/syscall", url_kernel)
	contexto := fmt.Sprintf("%d %d", *pid_ejecutando, *pc_ejecutando)
	utils.EnviarSolicitudHTTPString("POST", url, contexto)

	log.Println("Se envio el contexto por interrupcion correctamente")
}
