package main

import (
	"fmt"
	"log"
	"math"
	"strconv"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func requestWRITE(direccion int, datos string) (string, error) {

	peticion_WRITE := fmt.Sprintf("%d %s", direccion, datos)
	fullUrl := fmt.Sprintf("http://%s/memoria/WRITE", url_memo)

	log.Printf("Se esta intentando escribir %s en la direccion %d", datos, direccion)
	respuesta, err := utils.EnviarSolicitudHTTPString("POST", fullUrl, peticion_WRITE)

	return respuesta, err
}

// READ    200         50
//
//	DIR LOGICA   OFFSET
func requestREAD(direccion_logica int, tamanio int) (string, DireccionFisica, error) {

	// busquedaMemoria(direccion_logica)
	var dir_fisica DireccionFisica
	dir_fisica, err := traduccionMMU(direccion_logica)

	if err != nil {
		return "", dir_fisica, err
	}

	peticion_READ := fmt.Sprintf("%d %d %d", dir_fisica.frame, dir_fisica.offset, tamanio)
	fullUrl := fmt.Sprintf("http://%s/memoria/READ", url_memo)

	respuesta, err := utils.EnviarSolicitudHTTPString("POST", fullUrl, peticion_READ)

	log.Printf("Se esta intentando leer en la direccion fisica [ %d | %d ]", dir_fisica.frame, dir_fisica.offset)

	return respuesta, dir_fisica, err
}

func traduccionMMU(direccion_logica int, nro_pagina float64) (DireccionFisica, error) {
	//nro_pagina := math.Floor(float64(direccion_logica) / float64(tam_pag))
	desplazamiento := int(direccion_logica) % int(tam_pag)
	dir_fisica := DireccionFisica{
		frame:  -1,
		offset: -1,
	}

	for nivel_actual := 1; nivel_actual <= int(cant_niveles); nivel_actual++ {
		//Santi: Lo mas hardcodeado que vi
		entrada_nivel_X := int(math.Floor(nro_pagina/math.Pow(float64(cant_entradas_tpag), float64((cant_niveles-nivel_actual))))) % int(cant_entradas_tpag)

		respuesta := busquedaTabla(*pid_ejecutando, nivel_actual, entrada_nivel_X)
		if respuesta >= 0 {
			utils.LoggerConFormato("PID : %d - OBTENER MARCO - Página: %d - Marco: %d", *pid_ejecutando, int(nro_pagina), respuesta)
			dir_fisica.frame = respuesta
			dir_fisica.offset = desplazamiento
			return dir_fisica, nil

		} else if respuesta == -2 {
			break
		}

	}
	return dir_fisica, fmt.Errorf("direccionamiento invalido")
}

func busquedaTabla(pid int, nivel_actual int, entrada_a_acceder int) int {
	solicitud_acceso_tpaginas := fmt.Sprintf("%d %d %d", pid, nivel_actual, entrada_a_acceder)
	aux, _ := utils.EnviarSolicitudHTTPString("POST", url_memo, solicitud_acceso_tpaginas)
	respuesta, _ := strconv.Atoi(aux)
	return respuesta
}

func TLB(nro_pagina int, dir_logica int) {

}
