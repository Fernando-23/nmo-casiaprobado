package main

import (
	"encoding/json"
	"log"
	"os"
)

type ConfigKernel struct {
	Ip_memoria              string `json:"ip_memory"`
	Puerto_Memoria          int    `json:"port_memory"`
	Algoritmo_Plani         string `json:"scheduler_algorithm"`
	Ready_ingress_algorithm string `json:"ready_ingress_algorithm"`
	Alfa                    int    `json:"alpha"`
	Tiempo_Suspension       int    `json:"suspension_time"`
	Log_leveL               string `json:"log_level"`
	Puerto_Kernel           int    `json:"port_kernel"`
	Ip_kernel               string `json:"ip_kernel"`
}

var config_kernel *ConfigKernel

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
