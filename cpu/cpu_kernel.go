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
	hay_datos = respuesta
	datos := strings.Split(respuesta, " ")
	if len(datos) != 2 {
		http.Error(w, "Se recibio una formato incorrecto", http.StatusBadRequest)
		return
	}

	pid_aux, _ := strconv.Atoi(datos[0])
	pc_aux, _ := strconv.Atoi(datos[1])

	*pid_ejecutando = pid_aux
	*pc_ejecutando = pc_aux

	utils.LoggerConFormato("Me llego el proceso con pid %d de kernel", *pid_ejecutando)
	sem_datos_kernel.Unlock()
}

func registrarCpu(url2_kernel string) {

	peticion := fmt.Sprintf("%s %s %d", id_cpu, config_CPU.Ip_CPU, config_CPU.Puerto_CPU)
	url := fmt.Sprintf("%s/cpu/registrar_cpu", url2_kernel)

	if respuesta, err := utils.EnviarSolicitudHTTPString("POST", url, peticion); err != nil {
		fmt.Println("Respuesta de Kernel", respuesta)
		log.Println("No se pudo registrar la cpu")
		return
	}

	log.Println("CPU registrada correctamente!")

}

func enviarSyscall(cod_op_syscall string, syscall string) {

	url := fmt.Sprintf("%s/cpu/syscall", url_kernel)

	fmt.Println("Intento enviar la syscall...")
	fmt.Println(syscall)
	utils.EnviarSolicitudHTTPString("POST", url, syscall)
	fmt.Println("Che aca le pase la syscall a kernel y soy un genio ;)")

	log.Println("Se envio correctamente la Syscall: ", cod_op_syscall)
}

func actualizarContexto() {

	url := fmt.Sprintf("%s/cpu/syscall", url_kernel)
	contexto := fmt.Sprintf("%d %d", *pid_ejecutando, *pc_ejecutando)
	utils.EnviarSolicitudHTTPString("POST", url, contexto)

	log.Println("Se envio el contexto por interrupcion correctamente")
}
