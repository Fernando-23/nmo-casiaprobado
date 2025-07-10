package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func main() {

	config := new(ConfigMemo)

	if err := utils.IniciarConfiguracion("memoria.json", config); err != nil {
		fmt.Println("Error cargando config memoria: ", err)
		return
	}

	cant_frames_totales := int(config.Tamanio_memoria / config.Tamanio_pag)
	gb_tam_memo_actual = config.Tamanio_memoria
	gb_tamanio_pag = config.Tamanio_pag

	memo := &Memo{
		memoria_sistema:   make(map[int][]string),
		memoria_principal: make([]byte, config.Tamanio_memoria),
		ptrs_raiz_tpag:    make(map[int]*NivelTPag),
		bitmap:            make([]int, cant_frames_totales),
		l_proc:            make(map[int]*Proceso),
		metricas:          make(map[int][]int),
		config_memo:       config,
		swap: &DataSwap{
			ultimo_byte:      0,
			espacio_contiguo: make(map[int]*ProcesoEnSwap),
			espacio_libre: []*EspacioLibre{
				{
					inicio:  0,
					tamanio: config.Tamanio_memoria,
				},
			},
		},
	}

	utils.ConfigurarLogger("memoria", memo.config_memo.Log_level)

	memo.InicializarTablaFramesGlobal(cant_frames_totales)

	mux := http.NewServeMux()
	// GENERAL
	mux.HandleFunc("/memoria/handshake", memo.Hanshake)

	// ======== APIs CPU  ========
	mux.HandleFunc("/memoria/fetch", memo.Fetch)                                  //---sincronizado
	mux.HandleFunc("/memoria/busqueda_tabla", memo.buscarEnTablaAsociadoAProceso) //---sincronizado
	mux.HandleFunc("/memoria/READ", memo.LeerEnMemoria)                           //---sincronizado
	mux.HandleFunc("/memoria/WRITE", memo.EscribirEnMemoria)                      //---sincronizado
	// mux.HandleFunc("/memoria/actualizar_entrada_cache", memo.ActualizarEntradaCache)
	// ===========
	// APIs Kernel
	// ===========
	mux.HandleFunc("/memoria/hay_lugar", memo.VerificarHayLugar)  //--------------------sincronizado
	mux.HandleFunc("/memoria/MEMORY_DUMP", memo.DumpMemory)       //--------------------sincronizado
	mux.HandleFunc("/memoria/EXIT_PROC", memo.FinalizarProceso)   //--------------------sincronizado
	mux.HandleFunc("/memoria/SUSPEND_PROC", memo.EscribirEnSwap)  //--------------------sincronizado
	mux.HandleFunc("/memoria/DE_SUSPEND_PROC", memo.QuitarDeSwap) //--------------------sincronizado

	url := fmt.Sprintf(":%d", memo.config_memo.Puerto_mem)

	slog.Debug("Iniciando servidor")
	go http.ListenAndServe(url, mux)

	select {}

}
