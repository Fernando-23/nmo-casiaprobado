package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

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
