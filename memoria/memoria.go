package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
	servidor "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Servidor"
)

func main() {

	//var configuracion *Config = cliente.iniciarConfiguracion("config.json")
	config_memo = &ConfigMemo{}
	memo := &Memo{
		memoria_sistema:  make(map[int][]string),
		global_ptrs_tpag: make(map[int][]*Tabla),
	}
	utils.IniciarConfiguracion("config.json", config_memo)

	memoria_usuario = make([]byte, config_memo.Tamanio_memoria)

	tam_memo_actual = config_memo.Tamanio_memoria

	mux := http.NewServeMux()
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)

	// APIs a hacer
	// mux.HandleFUnc("/memoria/WRITE",Escribir)
	mux.HandleFunc("/memoria/handshake", Hanshake)
	mux.HandleFunc("/memoria/hay_lugar", memo.VerificarHayLugar)
	mux.HandleFunc("/memoria/fetch", memo.Fetch)
	// mux.HandleFunc("/memoria/crear_proceso", CrearProceso)
	// mux.HandleFunc("/memoria/READ",LeerEnMemoria)

	url := fmt.Sprintf(":%d", config_memo.Puerto_mem)

	slog.Debug("Iniciando servidor")
	err := http.ListenAndServe(url, mux)
	if err != nil {
		panic("Error al iniciar el servidor")
	}

}
