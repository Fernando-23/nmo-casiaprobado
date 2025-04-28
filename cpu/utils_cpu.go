package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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

type PidRequest struct {
	pid string `json:"pid"`
}

type WriteRequest struct {
	direccion int    `json:"direccion"`
	datos     string `json:"datos"`
}

type ReadRequest struct {
	direccion int `json:"direccion"`
	tamanio   int `json:"tamanio"`
}

type datosCicloResponse struct {
	pid int
	pc  int
}

var config_CPU *ConfigCPU

func iniciarConfiguracionIO(filePath string) *ConfigCPU {
	var configuracion *ConfigCPU
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&configuracion)

	return configuracion
}

func fetch(PC *int, PID *int) string {

	instruccion := memoria.enviarInstruccion(*PID, *PC)

	return instruccion
}

func decode(instruccion string) (string, []string) {
	l_instruccion := strings.Split(instruccion, " ")
	cod_op := l_instruccion[0]
	operacion := l_instruccion[1:]

	return cod_op, operacion
}

func execute(cod_op string, operacion []string, PC *int) {

	switch cod_op {
	case "NOOP":
	//consume el tiempo de ciclo de instruccion
	case "WRITE":
		direccion, _ := strconv.Atoi(operacion[0])
		datos := operacion[1]

		requestWRITE(direccion, datos, config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria)

	case "READ":
		direccion, _ := strconv.Atoi(operacion[0])
		tamanio, _ := strconv.Atoi(operacion[1])

		requestREAD(direccion, tamanio, config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria)
		slog.info

	case "GOTO":
		valor, _ := strconv.Atoi(operacion[0])
		*PC = valor

	//case "IO": enviar todo el paquete de IO o separado??

	default:
		fmt.Println("Error, ingrese una instruccion valida")
	}

	if cod_op != "GOTO" {
		*PC++
	}

}

func checkInterrupt(PC *int, PID *int) bool {
	var interrupcion bool = kernel.hayInterrupcion()
	if interrupcion {
		kernel.recibirInterrupcion(PC, PID)
	}
	return interrupcion
}

// type PidRequest struct {
// 		pid string `json:"pid"`
// }

func pedirDatosCiclo(ip string, puerto int) (int, int, error) {
	peticion := PidRequest{pid: "pid"}

	url := fmt.Sprintf("http://%s:%d/", ip, puerto)
	body, err := json.Marshal(peticion)
	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return -1, -1, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error pidiendo datos de ciclo de inst. a ip:%s puerto:%d", ip, puerto)
		return -1, -1, err
	}

	var response datosCicloResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Error decodificando mensaje: %s", err.Error())
		return -1, -1, err
	}

	return response.pid, response.pc, nil
}

func requestWRITE(direccion int, datos string, ip string, puerto int) (string, error) {
	peticion_WRITE := WriteRequest{
		direccion: direccion,
		datos:     datos,
	}

	url := fmt.Sprintf("http://%s:%d/", ip, puerto)
	body, err := json.Marshal(peticion_WRITE)
	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error enviando peticion WRITE")
		return "", err
	}

	log.Printf("Se esta intentando escribir %s en la direccion %d", datos, direccion)

	var response string
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Error decodificando mensaje: %s", err.Error())
		return "", err
	}

	return response, nil
}

func requestREAD(direccion int, tamanio int, ip string, puerto int) (string, error) {
	peticion_READ := ReadRequest{
		direccion: direccion,
		tamanio:   tamanio,
	}
	url := fmt.Sprintf("http://%s:%d/", ip, puerto)
	body, err := json.Marshal(peticion_READ)
	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error enviando peticion WRITE")
		return "", err
	}

	log.Printf("Se esta intentando leer en la direccion %d", direccion)

	var response string
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Error decodificando mensaje: %s", err.Error())
		return "", err
	}

	return response, nil
}

// func prepararDatosCicloQueryPath(ip string, puerto int, pid int){
// 	url := fmt.Sprintf("http://%s:%d/%d", ip, puerto,pid)

// }
