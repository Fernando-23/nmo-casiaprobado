package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func RegistrarIO(nombre string) {

	fullURL := fmt.Sprintf("http://%s:%d/kernel/registrar_io", config_IO.Ip_kernel, config_IO.Puerto_kernel)
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

	// mandarle el pid

	AvisarFinIO(pid_recibido)

}

func AvisarFinIO(pid string) {
	fullURL := fmt.Sprintf("http://%s:%d/kernel/fin_io", config_IO.Ip_kernel, config_IO.Puerto_kernel)
	fullData := fmt.Sprintf("%s %s", pid, nombre_io)
	utils.LoggerConFormato("## PID: %s - Fin de IO", pid)
	utils.EnviarSolicitudHTTPString("POST", fullURL, fullData)
}

func AvisarDesconexionIO() { //gracias que te aviso pa
	fullURL := fmt.Sprintf("http://%s:%d/kernel/desconectar_io", config_IO.Ip_kernel, config_IO.Puerto_kernel)
	utils.EnviarSolicitudHTTPString("POST", fullURL, url_io)
}
