package main

import "sync"

type ConfigMemo struct {
	Puerto_mem        int    `json:"port_memory"`
	Ip_memoria        string `json:"ip_memory"`
	Tamanio_memoria   int    `json:"memory_size"`
	Tamanio_pag       int    `json:"page_size"`
	Cant_entradasXpag int    `json:"entries_per_page"`
	Cant_niveles      int    `json:"number_of_levels"`
	Delay_memoria     int    `json:"memory_delay"`
	Path_swap         string `json:"swapfile_path"`
	Delay_swap        int    `json:"swap_delay"`
	Log_level         string `json:"log_level"`
	Path_dump         string `json:"dump_path"`
}

type Memo struct {
	memoria_sistema   map[int][]string //mapeo de arch pseudo
	memoria_principal []byte
	config_memo       *ConfigMemo

	swap           *DataSwap
	l_proc         map[int]*Proceso
	ptrs_raiz_tpag map[int]*NivelTPag
	bitmap         []int //sincronizacion
	metricas       map[int][]int
}

type Proceso struct {
	ptr_a_frames_asignados []*int //apunta a elementos de la tabla_frames
	tamanio                int
}

type DataSwap struct {
	ultimo_byte      int
	espacio_contiguo map[int]*ProcesoEnSwap
	espacio_libre    []*EspacioLibre
}

type ProcesoEnSwap struct {
	inicio  int
	tamanio int
}

type EspacioLibre struct {
	inicio  int
	tamanio int
}

type NivelTPag struct {
	lv_tabla     int
	sgte_nivel   *NivelTPag
	ultimo_nivel *UltimoNivelTPag
}

type UltimoNivelTPag struct {
	entradas []*int
}

const (
	Accesos_a_tpags = iota
	Cant_instr_solicitadas
	Bajadas_de_swap
	Subidas_a_memoria
	Cant_read
	Cant_write
)

var (
	gb_frames_disponibles int
	gb_tam_memo_actual    int
	gb_tamanio_pag        int
)

var (
	mutex_tamanioMemoActual sync.Mutex
	mutex_framesDisponibles sync.Mutex
	mutex_memoriaPrincipal  sync.Mutex
	mutex_memoriaSistema    sync.Mutex
	mutex_metricas          sync.Mutex
	mutex_tablaPaginas      sync.Mutex
	mutex_lprocs            sync.Mutex
	mutex_bitmap            sync.Mutex
	mutex_swap              sync.Mutex
)

const cant_metricas = 6
