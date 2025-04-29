package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
)

func main() {
	args := os.Args
	nombre_io := args[1]

	fmt.Println("Iniciando I/O..")
	config_IO = iniciarConfiguracionIO("config.json")

	handshakeKernel(nombre_io)

	//Iniciando servidor para peticiones I/O - Kernel
	mux := http.NewServeMux()
	mux.HandleFunc("/recibir_peticion", atenderPeticion)

	slog.Debug("Iniciando servidor para peticiones I/O - Kernel")

	socket:= fmt.Sprintf(":%d", config_IO.Puerto_io)
	api_peticiones_kernel := http.ListenAndServe(socket, mux)
	if api_peticiones_kernel != nil {
		panic("Error al iniciar el servidor")
	}

	cliente.EnviarMensaje(config_IO.Ip_kernel, config_IO.Puerto_kernel, "Hola soy modulo IO")

}

