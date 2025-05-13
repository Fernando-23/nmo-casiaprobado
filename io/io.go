package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	args := os.Args
	nombre_io := args[1]

	fmt.Println("Iniciando I/O..")
	config_IO = iniciarConfiguracionIO("config.json")
	url_io = fmt.Sprintf("http://%d:%d/", config_IO.Ip_io, config_IO.Puerto_io)
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
