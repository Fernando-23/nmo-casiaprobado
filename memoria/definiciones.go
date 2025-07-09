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
	memoria_sistema map[int][]string
	swap            *DataSwap
	l_proc          map[int]*Proceso
	ptrs_raiz_tpag  map[int]*NivelTPag
	tabla_frames    []int //sincronizacion
	metricas        map[int][]int
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
	lv_tabla   int
	entradas   []*int
	sgte_nivel *NivelTPag
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
	memoria_principal       []byte
	config_memo             *ConfigMemo
	frames_disponibles      int
	tam_memo_actual         int
	tamanio_pag             int
	mutex_memoria_principal sync.Mutex
)

const cant_metricas = 6
