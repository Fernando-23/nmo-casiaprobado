package main

import (
	"sync"
	"time"
)

type ConfigCPU struct {
	Puerto_CPU          int    `json:"port_cpu"`
	Ip_CPU              string `json:"ip_cpu"`
	Ip_Memoria          string `json:"ip_memory"`
	Puerto_Memoria      int    `json:"port_memory"`
	Ip_Kernel           string `json:"ip_kernel"`
	Puerto_Kernel       int    `json:"port_kernel"`
	Cant_entradas_TLB   int    `json:"tlb_entries"`
	Alg_repl_TLB        string `json:"tlb_replacement"`
	Cant_entradas_cache int    `json:"cache_entries"`
	Alg_repl_cache      string `json:"cache_replacement"`
	Delay_Cache         int    `json:"cache_delay"`
	Log_level           string `json:"log_level"`
}

type DireccionFisica struct {
	frame  int
	offset int
}

type EntradaTLB struct {
	pagina             int
	frame              int
	last_recently_used time.Time
	tiempo_vida        time.Time
}

type EntradaCachePag struct {
	pagina         int
	frame          int
	offset         int
	contenido      string
	bit_uso        int
	bit_modificado int
}

var (
	config_CPU              *ConfigCPU
	tlb                     []*EntradaTLB
	noticiero_metereologico time.Time
	cache_pags              []*EntradaCachePag
	//url_cpu                 string
	url_kernel        string
	url_memo          string
	hay_interrupcion  bool
	tlb_activa        bool
	cache_pags_activa bool

	pid_ejecutando *int
	pc_ejecutando  *int

	//No hace falta sincronizar
	id_cpu             string
	cant_niveles       int
	cant_entradas_tpag int
	tam_pag            int
)

// Mutexs
// a fer le gustan las bariavlhez glovaldez
var (
	ch_esperar_datos       chan struct{}
	mutex_hay_interrupcion sync.Mutex
)
