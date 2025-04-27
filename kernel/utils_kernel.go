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

type PCB struct {
	Pid int
	Pc  int
	Me  [cantEstados]int     //Metricas de Estado
	Mt  [cantEstados]float64 //Metricas de Tiempo
	tamanio int //revisar a futuro
}

var config_kernel *ConfigKernel


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
func iniciarPlanificadorLP(tamanio int, pid *int) {

	planificador()

}

func crearPcb(pid *int, tamanio int) pcb PCB{
	pcb := new(PCB)
	pcb.Pid = *pid
	pcb.tamanio = tamanio
	incrementarPid(pid)
    //pcb.Pc = 0
	return pcb
}



func incrementarPid(pid *int){
	*pid++
}

func FIFO(l_estado *PCB, pcb PCB){//FIFO
	l_estado = append(l_estado, pcb)
}



func planiCortoPlazo(l_ready *PCB, )

func planiLargoPlazo(l_new *PCB, l_ready *PCB, pcb PCB, algoritmoPlani string){//fijarte si podes hacer que entre a la cola de new y que prg dsp por el sig
	if/*hayespacio == true*/{
		if /*algoritmoPlani == "PMCP" || l_new == nil*/{
			pcb.Me[0]++ //lo podrias gacer en otra funcion y remplazar el 0 por una constante mas descriptiva
			//si entra anew sacar de new e insertar en ready bajo algoritmo 
			//meter a ready con algritmo correcto
		}
	}
	FIFO(l_new,PCB)
}





/*func ingresarColaNew(pid *int) {
	crearPcb(*pid)
	
	//inicio := time.Now()
	cola_new = append(cola_new, pcb)

}*/ //obsolteta aparentemente


func modificarEstado(pcb *PCB, pos int) {

	pcb.Me[pos]++

}
