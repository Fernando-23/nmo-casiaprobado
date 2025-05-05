package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
	servidor "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Servidor"
)

func main() {

	args := os.Args

	archivoPseudo := args[1]
	tamanio, _ := strconv.Atoi(args[2])

	fmt.Println("archivo de pseudocodigo ", args[1])
	fmt.Println("tamaño de proceso", args[2])

	fmt.Printf("Iniciando Kernel...")
	cliente.ConfigurarLogger()
	config_kernel = iniciarConfiguracionKernel("config.json")
	url := fmt.Sprintf("http://%s:%d/", config_kernel.Ip_kernel, config_kernel.Puerto_Memoria)

	//Conexion con Memoria
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Info("Error al conectarse con memoria", req)
	}

	inicio := time.Now()
	duracion := time.Since(inicio)

	fmt.Println("tiempo: ", duracion)

	//Iniciar servidor
	mux := http.NewServeMux()

	mux.HandleFunc("cpu/nuevaCPU", conectarNuevaCPU)
	mux.HandleFunc("cpu/syscall", recibirSyscallCPU)
	mux.HandleFunc("cpu/interrupt", interruptHandler)
	mux.HandleFunc("io/nuevaIO", conectarNuevaIO) // cambiar en io lo que envia
	mux.HandleFunc("io/bloquearPorIo", manejarIO)
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)

	// Objetivos a hacer
	// mux.HandleFunc("/registrar_cpu", conectarNuevaCPU)  --Hecho
	// mux.HandleFunc("/registrar_io", conectarNuevaIO)    --Hecho
	// mux.HandleFunc("/syscall",gestionarSyscallCPU)      Hay que hacer
	// mux.HandleFunc("/gestionar_io",gestionarIO)		   Hay que hacer

	slog.Debug("Iniciando Servidor de KERNEL")

	url_socket := fmt.Sprintf(":%d", config_kernel.Puerto_Kernel)
	socket_kernel := http.ListenAndServe(url_socket, mux)
	if socket_kernel != nil {
		panic("Error al iniciar el servidor")
	}

	cliente.EnviarMensaje(config_kernel.Ip_memoria, config_kernel.Puerto_Memoria, "Hola soy modulo Kernel")

	//2d0 proc
	//enviamos msg a memoria funcion (tamanio pid)

	// Detener kERNEL para esperar las conexiones de CPU y despues iniciar la PLANIFICACION

	detenerKernel()

	//estados := [cantEstados]string{"NEW", "READY", "EXECUTE", "BLOCK", "BLOCK-SUSPENDED", "BLOCK-READY", "EXIT"}

	planiLargoPlazo(&pid, tamanio, archivoPseudo, &l_new, &l_ready)

	//sort.Sort(PorTamanio(l_new))

}
