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

type CPU struct {
	Id              string
	Url_memoria     string
	Url_kernel      string
	Proc_ejecutando *Proceso
	Tlb             []*EntradaTLB
	Cache_pags      []*EntradaCachePag
	Config_CPU      *ConfigCPU
}

type Proceso struct {
	Pid int
	Pc  int
}

var (
	noticiero_metereologico time.Time

	hay_que_actualizar_contexto    bool
	tenemos_interrupt              bool
	tengo_que_actualizar_en_kernel bool
	tlb_activa                     bool
	cache_pags_activa              bool

	//No hace falta sincronizar
	cant_niveles       int
	cant_entradas_tpag int
	tam_pag            int

	//puntero para clock
	puntero int
)

// Mutexs
var (
	mutex_hayQueActualizarContexto   sync.Mutex
	mutex_tenemosInterrupt           sync.Mutex
	mutex_tengoQueActualizarEnKernel sync.Mutex
)

// //MIS MEJORES AMIGOS, LOS CHANNELS
var (
	ch_esperar_datos chan struct{}
)
