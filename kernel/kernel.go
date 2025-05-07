package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
	servidor "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Servidor"
)

func main() {

	fmt.Printf("Iniciando Kernel...")
	cliente.ConfigurarLogger()
	kernel := &Kernel{
		procesoPorEstado: make(map[int][]*PCB),
		cpusLibres:       make(map[int]*CPU),
		ConfigKernel:     new(ConfigKernel),
		ios:              make(map[string]*IOS),
	}

	kernel.InicializarEstados()

	err := IniciarConfiguracion("config.json", kernel.ConfigKernel)

	if err != nil {
		slog.Info("Error al iniciar config")
		return
	}

	err = kernel.HandshakeMemoria()

	if err != nil {
		slog.Info("Memoria no activa")
		return
	}
	args := os.Args

	archivoPseudo := args[1]
	tamanio, _ := strconv.Atoi(args[2])

	fmt.Println("Archivo de pseudocodigo ", args[1])
	fmt.Println("Tamaño de proceso", args[2])

	pcb := kernel.IniciarProceso(tamanio, archivoPseudo)

	detenerKernel()

	kernel.BolicheMomento(pcb) //punchi punchi

	kernel.IntentarEnviarProcesoAExecute()

	//Iniciar servidor
	mux := http.NewServeMux()

	mux.HandleFunc("cpu/registrar_cpu", kernel.registrarNuevaCPU)
	mux.HandleFunc("cpu/syscall", kernel.RecibirSyscallCPU)
	mux.HandleFunc("io/registrar_io", kernel.registrarNuevaIO) // cambiar en io lo que envia
	mux.HandleFunc("io/recibir_respuesta", kernel.RecibirRespuestaIO)
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)

	slog.Debug("Iniciando Servidor de KERNEL")

	url_socket := fmt.Sprintf(":%d", kernel.ConfigKernel.Puerto_Kernel)
	socket_kernel := http.ListenAndServe(url_socket, mux)
	if socket_kernel != nil {
		panic("Error al iniciar el servidor")
	}

	cliente.EnviarMensaje(kernel.ConfigKernel.Ip_memoria, kernel.ConfigKernel.Puerto_Memoria, "Hola soy modulo Kernel")

	//2d0 proc
	//enviamos msg a memoria funcion (tamanio pid)

	// Detener kERNEL para esperar las conexiones de CPU y despues iniciar la PLANIFICACION

	//estados := [cantEstados]string{"NEW", "READY", "EXECUTE", "BLOCK", "BLOCK-SUSPENDED", "BLOCK-READY", "EXIT"}

	//sort.Sort(PorTamanio(l_new))

}
