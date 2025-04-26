package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
	servidor "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Servidor"
)

func main() {

	args := os.Args

	fmt.Println("archivo de pseudocodigo ", args[1])
	fmt.Println("tamaño de proceso", args[2])

	fmt.Printf("Iniciando Kernel...")

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
	pid := 0

	mux.HandleFunc("/paquetes", servidor.RecibirPaquetes)
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)

	slog.Debug("Iniciando Servidor de KERNEL")

	url_socket := fmt.Sprintf(":%d", config_kernel.Puerto_Kernel)
	socket_kernel := http.ListenAndServe(url_socket, mux)
	if socket_kernel != nil {
		panic("Error al iniciar el servidor")
	}

	cliente.EnviarMensaje(config_kernel.Ip_memoria, config_kernel.Puerto_Memoria, "Hola soy modulo Kernel")

	// Detener kERNEL para esperar las conexiones de CPU y despues iniciar la PLANIFICACION

	detenerKernel()

	iniciarPlanificadorLP(args[2], &pid)

}
