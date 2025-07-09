package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (cpu *CPU) EsperarDatosKernel(w http.ResponseWriter, r *http.Request) {
	var respuesta string

	body_bytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Error leyendo la solicitud:", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}

	respuesta = string(body_bytes)

	datos := strings.Split(respuesta, " ")
	if len(datos) != 2 {
		http.Error(w, "Se recibio una formato incorrecto", http.StatusBadRequest)
		return
	}

	pid_aux, _ := strconv.Atoi(datos[0])
	pc_aux, _ := strconv.Atoi(datos[1])

	cpu.Proc_ejecutando.Pid = pid_aux
	cpu.Proc_ejecutando.Pc = pc_aux

	utils.LoggerConFormato("Me llego el proceso con pid %d de kernel", cpu.Proc_ejecutando.Pid)
	ch_esperar_datos <- struct{}{}

}

func (cpu *CPU) RegistrarCpu() error {

	respuesta, err := utils.FormatearUrlYEnviar(cpu.Url_kernel, "/registrar_cpu", true, "%s %s %d", cpu.Id, cpu.Config_CPU.Ip_CPU, cpu.Config_CPU.Puerto_CPU)
	fmt.Println(respuesta)

	if respuesta != "OK" || err != nil {
		return fmt.Errorf("no se puede registar la cpu %s", cpu.Id)
	}

	slog.Debug("CPU registrada correctamente!")
	return nil

}

func (cpu *CPU) EnviarSyscall(cod_op_syscall string, syscall string) {

	utils.FormatearUrlYEnviar(cpu.Url_kernel, "/syscall", false, "%s", syscall)

	slog.Debug("Debug - (enviarSyscall) - Syscall enviado correctamente",
		"syscall", cod_op_syscall)
}

func (cpu *CPU) ActualizarContexto() {
	utils.FormatearUrlYEnviar(cpu.Url_kernel, "/syscall", false, "%d %d", cpu.Proc_ejecutando.Pid, cpu.Proc_ejecutando.Pc)
	log.Println("Se envio el contexto por interrupcion correctamente")
}
