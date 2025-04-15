package main

import (
	"log/slog"
	"net/http"

	servidor "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Servidor"
)

func main() {

	//var configuracion *Config = cliente.iniciarConfiguracion("config.json")

	mux := http.NewServeMux()
	mux.HandleFunc("/paquetes", servidor.RecibirPaquetes)
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)

	slog.Debug("Iniciando servidor")
	err := http.ListenAndServe(":8002", mux)
	if err != nil {
		panic("Error al iniciar el servidor")
	}
}
