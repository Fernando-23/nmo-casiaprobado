package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
}

type PorTamanio []PCB

// Metodos para usar sort(ordenamiento ascendente)
func (pcb PorTamanio) Swap(i int, j int) { pcb[i], pcb[j] = pcb[j], pcb[i] }

func (pcb PorTamanio) Len() int { return len(pcb) }

func (pcb PorTamanio) Less(i int, j int) bool { return pcb[i].tamanio < pcb[j].tamanio }

// var estados = []string{"NEW", "READY", "EXECUTE", "BLOCK", "BLOCK-SUSPENDED", "BLOCK-READY", "EXIT"}
var config_kernel *ConfigKernel

type solicitudIniciarProceso struct {
	pid           int    `json:"pid"`
	archivoPseudo string `json:"archivoPseudo"`
	tamanio       int    `json:"tamanio"`
}

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

func crearPcb(pid *int, tamanio int) *PCB {
	pcb := new(PCB)
	pcb.Pid = *pid
	pcb.tamanio = tamanio
	incrementarPid(pid)
	//pcb.Pc = 0
	return pcb
}

//

func incrementarPid(pid *int) {
	*pid++
}

func FIFO(l_estado []*PCB, pcb *PCB) { //FIFO
	l_estado = append(l_estado, pcb)
}

func cambiarMetricasEstado(pPcb *PCB, posEstado int) {
	pPcb.Me[posEstado]++ //ver si puede quedar mas lindo
}

func cambiarMetricasTiempo(pPcb *PCB, posEstado int) {
	var tiempoAcutal time.Time
	tiempoAcutal = time.Now()
	(pPcb).Mt[posEstado] = tiempoAcutal.Sub((pPcb).contador)
}

func planiCortoPlazo(l_ready *PCB) {

}

//agregarACola(){
//
//}

func iniciarProceso(pid *int, tamanio int, l_new []*PCB) {
	pcb := crearPcb(pid, tamanio)
	pcb.contador = time.Now()
	const PosEstado int = 0 // estado NEW
	cambiarMetricasEstado(pcb, PosEstado)
	FIFO(l_new, pcb) //meter en la cola new no hay planificacion para meter en la cola new
}

func planiLargoPlazo(pid *int, tamanio int, archivoPseudo string, l_new []*PCB, l_ready []*PCB) error { //fijarte si podes hacer que entre a la cola de new y que prg dsp por el sig

	iniciarProceso(pid, tamanio, l_new)
	hay_espacio, err := hayEspacio(*pid, tamanio, archivoPseudo)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return err
	}
	if hay_espacio {
		if config_kernel.Ready_ingress_algorithm == "PMCP" || len(l_new) == 1 { //solo  el elemento que agregue,  meter a ready
			//lo podrias hacer en otra funcion y remplazar el 0 por una constante mas descriptiva
			pos_a_eliminar := len(l_new) - 1
			l_ready = append(l_ready, l_new[pos_a_eliminar])
			l_new = append(l_new[:pos_a_eliminar], l_new[pos_a_eliminar+1:]...)
		} //si entra anew sacar de new e insertar en ready bajo algoritmo
		//meter a ready con algritmo correcto
		//	}
	}
	return nil //dudoso
}

/*func ingresarColaNew(pid *int) {
	crearPcb(*pid)

	//inicio := time.Now()
	cola_new = append(cola_new, pcb)

}*/ //obsolteta aparentemente

func modificarEstado(pcb *PCB, pos int) {

	pcb.Me[pos]++

}

func solicitudMemo(pidAPreguntar int, tamanio int, archivoPseudo string) (string, error) {
	preguntaMemo := solicitudIniciarProceso{
		pid:           pidAPreguntar,
		archivoPseudo: archivoPseudo,
		tamanio:       tamanio,
	}
	url := fmt.Sprintf("http://%s:%d/", config_kernel.Ip_kernel, config_kernel.Puerto_Memoria)
	body, err := json.Marshal(preguntaMemo)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	respuesta, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	var respuestaMemo string
	err = json.NewDecoder(respuesta.Body).Decode(&respuestaMemo)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	return respuestaMemo, nil

}

func solicitudEliminarProceso(pid int) (string, error) {

	url := fmt.Sprintf("http://%s:%d/{%d}", config_kernel.Ip_kernel, config_kernel.Puerto_Memoria, pid)

	respuestaMemo, err := http.Get(url)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	bodyByte, err := io.ReadAll(respuestaMemo.Body)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	fmt.Println(string(bodyByte))

	return string(bodyByte), nil

}

func hayEspacio(pid int, tamanio int, archivoPseudo string) (bool, error) {

	mensaje, err := solicitudMemo(pid, tamanio, archivoPseudo)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return false, err
	}

	if mensaje == "okay" {
		return true, nil
	}
	return false, nil
	//mux.HandleFunc("/hay_espacio", hayEspacio())

}
