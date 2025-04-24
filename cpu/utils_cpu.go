package main

import (
	"encoding/json"
	"fmt"
	"log"
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
		direccion := operacion[0]
		datos := operacion[1]

		memoria.escribirDatos(direccion, datos)

	case "READ":
		direccion := operacion[0]
		tamanio := operacion[1]

		memoria.leerDatos(direccion, tamanio)
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
