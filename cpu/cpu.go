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

	if len(os.Args) < 3 {
		fmt.Println("Faltan argumentos")
		os.Exit(1)
	}

	// Preparacion incial
	id_cpu := os.Args[1]
	instancia := os.Args[2]
	path_config_cpu := fmt.Sprintf("%s.json", instancia)
	noticiero_metereologico = time.Now()

	cpu := crearCPU(id_cpu, path_config_cpu)

	cpu.ChequarTLBActiva()
	cpu.ChequearCachePagsActiva()

	if tlb_activa {
		cpu.InicializarTLB()
	}

	if cache_pags_activa {
		cpu.IniciarCachePags()
	}

	// cliente.EnviarMensaje(config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria, "Conexion hecha con modulo CPU")

	handshake_memoria, err := utils.FormatearUrlYEnviar(cpu.Url_memoria, "/handshake", true, "CPU")
	fmt.Println(handshake_memoria)
	if handshake_memoria == "NO_OK" || err != nil {
		slog.Error("Error - (main) - NO se pudo establecen conexion con memoria")
		return
	}
	slog.Debug("(handshake_memoria)")

	aux_datos_mmu := strings.Split(handshake_memoria, " ")
	cant_niveles, _ = strconv.Atoi(aux_datos_mmu[0])
	//cant_entradas_tpag, _ = strconv.Atoi(aux_datos_mmu[1])
	tam_pag, _ = strconv.Atoi(aux_datos_mmu[2])

	// Conexion con Kernel
	slog.Debug("Iniciando handshake con kernel")
	if err := cpu.RegistrarCpu(); err != nil {
		slog.Error("Error registrando cpu - Me muero",
			"error", err,
		)
		return
	}

	var instruccion string

	//inicializo flags pq despues me olvido
	hay_que_actualizar_contexto = false
	tenemos_interrupt = false
	var requiere_realmente_desalojo string

	mux := http.NewServeMux()
	mux.HandleFunc("/cpu/dispatch", cpu.EsperarDatosKernel)
	mux.HandleFunc("/cpu/interrupt", cpu.RecibirInterrupt)

	socket_cpu := fmt.Sprintf(":%d", cpu.Config_CPU.Puerto_CPU)
	go http.ListenAndServe(socket_cpu, mux)
	// Ciclo de instruccion
	ch_esperar_datos = make(chan struct{})

	for {
		slog.Debug("Debug - (CicloInstruccion) - Esperando datos de kernel",
			"cpu", id_cpu)

		<-ch_esperar_datos

		for !hay_que_actualizar_contexto { //consulta el valor en un tiempo t no necesito sincronizar
			instruccion = cpu.Fetch()

			if instruccion == "TODO MAL" {
				slog.Error("Error - (Fetch) - Instruccion invalida")
				return
			}

			if instruccion == "" {
				slog.Error("No hay una instruccion valida asociado a este Program Counter.")
				return
			}

			utils.LoggerConFormato("## PID: %d - FETCH - Program Counter: %d",
				cpu.Proc_ejecutando.Pid, cpu.Proc_ejecutando.Pc)

			cod_op, operacion := cpu.Decode(instruccion)
			slog.Debug("Debug - (CicloInstruccion) - Instruccion a ejecutar",
				"instruccion", instruccion)

			requiere_realmente_desalojo = cpu.Execute(cod_op, operacion, instruccion)

			slog.Debug("Debug - (main) - Saliendo de execute, obtuve esto",
				"instruccion", cod_op, "requiere_desalojo", requiere_realmente_desalojo)
		}

		slog.Debug("Debug - (CicloInstruccion) - Se va a cambiar el contexto")
		cpu.LiberarCaches()

		CambiarValorActualizarContexto(false)

		cpu.ChequearSiTengoQueActualizarEnKernel(requiere_realmente_desalojo)

		utils.LoggerConFormato("## PID: %d - Finaliza la ejecucion", cpu.Proc_ejecutando.Pid)
	}
}
