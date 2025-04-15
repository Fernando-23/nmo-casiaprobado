package main

import (
	"fmt"
	"log/slog"
	"net/http"

	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
	servidor "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Servidor"
)

func main() {

	url := fmt.Sprintf("http://%s:%d/", "127.0.0.1", 8002)

	//Conexion con Memoria
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Info("Error al conectarse con memoria", req)
	}

	//Iniciar servidor
	mux := http.NewServeMux()

	mux.HandleFunc("/paquetes", servidor.RecibirPaquetes)
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)

	slog.Debug("Iniciando Servidor de KERNEL")
	socket_kernel := http.ListenAndServe(":8001", mux)
	if socket_kernel != nil {
		panic("Error al iniciar el servidor")
	}

	cliente.EnviarMensaje("127.0.0.1", 8002, "Hola soy modulo Kernel")
}
