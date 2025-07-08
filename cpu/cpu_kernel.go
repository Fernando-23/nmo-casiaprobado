package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
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

	utils.LoggerConFormato("Me llego el proceso con pid %d de kernel", *pid_ejecutando)
	ch_esperar_datos <- struct{}{}

}

func registrarCpu(url_kernel string) error {

	respuesta, err := utils.FormatearUrlYEnviar(url_kernel, "/registrar_cpu", true, "%s %s %d", id_cpu, config_CPU.Ip_CPU, config_CPU.Puerto_CPU)
	if respuesta != "OK" || err != nil {
		return fmt.Errorf("no se puede registar la cpu %s", id_cpu)
	}

	slog.Debug("CPU registrada correctamente!")
	return nil

}

func enviarSyscall(cod_op_syscall string, syscall string) {

	utils.FormatearUrlYEnviar(url_kernel, "/syscall", false, "%s", syscall)

	slog.Debug("Debug - (enviarSyscall) - Syscall enviado correctamente",
		"syscall", cod_op_syscall)
}

func actualizarContexto() {
	utils.FormatearUrlYEnviar(url_kernel, "/syscall", false, "%d %d", *pid_ejecutando, *pc_ejecutando)
	log.Println("Se envio el contexto por interrupcion correctamente")
}
