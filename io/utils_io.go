package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

type ConfigIO struct {
	Ip_kernel     string `json:"ip_kernel"`
	Puerto_kernel int    `json:"port_kernel"`
	Puerto_io     int    `json:"port_io"`
	Ip_io         string `json:"ip_io"`
	Log_level     string `json:"log_level"`
}

var (
	config_IO *ConfigIO
	url_io    string
)

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

func RegistrarIO(nombre string) {

	fullURL := fmt.Sprintf("http://%s:%d/io/registrar_io", config_IO.Ip_kernel, config_IO.Puerto_kernel)
	registro := fmt.Sprintf("%s %s %d", nombre, config_IO.Ip_io, config_IO.Puerto_io)

	utils.EnviarSolicitudHTTPString("POST", fullURL, registro)

	log.Println("Hanshake realizado correctamente!")
	log.Println("Se registro la IO correctamente")

}

func AtenderPeticion(w http.ResponseWriter, r *http.Request) {
	var peticion_recibida string
	err := json.NewDecoder(r.Body).Decode(&peticion_recibida)
	if err != nil {
		log.Println("Error recibiendo datos")
		return
	}

	aux := strings.Split(peticion_recibida, " ")
	pid_recibido := aux[0]
	tiempo_recibido, _ := strconv.Atoi(aux[1])

	utils.LoggerConFormato("## PID: %s - Inicio de IO - Tiempo: %d", pid_recibido, tiempo_recibido)
	time.Sleep(time.Duration(tiempo_recibido) * time.Millisecond)
	finalizoIO := fmt.Sprintf("FIN_IO %s", pid_recibido)
	utils.LoggerConFormato("## PID: %s - Fin de IO", pid_recibido)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(finalizoIO))

}
