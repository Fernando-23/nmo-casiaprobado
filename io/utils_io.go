package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type ConfigIO struct {
	Ip_kernel     string `json:"ip_kernel"`
	Puerto_kernel int    `json:"port_kernel"`
	Puerto_io     int    `json:"port_io"`
	Ip_io         string `json:"ip_io"`
	Log_level     string `json:"log_level"`
}

type HandshakeRequest struct {
	NombreIO string `json:"nombre_io"`
	PuertoIO int    `json:"puerto_io`
	IpIO     string `json:"ip_io"`
}

type Peticion struct {
	Pid    int `json:"pid"`
	Tiempo int `json:"tiempo"`
}

var config_IO *ConfigIO

func iniciarConfiguracionIO(filePath string) *ConfigIO {
	var configuracion *ConfigIO
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&configuracion)

	return configuracion
}

func handshakeKernel(nombre string) {
	peticion := HandshakeRequest{
		NombreIO: nombre,
		PuertoIO: (*config_IO).Puerto_io,
		IpIO:     (*config_IO).Ip_io,
	}

	url := fmt.Sprintf("http://%s:%d/", (*config_IO).Ip_io, (*config_IO).Puerto_kernel)
	body, err := json.Marshal(peticion)
	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Println("Error enviando datos", resp)
		return
	}

	log.Println("Hanshake realizado correctamente!")
}

func atenderPeticion(w http.ResponseWriter, r *http.Request) {
	var peticion_recibida Peticion
	err := json.NewDecoder(r.Body).Decode(&peticion_recibida)

	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	time.Sleep(time.Duration(peticion_recibida.Tiempo) * time.Millisecond )
	w.WriteHeader(http.StatusOK)
	finalizoIO := fmt.Sprintf("%d",peticion_recibida.Pid)
	w.Write([]byte(finalizoIO))

}
