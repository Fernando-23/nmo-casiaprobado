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

func requestREAD(direccion_logica int, tamanio int) (string, DireccionFisica, error) {
	var dir_fisica DireccionFisica
	// busquedaMemoria(direccion_logica)

	nro_pagina := math.Floor(float64(direccion_logica) / float64(tam_pag))
	dir_fisica, err := traduccionMMU(direccion_logica, nro_pagina)

	if err != nil {
		return "", dir_fisica, err
	}

	peticion_READ := fmt.Sprintf("%d %d %d", *pid_ejecutando, dir_fisica.frame, dir_fisica.offset, tamanio)
	fullUrl := fmt.Sprintf("http://%s/memoria/READ", url_memo)

	respuesta, err := utils.EnviarSolicitudHTTPString("POST", fullUrl, peticion_READ)
	//---------------------------------------------- PID FRAME OFFSET TAMANIO

	log.Printf("Se esta intentando leer en la direccion fisica [ %d | %d ]", dir_fisica.frame, dir_fisica.offset)

	return respuesta, dir_fisica, err
}

func traduccionMMU(direccion_logica int, nro_pagina float64) (DireccionFisica, error) {

	desplazamiento := int(direccion_logica) % int(tam_pag)
	dir_fisica := DireccionFisica{
		frame:  -1,
		offset: -1,
	}

	// 1
	for nivel_actual := 1; nivel_actual <= int(cant_niveles); nivel_actual++ {
		//Santi: Lo mas hardcodeado que vi
		entrada_nivel_final := int(math.Floor(nro_pagina/math.Pow(float64(cant_entradas_tpag), float64((cant_niveles-nivel_actual))))) % int(cant_entradas_tpag)
		respuesta := busquedaTabla(*pid_ejecutando, nivel_actual, entrada_nivel_final)
		// -2 Direccionamiento invalido
		// -1 Todo bien, sigo al sgte nivel
		//>=0 Es un frame, lo devuelvo
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
	fullUrl := fmt.Sprintf("http://%s/memoria/busqueda_tabla", url_memo)
	aux, _ := utils.EnviarSolicitudHTTPString("POST", fullUrl, solicitud_acceso_tpaginas)
	respuesta, _ := strconv.Atoi(aux)
	return respuesta
}

func TLB(direccion_logica int, nro_pagina int) (int, error) {

	for i := 0; i <= config_CPU.Cant_entradas_TLB; i++ {
		if tlb[i].pagina == nro_pagina {
			// Caso TLB HIT
			utils.LoggerConFormato("PID: %d - TLB HIT - Pagina: %d", *pid_ejecutando, nro_pagina)
			return tlb[i].frame, nil
		}
	}
	// Caso TLB MISS

	dir_fisica, _ := traduccionMMU(direccion_logica, float64(nro_pagina))
	hayEspacioEnTLB, entrada := chequearEspacioEnTLB()

	if hayEspacioEnTLB {
		cambiarEstadoMarco(nro_pagina, dir_fisica.frame, entrada)
		return dir_fisica.frame, nil
	}

	aplicarDesalojo()
	return dir_fisica.frame, nil
}

func aplicarDesalojo() error {
	alg_reemplazo := config_CPU.Alg_repl_TLB
	switch alg_reemplazo {
	case "FIFO":
		aux := tlb[0]
		aux_entrada := 0
		for i := 1; i < config_CPU.Cant_entradas_TLB; i++ {
			if aux.timestamp_tiempo_vida < tlb[i].timestamp_tiempo_vida {
				aux = tlb[i]
				aux_entrada = i
			}
		}
		cambiarEstadoMarco(aux.pagina, aux.frame, aux_entrada)
		return nil
	case "LRU":
		aux := tlb[0]
		aux_entrada := 0
		for i := 1; i < config_CPU.Cant_entradas_TLB; i++ {
			if aux.timestamp_lru < tlb[i].timestamp_lru {
				aux = tlb[i]
				aux_entrada = i
			}
		}
		cambiarEstadoMarco(aux.pagina, aux.frame, aux_entrada)
		return nil
	default:
		return fmt.Errorf("y,la verda q me pasaste cualquier verdura en el algoritmo de TLB")
	}
}

func cachePags() {

}

func cambiarEstadoMarco(nro_pagina int, frame int, entrada_tlb int) {
	tlb[entrada_tlb].pagina = nro_pagina
	tlb[entrada_tlb].frame = frame
	utils.LoggerConFormato("Se realizo un cambio de marco en la TLB correctamente")
}

func chequearEspacioEnTLB() (bool, int) {
	for i := 0; i <= config_CPU.Cant_entradas_TLB; i++ {
		if tlb[i].pagina < 0 {
			return true, i
		}
	}
	return false, -1
}

func liberarTLB() {
	for i := 0; i <= config_CPU.Cant_entradas_TLB; i++ {
		tlb[i].pagina = -1
		tlb[i].frame = -1
	}
}

/*

func busquedaMemoria() int {
	frame := -1

	if cache_pags_activa {
		frame, err = cache_pags()
		if frame >= 0 {
			return frame
		}
	}

	if tlb_activa {
		frame, err = TLB()
		if frame >= 0 {
			return frame
		}
	}

	traduccionMMU()

}
*/
