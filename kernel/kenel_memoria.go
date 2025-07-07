package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (k *Kernel) HandshakeMemoria() error {
	url := fmt.Sprintf("http://%s:%d/memoria/handshake", k.Configuracion.Ip_memoria, k.Configuracion.Puerto_Memoria)
	respuesta, err := utils.EnviarSolicitudHTTPString("GET", url, nil)
	if err != nil {
		return fmt.Errorf("memoria no responde: %w", err)
	}

	if respuesta != "OK" {
		return fmt.Errorf("respuesta inesperada de memoria: %s", respuesta)
	}

	return nil
}

func (k *Kernel) MemoHayEspacio(pid int, tamanio int, archivoPseudo string) (bool, error) {

	solicitud_memoria := fmt.Sprintf("%d %d %s", pid, tamanio, archivoPseudo)
	url := fmt.Sprintf("http://%s:%d/memoria/hay_lugar", k.Configuracion.Ip_memoria, k.Configuracion.Puerto_Memoria)
	respuesta, err := utils.EnviarSolicitudHTTPString("POST", url, solicitud_memoria)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return false, err
	}

	if respuesta == "Si kernel, hay espacio" {
		fmt.Println("PRUEBA - efectivamente, habia espacio")
		return true, nil
	}
	return false, nil

}

func (k *Kernel) solicitudEliminarProceso(pid int) (string, error) {

	url := fmt.Sprintf("http://%s:%d/memoria/exit/", k.Configuracion.Ip_memoria, k.Configuracion.Puerto_Memoria)
	pid_string := strconv.Itoa(pid)
	respuestaMemo, err := utils.EnviarSolicitudHTTPString("POST", url, pid_string)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	//Deberia responder "OK"
	return respuestaMemo, err

}
