package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
)

func main() {
	fmt.Println("Iniciando CPU...")
	// Preparacion incial
	config_CPU = &ConfigCPU{}
	utils.IniciarConfiguracion("config.json", config_CPU)
	args := os.Args
	id_cpu = args[1]
	pid_ejecutando = new(int)
	pc_ejecutando = new(int)

	url_cpu = fmt.Sprintf("http://%s:%d", config_CPU.Ip_CPU, config_CPU.Puerto_CPU)
	url_kernel = fmt.Sprintf("http://%s:%d", config_CPU.Ip_Kernel, config_CPU.Puerto_Kernel)
	url_memo = fmt.Sprintf("http://%s:%d", config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria)
	nombre_logger := fmt.Sprintf("cpu %s", id_cpu)
	cliente.ConfigurarLogger(nombre_logger)

	cliente.EnviarMensaje(config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria, "Conexion hecha con modulo CPU")

	// Conexion con Kernel
	fmt.Println("Iniciando handshake con kernel")
	registrarCpu(url_kernel)

	var instruccion string

	hay_interrupcion = false
	mux := http.NewServeMux()
	mux.HandleFunc("/cpu/dispatch", esperarDatosKernel)
	mux.HandleFunc("/cpu/interrupt", recibirInterrupt)

	socket_cpu := fmt.Sprintf(":%d", config_CPU.Puerto_CPU)
	go http.ListenAndServe(socket_cpu, mux)
	// Ciclo de instruccion

	for {
		sem_datos_kernel.Lock()
		utils.LoggerConFormato("Me unlockee jejeje")
		time.Sleep(9 * time.Second)
		pid_log := strconv.Itoa(*pid_ejecutando)
		pc_log := strconv.Itoa(*pc_ejecutando)
		for !hay_interrupcion {
			instruccion = fetch(url_memo)
			slog.Info(fmt.Sprintf("## PID: %s - FETCH - Program Counter: %s\n", pid_log, pc_log))
			fmt.Println("PRUEBA - PC: ", *pc_ejecutando)

			if instruccion == "" {
				fmt.Printf("No hay una instruccion valida asociado a este Program Counter.")
				break
			}

			cod_op, operacion := decode(instruccion)
			fmt.Println("Aca llego la instruccion: ", instruccion)
			execute(cod_op, operacion)

		}
		actualizarContexto()
		hay_interrupcion = false
	}

}
