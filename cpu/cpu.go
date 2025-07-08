package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func main() {
	fmt.Println("Iniciando CPU...")
	// Preparacion incial
	args := os.Args
	id_cpu = args[1]
	config_CPU = &ConfigCPU{}
	path_config_cpu := fmt.Sprintf("cpu%s.json", id_cpu)
	utils.IniciarConfiguracion(path_config_cpu, config_CPU)
	pid_ejecutando = new(int)
	pc_ejecutando = new(int)
	noticiero_metereologico = time.Now()

	//url_cpu = fmt.Sprintf("http://%s:%d", config_CPU.Ip_CPU, config_CPU.Puerto_CPU)
	url_kernel = fmt.Sprintf("http://%s:%d/kernel", config_CPU.Ip_Kernel, config_CPU.Puerto_Kernel)
	url_memo = fmt.Sprintf("http://%s:%d/memoria", config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria)
	nombre_logger := fmt.Sprintf("cpu%s", id_cpu)
	utils.ConfigurarLogger(nombre_logger, config_CPU.Log_level)

	chequarTLBActiva()
	chequearCachePagsActiva()

	if tlb_activa {
		tlb = make([]*EntradaTLB, config_CPU.Cant_entradas_TLB)
		inicializarTLB()
	}

	if cache_pags_activa {
		cache_pags = make([]*EntradaCachePag, config_CPU.Cant_entradas_cache)
		reiniciarCachePags()
	}

	// cliente.EnviarMensaje(config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria, "Conexion hecha con modulo CPU")
	handshake_memoria, _ := utils.EnviarStringConEspera("GET", url_memo, "")
	aux_datos_mmu := strings.Split(handshake_memoria, " ")
	cant_niveles, _ = strconv.Atoi(aux_datos_mmu[0])
	cant_entradas_tpag, _ = strconv.Atoi(aux_datos_mmu[1])
	tam_pag, _ = strconv.Atoi(aux_datos_mmu[2])

	// Conexion con Kernel
	slog.Debug("Iniciando handshake con kernel")
	if err := registrarCpu(url_kernel); err != nil {
		slog.Error("Error registrando cpu - Me muero",
			"error", err,
		)
		return
	}

	var instruccion string

	hay_interrupcion = false
	mux := http.NewServeMux()
	mux.HandleFunc("/cpu/dispatch", esperarDatosKernel)
	mux.HandleFunc("/cpu/interrupt", recibirInterrupt)

	socket_cpu := fmt.Sprintf(":%d", config_CPU.Puerto_CPU)
	go http.ListenAndServe(socket_cpu, mux)
	// Ciclo de instruccion
	ch_esperar_datos = make(chan struct{})

	for {
		<-ch_esperar_datos

		for !hay_interrupcion { //consulta el valor en un tiempo t no necesito sincronizar
			instruccion = fetch(url_memo)

			if instruccion == "TODO MAL" {
				slog.Error("Error - (Fetch) - Instruccion invalida")
				return
			}

			utils.LoggerConFormato("## PID: %d - FETCH - Program Counter: %d", *pid_ejecutando, *pc_ejecutando)

			if instruccion == "" {
				slog.Error("No hay una instruccion valida asociado a este Program Counter.")
				break
			}

			cod_op, operacion := decode(instruccion)
			fmt.Println("Aca llego la instruccion: ", instruccion)
			execute(cod_op, operacion, instruccion)

		}

		actualizarContexto()
		liberarCaches()
		HabilitarInterrupt(false)

		utils.LoggerConFormato("## PID: %d - Finaliza la ejecucion", *pid_ejecutando)
	}
}
