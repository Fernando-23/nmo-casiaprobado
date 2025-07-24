package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
	servidor "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Servidor"
)

func main() {

	fmt.Println("Iniciando Kernel...")
	//cliente.ConfigurarLogger("kernel")
	kernel := &Kernel{
		CPUsConectadas: make(map[int]*CPU),
		Configuracion:  new(ConfigKernel),
		DispositivosIO: make(map[string]*InstanciasPorDispositivo),
	}

	if len(os.Args) < 4 { // ruta archivo-pseudo tamanio
		slog.Error("[main] Cantidad de argumentos incorrecta. Uso: ruta <archivo-pseudo> <tamanio>")
		os.Exit(1)
	}
	args := os.Args

	archivoPseudo := args[1]
	tamanio, _ := strconv.Atoi(args[2])
	instancia := args[3]
	ruta_config := fmt.Sprintf("%s.json", instancia)

	err := IniciarConfiguracion(ruta_config, kernel.Configuracion)
	url_memo = fmt.Sprintf("http://%s:%d/memoria", kernel.Configuracion.Ip_memoria, kernel.Configuracion.Puerto_Memoria)
	kernel.InicializarMapaDeEstados()

	if err != nil {
		fmt.Println("[main] Error al iniciar config")
		return
	}

	err = ConfigurarLogger("kernel", kernel.Configuracion.Log_leveL)
	if err != nil {
		fmt.Println("[main] Error al configurar logger:", err)
		os.Exit(1)
	}

	gb_cant_cpus_conectadas = 0
	slog.Debug("Parámetros iniciales", "archivo", archivoPseudo, "tamanio", tamanio)

	//Iniciar servidor
	mux := http.NewServeMux()

	mux.HandleFunc("/kernel/registrar_cpu", kernel.LlegaNuevaCPU) //SINCRO HECHA
	mux.HandleFunc("/kernel/interrupido", kernel.llegaFinInterrupcion)
	mux.HandleFunc("/kernel/syscall", kernel.llegaSyscallCPU)
	mux.HandleFunc("/kernel/registrar_io", kernel.llegaNuevaIO)         // SINCRO HECHA
	mux.HandleFunc("/kernel/desconectar_io", kernel.llegaDesconexionIO) // revisado y corregido 20/6
	mux.HandleFunc("/kernel/fin_io", kernel.llegaFinIO)                 // revisado y corregido 20/6
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)

	url_socket := fmt.Sprintf(":%d", kernel.Configuracion.Puerto_Kernel)

	go func() {
		if err := http.ListenAndServe(url_socket, mux); err != nil {
			slog.Error("Error al iniciar el servidor HTTP", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("Estado inicial del planificador largo plazo", "estado", "STOP")

	ch_avisoCPULibrePorId = make(map[int]chan struct{})
	//mutex_cpus = make(map[int]sync.Mutex)
	waitEnter := make(chan struct{}, 1)

	//ch_aviso_cpu_libre = make(chan struct{})

	// Lanzamos la rutina que espera el ENTER
	go esperarEnter(waitEnter)
	// Esperamos hasta que la gorutine avise que el ENTER fue presionado (queda bloqueada la main rutine)
	<-waitEnter

	slog.Info("Comenzando la planificación", "evento", "ENTER presionado por el usuario")
	slog.Info("Estado actual del planificador largo plazo", "estado", "RUNNING")

	pcb := kernel.IniciarProceso(tamanio, archivoPseudo)
	pid := pcb.Pid
	kernel.AgregarAEstado(EstadoNew, pcb, true)
	utils.LoggerConFormato("## (%d) Se crea el proceso - Estado: NEW", pid)

	unElemento, estaEnReady := kernel.UnicoEnNewYNadaEnSuspReady()

	if !estaEnReady || !unElemento {
		slog.Error("Condición inválida al iniciar planificación", "motivo", "primer proceso y no es único del sistema")
		return
	}

	//kernel.IntentarEnviarProcesoAExecute()

	//aca iria al magia

	for i := 1; i <= gb_cant_cpus_conectadas; i++ {
		go func(i int) {
			var cpu *CPU
			for {
				slog.Debug("Esperando canal")
				<-ch_avisoCPULibrePorId[i]
				slog.Debug("Paso el canal")
				mutex_CPUsConectadas.Lock()
				cpu = kernel.BuscarCPUPorID(i)
				mutex_CPUsConectadas.Unlock()

				kernel.IntentarEnviarProcesoAExecutePorCPU(cpu)

			}
		}(i)
	}

	// go func() {
	// 	for {
	// 		<-ch_aviso_cpu_libre
	// 		kernel.IntentarEnviarProcesoAExecute()
	// 	}
	// }()

	select {}
}

//por si la cosa no mejora...
//if archivoPseudo == "PLANI_CORTO_PLAZO" || archivoPseudo == "ESTABILIDAD_GENERAL" {
//go func() {
// 	for {
// 		time.Sleep(2 * time.Second)
// 		kernel.IntentarEnviarProcesoAExecute()
// 	}
// }()
//}
