package main

import (
	"sync"
	"time"
)

type ConfigKernel struct {
	Ip_memoria              string  `json:"ip_memory"`
	Puerto_Memoria          int     `json:"port_memory"`
	Algoritmo_Plani         string  `json:"scheduler_algorithm"`
	Ready_ingress_algorithm string  `json:"ready_ingress_algorithm"`
	Alfa                    float64 `json:"alpha"`
	Tiempo_Suspension       int     `json:"suspension_time"`
	Log_leveL               string  `json:"log_level"`
	Puerto_Kernel           int     `json:"port_kernel"`
	Ip_kernel               string  `json:"ip_kernel"`
}

const cantEstados int = 7

type PCB struct {
	Pid      int
	Pc       int
	Me       [cantEstados]int           //Metricas de Estado
	Mt       [cantEstados]time.Duration //Metricas de Tiempo
	tamanio  int                        //revisar a futuro
	contador time.Time                  //revisar a futuro
	estado   int
}

type CPU struct {
	ID         int
	Url        string
	Pid int 
	Esta_libre bool
}

type IO struct {
	Urls           []string
	CantInstancias int
}

// Ver si volarlo
// type HandshakeRequest struct {
// 	NombreCPU string
// 	PuertoCPU int
// 	IpCPU     string
// }

var (
	cpuLibres         = make(map[int]CPU)
	IOs               = make(map[string]*IO)
	l_block           = make(map[string][]*PCB)
	l_execute         []*PCB
	ioMutex           sync.RWMutex
	mutex_cpus_libres sync.Mutex
	mutex_ios         sync.Mutex
)

type PorTamanio []PCB

// Metodos para usar sort(ordenamiento ascendente)
func (pcb PorTamanio) Swap(i int, j int) { pcb[i], pcb[j] = pcb[j], pcb[i] }

func (pcb PorTamanio) Len() int { return len(pcb) }

func (pcb PorTamanio) Less(i int, j int) bool { return pcb[i].tamanio < pcb[j].tamanio }

// var estados = []string{"NEW", "READY", "EXECUTE", "BLOCK", "BLOCK-SUSPENDED", "BLOCK-READY", "EXIT"}
var config_kernel *ConfigKernel

type solicitudIniciarProceso struct {
	Pid           int    `json:"pid"`
	ArchivoPseudo string `json:"archivoPseudo"`
	Tamanio       int    `json:"tamanio"`
}

const (
	EstadoNew = iota
	EstadoReady
	EstadoExecute
	EstadoBlock
	EstadoBlockSuspended
	EstadoBlockReady
	EstadoExit
)
