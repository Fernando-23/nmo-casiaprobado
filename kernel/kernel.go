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

	fmt.Printf("Iniciando Kernel...")
	//cliente.ConfigurarLogger("kernel")
	kernel := &Kernel{
		CPUsConectadas: make(map[int]*CPU),
		Configuracion:  new(ConfigKernel),
		DispositivosIO: make(map[string]*InstanciasPorDispositivo),
	}

	kernel.InicializarMapaDeEstados()

	err := IniciarConfiguracion("config.json", kernel.Configuracion)

	if err != nil {
		fmt.Println("[main] Error al iniciar config")
		return
	}

	err = ConfigurarLogger("kernel", kernel.Configuracion.Log_leveL)
	if err != nil {
		fmt.Println("[main] Error al configurar logger:", err)
		os.Exit(1)
	}

	if len(os.Args) < 3 { // ruta archivo-pseudo tamanio
		slog.Error("[main] Cantidad de argumentos incorrecta. Uso: ruta <archivo-pseudo> <tamanio>")
		os.Exit(1)
	}
	args := os.Args

	archivoPseudo := args[1]
	tamanio, _ := strconv.Atoi(args[2])

	slog.Debug("Parámetros iniciales", "archivo", archivoPseudo, "tamanio", tamanio)

	//Iniciar servidor
	mux := http.NewServeMux()

	mux.HandleFunc("/cpu/registrar_cpu", kernel.llegaNuevaCPU) //SINCRO HECHA
	mux.HandleFunc("/kernel/interrupido", kernel.llegaFinInterrupcion)
	mux.HandleFunc("/cpu/syscall", kernel.llegaSyscallCPU)
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

	waitEnter := make(chan struct{}, 1)

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

	kernel.IntentarEnviarProcesoAExecute()

	select {}
}
