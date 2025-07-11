package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

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
	gb_frames_disponibles = cant_frames_totales

	// Inicializar frames con valores por defecto
	frames := make([]*Frame, cant_frames_totales)
	for i := 0; i < cant_frames_totales; i++ {
		frames[i] = &Frame{
			Usado:        false,
			PidOcupante:  -1,
			NumeroPagina: -1,
		}
	}

	swapfile, err := os.OpenFile(config.Path_swap, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		slog.Error("error - abriendo el archivo de swap", err)
		return
	}
	defer swapfile.Close()

	memo := &Memo{
		memoria_sistema:   make(map[int][]string),
		memoria_principal: make([]byte, config.Tamanio_memoria),
		Procesos:          make(map[int]*Proceso),
		metricas:          make(map[int][]int),
		Frames:            frames,
		Config:            config,
		swap: &DataSwap{
			ultimo_byte:      0,
			espacio_contiguo: make(map[int]*ProcesoEnSwap),
			espacio_libre:    []*EspacioLibre{},
			SwapFile:         swapfile,
		},
	}

	utils.ConfigurarLogger("memoria", memo.Config.Log_level)

	mux := http.NewServeMux()
	// GENERAL
	mux.HandleFunc("/memoria/handshake", memo.Hanshake)

	// ======== APIs CPU  ========
	mux.HandleFunc("/memoria/fetch", memo.Fetch)                                  //----sincronizado
	mux.HandleFunc("/memoria/busqueda_tabla", memo.buscarEnTablaAsociadoAProceso) //----sincronizado
	mux.HandleFunc("/memoria/READ", memo.LeerEnMemoria)                           //----sincronizado
	mux.HandleFunc("/memoria/WRITE", memo.EscribirEnMemoria)                      //----sincronizado
	// mux.HandleFunc("/memoria/actualizar_entrada_cache", memo.ActualizarEntradaCache)
	// ===========
	// APIs Kernel
	// ===========
	mux.HandleFunc("/memoria/hay_lugar", memo.VerificarHayLugar)  //--------------------sincronizado
	mux.HandleFunc("/memoria/MEMORY_DUMP", memo.DumpMemory)       //--------------------sincronizado
	mux.HandleFunc("/memoria/EXIT_PROC", memo.FinalizarProceso)   //--------------------sincronizado
	mux.HandleFunc("/memoria/SUSPEND_PROC", memo.EscribirEnSwap)  //--------------------sincronizado
	mux.HandleFunc("/memoria/DE_SUSPEND_PROC", memo.QuitarDeSwap) //--------------------sincronizado

	url := fmt.Sprintf(":%d", memo.Config.Puerto_mem)

	slog.Debug("Iniciando servidor")
	go http.ListenAndServe(url, mux)

	select {}

}
