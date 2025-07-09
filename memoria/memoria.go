package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func main() {

	config_memo = &ConfigMemo{}
	utils.IniciarConfiguracion("memoria.json", config_memo)
	utils.ConfigurarLogger("memoria", config_memo.Log_level)
	cant_frames_totales := int(config_memo.Tamanio_memoria / config_memo.Tamanio_pag)

	memo := &Memo{
		memoria_sistema: make(map[int][]string),
		ptrs_raiz_tpag:  make(map[int]*NivelTPag),
		tabla_frames:    make([]int, cant_frames_totales),
		swap: &DataSwap{
			espacio_contiguo: make(map[int]*ProcesoEnSwap),
			espacio_libre:    []*EspacioLibre{},
		},
		metricas: make(map[int][]int),
	}

	tam_memo_actual = config_memo.Tamanio_memoria
	tamanio_pag = config_memo.Tamanio_pag
	memoria_principal = make([]byte, config_memo.Tamanio_memoria)
	fmt.Println(cant_frames_totales)
	memo.InicializarTablaFramesGlobal(cant_frames_totales)

	mux := http.NewServeMux()
	// GENERAL
	mux.HandleFunc("/memoria/handshake", Hanshake)
	// ========
	// APIs CPU
	// ========
	mux.HandleFunc("/memoria/fetch", memo.Fetch)
	mux.HandleFunc("/memoria/busqueda_tabla", memo.buscarEnTablaAsociadoAProceso)
	mux.HandleFunc("/memoria/READ", memo.LeerEnMemoria)
	mux.HandleFunc("/memoria/WRITE", memo.EscribirEnMemoria)
	// mux.HandleFunc("/memoria/actualizar_entrada_cache", memo.ActualizarEntradaCache)
	// ===========
	// APIs Kernel
	// ===========
	mux.HandleFunc("/memoria/hay_lugar", memo.VerificarHayLugar)
	mux.HandleFunc("/memoria/MEMORY_DUMP", memo.DumpMemory)
	mux.HandleFunc("/memoria/EXIT_PROC", memo.FinalizarProceso)
	mux.HandleFunc("/memoria/SUSPEND_PROC", memo.EscribirEnSwap)
	mux.HandleFunc("/memoria/DE_SUSPEND_PROC", memo.QuitarDeSwap)

	url := fmt.Sprintf(":%d", config_memo.Puerto_mem)

	slog.Debug("Iniciando servidor")
	go http.ListenAndServe(url, mux)

	select {}

}
