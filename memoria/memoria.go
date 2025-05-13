package main

import (
	"fmt"
	"log/slog"
	"net/http"

	servidor "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Servidor"
)

func main() {

	//var configuracion *Config = cliente.iniciarConfiguracion("config.json")
	tam_memo_actual = config_memo.Tamanio_memoria
	config_memo = iniciarConfiguracionMemo("config.json")
	mux := http.NewServeMux()
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)
	mux.HandleFunc("/memoria/fetch", Fetch)

	// APIs a hacer
	// mux.HandleFunc("/memoria/hay_lugar", VerificarHayLugar)
	// mux.HandleFunc("/memoria/crear_proceso", CrearProceso)
	// mux.HandleFunc("/memoria/handshake", HanshakeKernel)
	// mux.HandleFUnc("/memoria/WRITE",Escribir)
	// mux.HandleFunc("/memoria/READ",LeerEnMemoria)

	url := fmt.Sprintf(":%d", config_memo.Puerto_mem)

	slog.Debug("Iniciando servidor")
	err := http.ListenAndServe(url, mux)
	if err != nil {
		panic("Error al iniciar el servidor")
	}

}
