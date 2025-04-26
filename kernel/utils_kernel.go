package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
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

const cantEstados int = 6

type Pcb struct {
	Pid int
	Pc  int
	Me  [cantEstados]int     //Metricas de Estado
	Mt  [cantEstados]float64 //Metricas de Tiempo

}

var config_kernel *ConfigKernel
var cola_new []*Pcb
var cola_ready []*Pcb
var cola_block []*Pcb

// var cola_susp_block []
// var cola_susp_ready []

// array de arrays que contenga a todas las colas

func iniciarConfiguracionKernel(filePath string) *ConfigKernel {
	var configuracion *ConfigKernel
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&configuracion)

	return configuracion
}

func detenerKernel() {

	fmt.Println("Ingrese ENTER para empezar con la planificacion")

	bufio.NewReader(os.Stdin).ReadBytes('\n')

	fmt.Println("Empezando con la planificacion")

}
func iniciarPlanificadorLP(tamanio string, pid *int) {

	ingresarColaNew(pid)

}

func ingresarColaNew(pid *int) {
	pcb := new(Pcb)
	pcb.Pid = *pid
	*pid++
	pcb.Pc = 0
	//inicio := time.Now()
	cola_new = append(cola_new, pcb)

}

func modificarEstado(pcb *Pcb, pos int) {

	pcb.Me[pos]++

}
