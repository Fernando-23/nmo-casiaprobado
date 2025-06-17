package main

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func requestWRITE(direccion_logica int, datos string) (string, DireccionFisica) {

	desplazamiento := int(direccion_logica) % int(tam_pag)
	nro_pagina := math.Floor(float64(direccion_logica) / float64(tam_pag))

	frame, contenido := busquedaMemoriaFrame(direccion_logica, int(nro_pagina), "WRITE") //el contenido es solo para cache pags activa

	dir_fisica := MMU(frame, desplazamiento)

	//Encuentro en cache pags
	if contenido != "NO_ENCONTRE" {
		utils.LoggerConFormato("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", *pid_ejecutando, int(nro_pagina), frame)
		return contenido, dir_fisica
	}

	peticion_WRITE := fmt.Sprintf("%d %d %d %s", *pid_ejecutando, dir_fisica.frame, dir_fisica.offset, datos)
	fullUrl := fmt.Sprintf("http://%s/memoria/WRITE", url_memo)

	respuesta, _ := utils.EnviarSolicitudHTTPString("POST", fullUrl, peticion_WRITE)
	if cache_pags_activa {
		aplicarAlgoritmoCachePags(int(nro_pagina), frame, dir_fisica.offset, respuesta, "WRITE")
	}

	utils.LoggerConFormato("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", *pid_ejecutando, int(nro_pagina), frame)

	log.Printf("Se esta intentando escribir en la direccion fisica [ %d | %d ]", dir_fisica.frame, dir_fisica.offset)
	return respuesta, dir_fisica
}

func requestREAD(direccion_logica int, tamanio int) (string, DireccionFisica) {

	desplazamiento := int(direccion_logica) % int(tam_pag)
	nro_pagina := math.Floor(float64(direccion_logica) / float64(tam_pag))

	frame, contenido := busquedaMemoriaFrame(direccion_logica, int(nro_pagina), "READ") //el contenido es solo para cache pags activa
	dir_fisica := MMU(frame, desplazamiento)

	// Encuentro en cache pags
	if contenido != "NO_ENCONTRE" {
		utils.LoggerConFormato("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", *pid_ejecutando, int(nro_pagina), frame)
		return contenido, dir_fisica
	}

	peticion_READ := fmt.Sprintf("%d %d %d %d", *pid_ejecutando, dir_fisica.frame, dir_fisica.offset, tamanio)
	fullUrl := fmt.Sprintf("http://%s/memoria/READ", url_memo)

	respuesta, _ := utils.EnviarSolicitudHTTPString("POST", fullUrl, peticion_READ)

	if cache_pags_activa {
		aplicarAlgoritmoCachePags(int(nro_pagina), frame, dir_fisica.offset, respuesta, "READ")
	}

	utils.LoggerConFormato("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", *pid_ejecutando, int(nro_pagina), frame)
	log.Printf("Se esta intentando leer en la direccion fisica [ %d | %d ]", dir_fisica.frame, dir_fisica.offset)

	return respuesta, dir_fisica
}

func busquedaFrameAMemoria(direccion_logica int, nro_pagina float64) int {

	frame := -1

	for nivel_actual := 1; nivel_actual <= int(cant_niveles); nivel_actual++ {
		//Santi: Lo mas hardcodeado que vi
		entrada_nivel_final := int(math.Floor(nro_pagina/math.Pow(float64(cant_entradas_tpag), float64((cant_niveles-nivel_actual))))) % int(cant_entradas_tpag)
		respuesta := busquedaTabla(*pid_ejecutando, nivel_actual, entrada_nivel_final)
		// -2 Direccionamiento invalido
		// -1 Todo bien, sigo al sgte nivel
		//>=0 Es un frame, lo devuelvo
		if respuesta >= 0 {
			utils.LoggerConFormato("PID : %d - OBTENER MARCO - Página: %d - Marco: %d", *pid_ejecutando, int(nro_pagina), respuesta)
			return frame

		}
	}
	return frame
}

func busquedaTabla(pid int, nivel_actual int, entrada_a_acceder int) int {
	solicitud_acceso_tpaginas := fmt.Sprintf("%d %d %d", pid, nivel_actual, entrada_a_acceder)
	fullUrl := fmt.Sprintf("http://%s/memoria/busqueda_tabla", url_memo)
	aux, _ := utils.EnviarSolicitudHTTPString("POST", fullUrl, solicitud_acceso_tpaginas)
	respuesta, _ := strconv.Atoi(aux)
	return respuesta
}

func buscarEnTLB(direccion_logica int, nro_pagina int) int {

	for i := 0; i <= config_CPU.Cant_entradas_TLB; i++ {
		if tlb[i].pagina == nro_pagina {
			// Caso TLB HIT
			utils.LoggerConFormato("PID: %d - TLB HIT - Pagina: %d", *pid_ejecutando, nro_pagina)
			if tlb_activa {
				tlb[i].last_recently_used = time.Now()
			}

			return tlb[i].frame
		}
	}
	// Caso TLB MISS
	utils.LoggerConFormato("PID: %d - TLB MISS - Pagina: %d", *pid_ejecutando, nro_pagina)
	frame := busquedaFrameAMemoria(direccion_logica, float64(nro_pagina))
	hayEspacioEnTLB, entrada := chequearEspacioEnTLB()

	if hayEspacioEnTLB {
		cambiarEstadoMarco(nro_pagina, frame, entrada)
		return frame
	}

	aplicarAlgoritmoTLB()
	return frame
}

func buscarEnCachePags(dir_logica int, nro_pagina int, accion string) (int, string) {

	for i := 0; i <= config_CPU.Cant_entradas_cache; i++ {
		if cache_pags[i].pagina == nro_pagina {
			utils.LoggerConFormato("PID: %d- Cache Hit - Pagina: %d", *pid_ejecutando, nro_pagina)
			switch accion {
			case "READ":
				return cache_pags[i].frame, cache_pags[i].contenido

			case "WRITE":
				return cache_pags[i].frame, "OK"
			}

		}
	}

	utils.LoggerConFormato("PID: %d- Cache Miss - Pagina: %d", *pid_ejecutando, nro_pagina)

	if tlb_activa {
		frame := buscarEnTLB(dir_logica, nro_pagina)
		return frame, "NO_ENCONTRE"
	}

	frame := busquedaFrameAMemoria(dir_logica, float64(nro_pagina))
	return frame, "NO_ENCONTRE"

}

func aplicarAlgoritmoTLB() error {
	alg_reemplazo := config_CPU.Alg_repl_TLB
	switch alg_reemplazo {
	case "FIFO":
		aux := tlb[0]
		aux_entrada := 0
		aux_timestamp_tiempo_vida := aux.tiempo_vida.Sub(noticiero_metereologico)

		for i := 1; i < config_CPU.Cant_entradas_TLB; i++ {
			comparador_timestamp := tlb[i].tiempo_vida.Sub(noticiero_metereologico)
			if aux_timestamp_tiempo_vida < comparador_timestamp {
				aux = tlb[i]
				aux_entrada = i
				aux_timestamp_tiempo_vida = comparador_timestamp
			}
		}

		cambiarEstadoMarco(aux.pagina, aux.frame, aux_entrada)

		aux.tiempo_vida = time.Now()
		return nil

	case "LRU":
		aux := tlb[0]
		aux_entrada := 0
		aux_timestamp_lru := aux.tiempo_vida.Sub(noticiero_metereologico)

		for i := 1; i < config_CPU.Cant_entradas_TLB; i++ {
			comparador_timestamp := tlb[i].last_recently_used.Sub(noticiero_metereologico)
			if aux_timestamp_lru < comparador_timestamp {
				aux = tlb[i]
				aux_entrada = i
				aux_timestamp_lru = comparador_timestamp
			}
		}

		cambiarEstadoMarco(aux.pagina, aux.frame, aux_entrada)
		aux.last_recently_used = time.Now()

		return nil
	default:
		return fmt.Errorf("y,la verda q me pasaste cualquier verdura en el algoritmo de TLB")
	}
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

func busquedaMemoriaFrame(dir_logica int, nro_pagina int, accion string) (int, string) {
	/*---------------------------------------------------(frame,contenido,mensaje)
	frame
	*/

	if cache_pags_activa {
		frame, contenido := buscarEnCachePags(dir_logica, nro_pagina, accion)
		// Caso encontre
		if contenido != "NO_ENCONTRE" {
			return frame, contenido
		}
		return frame, contenido

	}

	if tlb_activa {
		frame := buscarEnTLB(dir_logica, nro_pagina)
		return frame, ""

	}

	frame := busquedaFrameAMemoria(dir_logica, float64(nro_pagina))

	return frame, ""
}

func aplicarAlgoritmoCachePags(nro_pagina int, frame int, offset int, contenido string, accion string) {
	algoritmo := config_CPU.Alg_repl_cache

	switch algoritmo {
	case "CLOCK":
		for i := range config_CPU.Cant_entradas_cache {
			if cache_pags[i].bit_uso == 0 {
				actualizarEntradaCache(i, nro_pagina, frame, offset, contenido, accion)
				return
			}
			cache_pags[i].bit_uso = 0
		}
		for i := 0; i < config_CPU.Cant_entradas_cache; i++ {
			if cache_pags[i].bit_uso == 0 {
				actualizarEntradaCache(i, nro_pagina, frame, offset, contenido, accion)
				return
			}
		}

	case "CLOCK-M":
		// 1) Primera pasada
		//    u=0;m=0
		// for i := 0; i < config_CPU.Cant_entradas_cache; i++ {
		// 	if cache_pags[i].bit_uso == 0 && cache_pags[i].bit_modificado == 0 {
		// 		actualizarEntradaCache(i, nro_pagina, frame, contenido, accion)
		// 		return
		// 	}
		// }

		cicloCLockM(0, 0, 0, nro_pagina, frame, offset, contenido, accion)
		// 2) Segunda pasada
		//    u=0;m=1
		//Si no encuentro, u=0 -> u=1
		// for i := 0; i < config_CPU.Cant_entradas_cache; i++ {
		// 	if cache_pags[i].bit_uso == 0 && cache_pags[i].bit_modificado == 1 {
		// 		actualizarEntradaCache(i, nro_pagina, frame, contenido, accion)
		// 		return
		// 	}
		// 	cache_pags[i].bit_uso = 0
		// }
		cicloCLockM(1, 0, 1, nro_pagina, frame, offset, contenido, accion)
		// 3) Tercera pasada
		// reintento de 1)
		// for i := 0; i < config_CPU.Cant_entradas_cache; i++ {
		// 	if cache_pags[i].bit_uso == 0 && cache_pags[i].bit_modificado == 0 {
		// 		actualizarEntradaCache(i, nro_pagina, frame, contenido, accion)
		// 		return
		// 	}
		// }
		cicloCLockM(0, 0, 0, nro_pagina, frame, offset, contenido, accion)
		// 4) Cuarta pasada
		// reintento de 2)
		// for i := 0; i < config_CPU.Cant_entradas_cache; i++ {
		// 	if cache_pags[i].bit_uso == 0 && cache_pags[i].bit_modificado == 1 {
		// 		actualizarEntradaCache(i, nro_pagina, frame, contenido, accion)
		// 		return
		// 	}
		// }
		cicloCLockM(0, 0, 1, nro_pagina, frame, offset, contenido, accion)
	}
}

func cicloCLockM(sector_extra int, valor_uso int, valor_modificado int, nro_pagina int, frame int, offset int, contenido string, accion string) {

	for i := range config_CPU.Cant_entradas_cache {
		if cache_pags[i].bit_uso == valor_uso && cache_pags[i].bit_modificado == valor_modificado {
			actualizarEntradaCache(i, nro_pagina, frame, offset, contenido, accion)
			return
		}
		if sector_extra == 1 {
			cache_pags[i].bit_uso = 0
		}

	}

}
func actualizarEntradaCache(posicion int, nro_pagina int, frame int, offset int, contenido string, accion string) {
	cache_pags[posicion].contenido = contenido
	cache_pags[posicion].frame = frame
	cache_pags[posicion].offset = offset
	cache_pags[posicion].pagina = nro_pagina
	cache_pags[posicion].bit_uso = 1

	if accion == "WRITE" {
		cache_pags[posicion].bit_modificado = 1
	}

	utils.LoggerConFormato("PID: %d - Cache Add - Pagina: %d", *pid_ejecutando, nro_pagina)
}

func MMU(frame int, offset int) DireccionFisica {
	dir_fisica := DireccionFisica{
		frame:  frame,
		offset: offset,
	}
	return dir_fisica
}

func actualizarPagCompleta(entrada_a_actualizar *EntradaCachePag) {
	fullUrl := fmt.Sprintf("http://%s/memoria/actualizar_entrada_cache", url_memo)
	peticion := fmt.Sprintf("%d %d", *pid_ejecutando, entrada_a_actualizar.frame)
	utils.EnviarSolicitudHTTPString("POST", fullUrl, peticion)
	utils.LoggerConFormato("PID: %d - Memory Update - Página: %d - Frame: %d",
		*pid_ejecutando, entrada_a_actualizar.pagina, entrada_a_actualizar.frame)
}
