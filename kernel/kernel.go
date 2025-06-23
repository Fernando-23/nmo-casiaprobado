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
		cpusLibres:   make(map[int]*CPU),
		ConfigKernel: new(ConfigKernel),
		ios:          make(map[string]*IOS),
	}

	kernel.InicializarMapaDeEstados()

	err := IniciarConfiguracion("config.json", kernel.ConfigKernel)

	if err != nil {
		slog.Info("Error al iniciar config")
		return
	}

	if len(os.Args) < 3 { // ruta archivo-pseudo tamanio
		fmt.Println("Cantidad de argumentos incorrecta. Uso: ruta <archivo-pseudo> <tamanio>")
		os.Exit(1)
	}

	args := os.Args

	archivoPseudo := args[1]
	tamanio, _ := strconv.Atoi(args[2])

	fmt.Println("Archivo de pseudocodigo ", args[1])
	fmt.Println("Tamanio de proceso", args[2])

	//Iniciar servidor
	mux := http.NewServeMux()

	mux.HandleFunc("/cpu/registrar_cpu", kernel.registrarNuevaCPU)
	mux.HandleFunc("/cpu/syscall", kernel.RecibirSyscallCPU)
	mux.HandleFunc("/kernel/registrar_io", kernel.registrarNuevaIO) // revisado y corregido 20/6
	mux.HandleFunc("/kernel/desconectar_io", kernel.FinalizarIO)    // revisado y corregido 20/6
	mux.HandleFunc("/kernel/fin_io", kernel.RecibirFinIO)           // revisado y corregido 20/6
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)

	url_socket := fmt.Sprintf(":%d", kernel.ConfigKernel.Puerto_Kernel)

	go func() {
		if err := http.ListenAndServe(url_socket, mux); err != nil {
			fmt.Println("Error al iniciar el servidor HTTP", err)
			os.Exit(1)
		}
	}()

	waitEnter := make(chan struct{}, 1)

	// Lanzamos la rutina que espera el ENTER
	go esperarEnter(waitEnter)
	// Esperamos hasta que la gorutine avise que el ENTER fue presionado (queda bloqueada la main rutine)
	<-waitEnter

	fmt.Println("Empezando con la planificacion (se presionó el ENTER)")

	pcb := kernel.IniciarProceso(tamanio, archivoPseudo)
	fmt.Println("1er Proceso creado: ", pcb.Pid)
	kernel.AgregarAEstado(EstadoNew, pcb)

	unElemento, err := kernel.UnicoEnNewYNadaEnSuspReady()

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
