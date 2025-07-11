package main

import (
	"os"
	"sync"
)

type ConfigMemo struct {
	Puerto_mem       int    `json:"port_memory"`
	Ip_memoria       string `json:"ip_memory"`
	Tamanio_memoria  int    `json:"memory_size"`
	Tamanio_pag      int    `json:"page_size"`
	EntradasPorNivel int    `json:"entries_per_page"`
	Cant_niveles     int    `json:"number_of_levels"`
	Delay_memoria    int    `json:"memory_delay"`
	Path_swap        string `json:"swapfile_path"`
	Delay_swap       int    `json:"swap_delay"`
	Log_level        string `json:"log_level"`
	Path_dump        string `json:"dump_path"`
}

type Memo struct {
	memoria_sistema   map[int][]string //mapeo de arch pseudo
	memoria_principal []byte
	Config            *ConfigMemo
	swap              *DataSwap
	Procesos          map[int]*Proceso
	Frames            []*Frame //sincronizacion
	metricas          map[int][]int
}

type Frame struct {
	Usado        bool
	PidOcupante  int
	NumeroPagina int //dentro del proceso ocupante
}

type Proceso struct {
	Pid           int
	TablaPagsRaiz *TablaDePaginas
	Tamanio       int
}

type TablaDePaginas struct {
	Entradas []*EntradaTablaDePaginas
}

type EntradaTablaDePaginas struct {
	SiguienteNivel *TablaDePaginas // si no es ultimo nivel
	NumeroDeFrame  *int            // si es ultimo nivel
}

type DataSwap struct {
	ultimo_byte      int
	espacio_contiguo map[int]*ProcesoEnSwap
	espacio_libre    []*EspacioLibre
	SwapFile         *os.File // swap real en disco
}

type ProcesoEnSwap struct {
	inicio  int
	tamanio int
}

type EspacioLibre struct {
	inicio  int
	tamanio int
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
)

var (
	mutex_tamanioMemoActual sync.Mutex
	mutex_framesDisponibles sync.Mutex
	mutex_memoriaPrincipal  sync.Mutex
	mutex_memoriaSistema    sync.Mutex
	mutex_metricas          sync.Mutex
	mutex_lprocs            sync.Mutex
	mutex_swap              sync.Mutex
)

const cant_metricas = 6
