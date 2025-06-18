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
	cliente.ConfigurarLogger("kernel")
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

	/*err = kernel.HandshakeMemoria()

	if err != nil {
		slog.Info("Memoria no activa")
		return
	}*/
	args := os.Args

	archivoPseudo := args[1]
	tamanio, _ := strconv.Atoi(args[2])

	fmt.Println("Archivo de pseudocodigo ", args[1])
	fmt.Println("Tamanio de proceso", args[2])

	//Iniciar servidor
	mux := http.NewServeMux()

	mux.HandleFunc("/cpu/registrar_cpu", kernel.registrarNuevaCPU)
	mux.HandleFunc("/cpu/syscall", kernel.RecibirSyscallCPU)
	mux.HandleFunc("/io/registrar_io", kernel.registrarNuevaIO) // cambiar en io lo que envia
	mux.HandleFunc("io/recibir_respuesta", kernel.RecibirRespuestaIO)
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)

	url_socket := fmt.Sprintf(":%d", kernel.ConfigKernel.Puerto_Kernel)
	go http.ListenAndServe(url_socket, mux)
	/*if socket_kernel != nil {
		panic("Error al iniciar el servidor")
	}*/

	waitEnter := make(chan struct{})

	// Lanzamos la rutina que espera el ENTER
	go esperarEnter(waitEnter)

	fmt.Println("Empezando con la planificacion (se presionó el ENTER)")

	// Esperamos hasta que la gorutine avise que el ENTER fue presionado (queda bloqueada la main rutine)
	<-waitEnter

	pcb := kernel.IniciarProceso(tamanio, archivoPseudo)
	fmt.Println("1er Proceso creado: ", pcb.Pid)
	kernel.AgregarAEstado(EstadoNew, pcb)

	unElemento, err := kernel.ListaNewSoloYo()

	if err != nil || !unElemento {
		return
	}

	//kernel.BolicheMomento(pcb) //punchi punchi

	kernel.IntentarEnviarProcesoAExecute()

	slog.Debug("Iniciando Servidor de KERNEL")

	//cliente.EnviarMensaje(kernel.ConfigKernel.Ip_memoria, kernel.ConfigKernel.Puerto_Memoria, "Hola soy modulo Kernel")

	select {}
	//2d0 proc
	//enviamos msg a memoria funcion (tamanio pid)

	// Detener kERNEL para esperar las conexiones de CPU y despues iniciar la PLANIFICACION

	//estados := [cantEstados]string{"NEW", "READY", "EXECUTE", "BLOCK", "BLOCK-SUSPENDED", "BLOCK-READY", "EXIT"}

	//sort.Sort(PorTamanio(l_new))

}
