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

	hay_proceso_io = true
	duracion_en_IO = float64(tiempo_recibido)
	tiempo_en_IO = time.Now()

	select {
	case <-time.After(time.Duration(tiempo_recibido) * time.Millisecond):
		utils.LoggerConFormato("Termino correctamente tiempo en io en AtenderPeticion")

		hay_proceso_io = false
		AvisarFinIO(pid_recibido)
	case <-ch_cancelar_IO:
		utils.LoggerConFormato("IO desconectada en medio de ejecucion de AtenderPeticion")
	}

}

func AvisarFinIO(pid string) {
	fullURL := fmt.Sprintf("http://%s:%d/kernel/fin_io", config_IO.Ip_kernel, config_IO.Puerto_kernel)
	fullData := fmt.Sprintf("%s %s", pid, nombre_io)
	utils.LoggerConFormato("## PID: %s - Fin de IO", pid)
	utils.EnviarSolicitudHTTPString("POST", fullURL, fullData)
}

func AvisarDesconexionIO() { //gracias que te aviso pa

	if hay_proceso_io {
		transcurrido := float64(time.Since(tiempo_en_IO).Milliseconds())
		duracion_en_IO -= transcurrido
	}

	fullURL := fmt.Sprintf("http://%s:%d/kernel/desconectar_io", config_IO.Ip_kernel, config_IO.Puerto_kernel)
	peticion := fmt.Sprintf("%s %s %f", nombre_io, url_io, duracion_en_IO)
	utils.LoggerConFormato("## Avisando desconexion IO: %s", peticion)
	utils.EnviarSolicitudHTTPString("POST", fullURL, peticion)
}
