package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func main() {
	args := os.Args
	nombre_io := args[1]

	fmt.Println("Iniciando I/O..")
	utils.IniciarConfiguracion("config.json", config_IO)
	RegistrarIO(nombre_io)

	//Iniciando servidor para peticiones I/O - Kernel
	mux := http.NewServeMux()
	mux.HandleFunc("/io/hace_algo", AtenderPeticion)

	slog.Debug("Iniciando servidor para peticiones I/O - Kernel")

	socket := fmt.Sprintf(":%d", config_IO.Puerto_io)
	api_peticiones_kernel := http.ListenAndServe(socket, mux)
	if api_peticiones_kernel != nil {
		panic("Error al iniciar el servidor")
	}

}
