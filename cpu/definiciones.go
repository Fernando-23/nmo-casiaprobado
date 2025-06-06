package main

import (
	"sync"
	"time"
)

type ConfigCPU struct {
	Puerto_CPU      int    `json:"port_cpu"`
	Ip_CPU          string `json:"ip_cpu"`
	Ip_Memoria      string `json:"ip_memory"`
	Puerto_Memoria  int    `json:"port_memory"`
	Ip_Kernel       string `json:"ip_kernel"`
	Puerto_Kernel   int    `json:"port_kernel"`
	Entrada_TLB     int    `json:"tlb_entries"`
	Reemplazo_TLB   string `json:"tlb_replacement"`
	Entrada_Cache   int    `json:"cache_entries"`
	Reemplazo_Cache string `json:"cache_replacement"`
	Delay_Cache     int    `json:"cache_delay"`
	Log_level       string `json:"log_level"`
}

type DireccionFisica struct {
	frame  int
	offset int
}

type EntradaTLB struct {
	pagina    int
	frame     int
	timestamp time.Duration
}

type EntradaCachePag struct {
	pagina    int
	contenido string
}

var (
	config_CPU         *ConfigCPU
	tlb                []*EntradaTLB
	cache_pags         []*EntradaCachePag
	url_cpu            string
	url_kernel         string
	url_memo           string
	hay_interrupcion   bool
	tlb_activa         bool
	cache_pags_activa  bool
	id_cpu             string
	pid_ejecutando     *int
	pc_ejecutando      *int
	cant_niveles       int
	cant_entradas_tpag int
	tam_pag            int
	sem_datos_kernel   sync.Mutex
	hay_datos          string
)
