package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
)

func main() {
	fmt.Println("Iniciando CPU...")
	// Preparacion incial
	config_CPU = iniciarConfiguracionIO("config.json")
	args := os.Args
	id_cpu = args[1]
	pid_ejecutando = new(int)
	pid_ejecutando = new(int)

	url_cpu = fmt.Sprintf("http://%s:%d", config_CPU.Ip_CPU, config_CPU.Puerto_CPU)
	url_kernel = fmt.Sprintf("http://%s:%d", config_CPU.Ip_Kernel, config_CPU.Puerto_Kernel)
	url_memo = fmt.Sprintf("http://%s:%d", config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria)
	nombre_logger := fmt.Sprintf("cpu %s", id_cpu)
	cliente.ConfigurarLogger(nombre_logger)

	cliente.EnviarMensaje(config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria, "Conexion hecha con modulo CPU")

	// Conexion con Kernel
	fmt.Println("Iniciando handshake con kernel")
	//time.Sleep(2000 * time.Nanosecond) //mas que nada para probar
	registrarCpu()

	var instruccion string

	hay_interrupcion = false
	mux := http.NewServeMux()
	mux.HandleFunc("/cpu/dispatch", esperarDatosKernel)
	mux.HandleFunc("/cpu/interrupt", recibirInterrupt)

	http.ListenAndServe(id_cpu, mux)
	// Ciclo de instruccion

	for {

		pid_log := strconv.Itoa(*pid_ejecutando)
		pc_log := strconv.Itoa(*pc_ejecutando)
		for !hay_interrupcion {

			instruccion = fetch()
			slog.Info("## PID: %s - FETCH - Program Counter: %s", pid_log, pc_log)

			if instruccion == "" {
				fmt.Printf("No hay una instruccion valida asociado a este Program Counter.")
				break
			}

			cod_op, operacion := decode(instruccion)
			execute(cod_op, operacion)

		}

		actualizarContexto()
		hay_interrupcion = false

	}

}
