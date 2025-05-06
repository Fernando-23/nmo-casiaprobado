package main

import (
	"fmt"
	"log"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func requestWRITE(direccion int, datos string) (string, error) {

	peticion_WRITE := fmt.Sprintf("%d %s", direccion, datos)
	fullUrl := fmt.Sprintf("http://%s/memoria/WRITE", url_memo)

	log.Printf("Se esta intentando escribir %s en la direccion %d", datos, direccion)
	respuesta, err := utils.EnviarSolicitudHTTP("POST", fullUrl, peticion_WRITE)

	return respuesta, err
}

func requestREAD(direccion int, tamanio int) (string, error) {

	peticion_READ := fmt.Sprintf("%d %d", direccion, tamanio)
	fullUrl := fmt.Sprintf("http://%s/memoria/READ", url_memo)

	respuesta, err := utils.EnviarSolicitudHTTP("POST", fullUrl, peticion_READ)

	log.Printf("Se esta intentando leer en la direccion %d", direccion)

	return respuesta, err
}
