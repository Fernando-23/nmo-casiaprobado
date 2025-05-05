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
	Estimacion_Inicial      float64 `json:"initial_estimate"`
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
	SJF      *SJF //Estimaciones para planificacion SJF
}

type SJF struct {
	Estimado_anterior float64
	Real_anterior     float64
}

type CPU struct {
	ID         int
	Url        string
	Pid        int
	Pc         int
	Esta_libre bool
}

type IOs struct {
	IOs *IO
}

type IO struct {
	Url        string
	Pid        int
	Esta_libre bool
}

var (
	cpuLibres         = make(map[int]*CPU)
	ios               = make(map[string]*IOs)
	l_block           []*PCB
	l_susp_block      []*PCB
	l_susp_ready      []*PCB
	l_execute         []*PCB
	l_new             []*PCB
	l_ready           []*PCB
	ioMutex           sync.RWMutex
	mutex_cpus_libres sync.Mutex
	mutex_ios         sync.Mutex
	pid               int
)

// PROCESO MAS CHICO PRIMERO
type PorTamanio []*PCB

// Metodos para usar sort(ordenamiento ascendente por tamanio)
func (pcb PorTamanio) Swap(i, j int) { pcb[i], pcb[j] = pcb[j], pcb[i] }

func (pcb PorTamanio) Len() int { return len(pcb) }

func (pcb PorTamanio) Less(i, j int) bool { return pcb[i].tamanio < pcb[j].tamanio }

// SJF
type PorSJF []*PCB

// Metodos para usar sort(ordenamiento ascendente por SJF)
func (pcb PorSJF) Swap(i, j int) { pcb[i], pcb[j] = pcb[j], pcb[i] }

func (pcb PorSJF) Len() int { return len(pcb) }

func (pcb PorSJF) Less(i, j int) bool {
	return int(pcb[i].SJF.Estimado_anterior) < int(pcb[j].SJF.Estimado_anterior)
}

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
const (
	IdCPU = iota
	PC
	CodOp
)
