package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
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

type WriteRequest struct {
	direccion int    `json:"direccion"`
	datos     string `json:"datos"`
}

type ReadRequest struct {
	direccion int `json:"direccion"`
	tamanio   int `json:"tamanio"`
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

func fetch(PID *int, PC *int) string {

	peticion := datosCiclo{
		Pid: *PID,
		Pc:  *PC,
	}

	url := fmt.Sprintf("http://%s:%d/", config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria)
	body, err := json.Marshal(peticion)
	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return ""
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("No se obtuvo algo valido de memoria durante la operacion fetch")
		return ""
	}

	var instruccion string
	err = json.NewDecoder(resp.Body).Decode(&instruccion)
	if err != nil {
		log.Printf("Error decodificando mensaje: %s", err.Error())
		return ""
	}

	return instruccion
}

func decode(instruccion string) (string, []string) {
	l_instruccion := strings.Split(instruccion, " ")
	cod_op := l_instruccion[0]
	operacion := l_instruccion[1:]

	return cod_op, operacion
}

func execute(cod_op string, operacion []string, PC *int, pid int) {

	pid_string := strconv.Itoa(pid)
	switch cod_op {

	case "NOOP":
		//consume el tiempo de ciclo de instruccion
		slog.Info("PID: %d - Ejecutando: %s", pid_string, cod_op)

	case "WRITE":
		direccion, _ := strconv.Atoi(operacion[0])
		datos := operacion[1]

		slog.Info("## PID: %s - Ejecutando: %s - %s - %s", pid_string, cod_op, operacion[0], datos)

		requestWRITE(direccion, datos, config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria)

		slog.Info("PID: " + pid_string + " - Acción: ESCRIBIR - Dirección Física: " + operacion[0] + " - Valor: " + datos)

	case "READ":
		direccion, _ := strconv.Atoi(operacion[0])
		tamanio, _ := strconv.Atoi(operacion[1])

		slog.Info("## PID: %s - Ejecutando: %s - %s - %s", pid_string, cod_op, operacion[0], operacion[1])

		//Gestionar mejor el error :p
		valor_leido, _ := requestREAD(direccion, tamanio, config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria)
		slog.Info("PID: " + pid_string + " - Acción: LEER - Dirección Física: " + operacion[0] + " - Valor: " + valor_leido)

	case "GOTO":
		slog.Info("PID: %s - Ejecutando: %s", pid_string, cod_op)

		valor, _ := strconv.Atoi(operacion[0])
		*PC = valor

	//Syscall
	case "IO":
		mensaje_io := "IO " + operacion[0]
		enviarSyscall(mensaje_io)
	case "INIT_PROC":
		mensaje_init_proc := "INIT_PROC " + operacion[0] + " " + operacion[1]
		enviarSyscall(mensaje_init_proc)

	case "DUMP_MEMORY":
		enviarSyscall("DUMP_MEMORY")

	case "EXIT":
		enviarSyscall("EXIT")

	default:
		fmt.Println("Error, ingrese una instruccion valida")
	}

	//Incrementar PC
	if cod_op != "GOTO" {
		*PC++
	}

}
