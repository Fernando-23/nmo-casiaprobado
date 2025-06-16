package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
	servidor "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Servidor"
)

func main() {

	config_memo = &ConfigMemo{}
	utils.IniciarConfiguracion("config.json", config_memo)
	cant_frames_totales := int(config_memo.Tamanio_memoria / config_memo.Tamanio_pag)
	memo := &Memo{
		memoria_sistema: make(map[int][]string),
		ptrs_raiz_tpag:  make(map[int]*NivelTPag),
		tabla_frames:    make([]int, cant_frames_totales),
	}

	tam_memo_actual = config_memo.Tamanio_memoria
	tamanio_pag = config_memo.Tamanio_pag
	memoria_principal = make([]byte, config_memo.Tamanio_memoria)
	memo.InicializarTablaFramesGlobal(cant_frames_totales)

	mux := http.NewServeMux()
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)
	mux.HandleFunc("/memoria/handshake", Hanshake)
	mux.HandleFunc("/memoria/hay_lugar", memo.VerificarHayLugar)
	mux.HandleFunc("/memoria/fetch", memo.Fetch)
	mux.HandleFunc("/memoria/busqueda_tabla", memo.buscarEnTablaAsociadoAProceso)
	mux.HandleFunc("/memoria/READ", memo.LeerEnMemoria)
	mux.HandleFunc("/memoria/WRITE", memo.EscribirEnMemoria)
	mux.HandleFunc("/memoria/MEMORY_DUMP", memo.DumpMemory)
	// mux.HandleFunc("/memoria/crear_proceso", CrearProceso)

	url := fmt.Sprintf(":%d", config_memo.Puerto_mem)

	slog.Debug("Iniciando servidor")
	err := http.ListenAndServe(url, mux)
	if err != nil {
		panic("Error al iniciar el servidor")
	}

}
