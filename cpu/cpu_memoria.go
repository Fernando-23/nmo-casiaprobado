package main

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (cpu *CPU) RequestWRITE(direccion_logica int, datos string) (string, DireccionFisica) {

	desplazamiento := int(direccion_logica) % int(tam_pag)
	nro_pagina := math.Floor(float64(direccion_logica) / float64(tam_pag))

	frame, contenido := cpu.BusquedaMemoriaFrame(int(nro_pagina), "WRITE") //el contenido es solo para cache pags activa

	dir_fisica := cpu.MMU(frame, desplazamiento)

	//Encuentro en cache pags
	if contenido != "NO_ENCONTRE" {
		utils.LoggerConFormato("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", cpu.Proc_ejecutando.Pid, int(nro_pagina), frame)
		return contenido, dir_fisica
	}

	respuesta, _ := utils.FormatearUrlYEnviar(cpu.Url_memoria, "/WRITE", true, "%d %d %d %s", cpu.Proc_ejecutando.Pid, dir_fisica.frame, dir_fisica.offset, datos)

	if cache_pags_activa {
		cpu.AplicarAlgoritmoCachePags(int(nro_pagina), frame, dir_fisica.offset, respuesta, "WRITE")
	}

	utils.LoggerConFormato("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", cpu.Proc_ejecutando.Pid, int(nro_pagina), frame)

	log.Printf("Se esta intentando escribir en la direccion fisica [ %d | %d ]", dir_fisica.frame, dir_fisica.offset)
	return respuesta, dir_fisica
}

func (cpu *CPU) RequestREAD(direccion_logica int, tamanio int) (string, DireccionFisica) {

	desplazamiento := int(direccion_logica) % int(tam_pag)
	nro_pagina := math.Floor(float64(direccion_logica) / float64(tam_pag))

	frame, contenido := cpu.BusquedaMemoriaFrame(int(nro_pagina), "READ") //el contenido es solo para cache pags activa
	dir_fisica := cpu.MMU(frame, desplazamiento)

	// Encuentro en cache pags
	if contenido != "NO_ENCONTRE" {
		utils.LoggerConFormato("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", cpu.Proc_ejecutando.Pid, int(nro_pagina), frame)
		return contenido, dir_fisica
	}

	respuesta, _ := utils.FormatearUrlYEnviar(cpu.Url_memoria, "/READ", true, "%d %d %d %d", cpu.Proc_ejecutando.Pid, dir_fisica.frame, dir_fisica.offset, tamanio)

	if cache_pags_activa {
		cpu.AplicarAlgoritmoCachePags(int(nro_pagina), frame, dir_fisica.offset, respuesta, "READ")
	}

	utils.LoggerConFormato("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", cpu.Proc_ejecutando.Pid, int(nro_pagina), frame)
	log.Printf("Se esta intentando leer en la direccion fisica [ %d | %d ]", dir_fisica.frame, dir_fisica.offset)

	return respuesta, dir_fisica
}

func (cpu *CPU) BusquedaFrameAMemoria(nro_pagina float64) int {

	frame := -1

	for nivel_actual := 1; nivel_actual <= int(cant_niveles); nivel_actual++ {
		//Santi: Lo mas hardcodeado que vi
		entrada_nivel_final := int(math.Floor(nro_pagina/math.Pow(float64(cant_entradas_tpag), float64((cant_niveles-nivel_actual))))) % int(cant_entradas_tpag)

		respuesta := cpu.BusquedaTabla(cpu.Proc_ejecutando.Pid, nivel_actual, entrada_nivel_final)
		//    "SEGUI" Todo bien, sigo al sgte nivel
		// != "SEGUI" Es un frame, lo devuelvo
		if respuesta != "SEGUI" {
			frame, _ := strconv.Atoi(respuesta)
			utils.LoggerConFormato("PID : %d - OBTENER MARCO - Página: %d - Marco: %d", cpu.Proc_ejecutando.Pid, int(nro_pagina), frame)

			return frame
		}
	}
	return frame
}

func (cpu *CPU) BusquedaTabla(pid int, nivel_actual int, entrada_a_acceder int) string {
	respuesta, _ := utils.FormatearUrlYEnviar(cpu.Url_memoria, "/busqueda_tabla", true, "%d %d %d", pid, nivel_actual, entrada_a_acceder)
	return respuesta
}

func (cpu *CPU) BuscarEnTLB(nro_pagina int) int {

	for i := 0; i <= cpu.Config_CPU.Cant_entradas_TLB; i++ {
		if cpu.Tlb[i].pagina == nro_pagina {
			// Caso TLB HIT
			utils.LoggerConFormato("PID: %d - TLB HIT - Pagina: %d", cpu.Proc_ejecutando.Pid, nro_pagina)
			if tlb_activa {
				cpu.Tlb[i].last_recently_used = time.Now()
			}

			return cpu.Tlb[i].frame
		}
	}
	// Caso TLB MISS
	utils.LoggerConFormato("PID: %d - TLB MISS - Pagina: %d", cpu.Proc_ejecutando.Pid, nro_pagina)
	frame := cpu.BusquedaFrameAMemoria(float64(nro_pagina))
	hayEspacioEnTLB, entrada := cpu.ChequearEspacioEnTLB()

	if hayEspacioEnTLB {
		cpu.CambiarEstadoMarco(nro_pagina, frame, entrada)
		return frame
	}

	cpu.AplicarAlgoritmoTLB()
	return frame
}

func (cpu *CPU) BuscarEnCachePags(nro_pagina int, accion string) (int, string) {

	for i := 0; i <= cpu.Config_CPU.Cant_entradas_cache; i++ {
		if cpu.Cache_pags[i].pagina == nro_pagina {
			utils.LoggerConFormato("PID: %d- Cache Hit - Pagina: %d", cpu.Proc_ejecutando.Pid, nro_pagina)
			switch accion {
			case "READ":
				return cpu.Cache_pags[i].frame, cpu.Cache_pags[i].contenido

			case "WRITE":
				return cpu.Cache_pags[i].frame, "OK"
			}

		}
	}

	utils.LoggerConFormato("PID: %d- Cache Miss - Pagina: %d", cpu.Proc_ejecutando.Pid, nro_pagina)

	//lock tlb_activa
	if tlb_activa {
		//unlock tlb_activa
		frame := cpu.BuscarEnTLB(nro_pagina)
		return frame, "NO_ENCONTRE"
	}

	frame := cpu.BusquedaFrameAMemoria(float64(nro_pagina))
	return frame, "NO_ENCONTRE"

}

func (cpu *CPU) AplicarAlgoritmoTLB() error {
	alg_reemplazo := cpu.Config_CPU.Alg_repl_TLB
	switch alg_reemplazo {
	case "FIFO":
		aux := cpu.Tlb[0]
		aux_entrada := 0
		aux_timestamp_tiempo_vida := aux.tiempo_vida.Sub(noticiero_metereologico)

		for i := 1; i < cpu.Config_CPU.Cant_entradas_TLB; i++ {
			comparador_timestamp := cpu.Tlb[i].tiempo_vida.Sub(noticiero_metereologico)
			if aux_timestamp_tiempo_vida < comparador_timestamp {
				aux = cpu.Tlb[i]
				aux_entrada = i
				aux_timestamp_tiempo_vida = comparador_timestamp
			}
		}

		cpu.CambiarEstadoMarco(aux.pagina, aux.frame, aux_entrada)

		aux.tiempo_vida = time.Now()
		return nil

	case "LRU":
		aux := cpu.Tlb[0]
		aux_entrada := 0
		aux_timestamp_lru := aux.tiempo_vida.Sub(noticiero_metereologico)

		for i := 1; i < cpu.Config_CPU.Cant_entradas_TLB; i++ {
			comparador_timestamp := cpu.Tlb[i].last_recently_used.Sub(noticiero_metereologico)
			if aux_timestamp_lru < comparador_timestamp {
				aux = cpu.Tlb[i]
				aux_entrada = i
				aux_timestamp_lru = comparador_timestamp
			}
		}

		cpu.CambiarEstadoMarco(aux.pagina, aux.frame, aux_entrada)
		aux.last_recently_used = time.Now()

		return nil
	default:
		return fmt.Errorf("y,la verda q me pasaste cualquier verdura en el algoritmo de TLB")
	}
}

func (cpu *CPU) CambiarEstadoMarco(nro_pagina int, frame int, entrada_tlb int) {
	cpu.Tlb[entrada_tlb].pagina = nro_pagina
	cpu.Tlb[entrada_tlb].frame = frame
	utils.LoggerConFormato("Se realizo un cambio de marco en la TLB correctamente")
}

func (cpu *CPU) ChequearEspacioEnTLB() (bool, int) {
	for i := 0; i <= cpu.Config_CPU.Cant_entradas_TLB; i++ {
		if cpu.Tlb[i].pagina < 0 {
			return true, i
		}
	}
	return false, -1
}

func (cpu *CPU) BusquedaMemoriaFrame(nro_pagina int, accion string) (int, string) {
	/*---------------------------------------------------(frame,contenido,mensaje)
	frame
	*/

	if cache_pags_activa {
		frame, contenido := cpu.BuscarEnCachePags(nro_pagina, accion)
		// Caso encontre
		if contenido != "NO_ENCONTRE" {
			return frame, contenido
		}
		return frame, contenido

	}

	if tlb_activa {
		frame := cpu.BuscarEnTLB(nro_pagina)
		return frame, ""

	}

	frame := cpu.BusquedaFrameAMemoria(float64(nro_pagina))

	return frame, ""
}

func (cpu *CPU) AplicarAlgoritmoCachePags(nro_pagina int, frame int, offset int, contenido string, accion string) {
	algoritmo := cpu.Config_CPU.Alg_repl_cache
	cant_entradas_cache := cpu.Config_CPU.Cant_entradas_cache

	switch algoritmo {
	case "CLOCK":
		for i := range cant_entradas_cache {
			if cpu.Cache_pags[i].bit_uso == 0 {
				cpu.ActualizarEntradaCache(i, nro_pagina, frame, offset, contenido, accion)
				return
			}
			cpu.Cache_pags[i].bit_uso = 0
		}
		for i := 0; i < cpu.Config_CPU.Cant_entradas_cache; i++ {
			if cpu.Cache_pags[i].bit_uso == 0 {
				cpu.ActualizarEntradaCache(i, nro_pagina, frame, offset, contenido, accion)
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

		cpu.CicloCLockM(0, 0, 0, nro_pagina, frame, offset, contenido, accion)
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
		cpu.CicloCLockM(1, 0, 1, nro_pagina, frame, offset, contenido, accion)
		// 3) Tercera pasada
		// reintento de 1)
		// for i := 0; i < config_CPU.Cant_entradas_cache; i++ {
		// 	if cache_pags[i].bit_uso == 0 && cache_pags[i].bit_modificado == 0 {
		// 		actualizarEntradaCache(i, nro_pagina, frame, contenido, accion)
		// 		return
		// 	}
		// }
		cpu.CicloCLockM(0, 0, 0, nro_pagina, frame, offset, contenido, accion)
		// 4) Cuarta pasada
		// reintento de 2)
		// for i := 0; i < config_CPU.Cant_entradas_cache; i++ {
		// 	if cache_pags[i].bit_uso == 0 && cache_pags[i].bit_modificado == 1 {
		// 		actualizarEntradaCache(i, nro_pagina, frame, contenido, accion)
		// 		return
		// 	}
		// }
		cpu.CicloCLockM(0, 0, 1, nro_pagina, frame, offset, contenido, accion)
	}
}

func (cpu *CPU) CicloCLockM(sector_extra int, valor_uso int, valor_modificado int, nro_pagina int, frame int, offset int, contenido string, accion string) {
	cant_entradas_cache := cpu.Config_CPU.Cant_entradas_cache
	for i := range cant_entradas_cache {
		if cpu.Cache_pags[i].bit_uso == valor_uso && cpu.Cache_pags[i].bit_modificado == valor_modificado {
			cpu.ActualizarEntradaCache(i, nro_pagina, frame, offset, contenido, accion)
			return
		}
		if sector_extra == 1 {
			cpu.Cache_pags[i].bit_uso = 0
		}

	}

}

func (cpu *CPU) ActualizarEntradaCache(posicion int, nro_pagina int, frame int, offset int, contenido string, accion string) {
	// Posiblemente cpu.Cache_pags[posicion] =&EntradaCachePag{

	// }
	cpu.Cache_pags[posicion].contenido = contenido
	cpu.Cache_pags[posicion].frame = frame
	cpu.Cache_pags[posicion].offset = offset
	cpu.Cache_pags[posicion].pagina = nro_pagina
	cpu.Cache_pags[posicion].bit_uso = 1

	if accion == "WRITE" {
		cpu.Cache_pags[posicion].bit_modificado = 1
	}

	utils.LoggerConFormato("PID: %d - Cache Add - Pagina: %d", cpu.Proc_ejecutando.Pid, nro_pagina)

}

func (cpu *CPU) MMU(frame int, offset int) DireccionFisica {
	dir_fisica := DireccionFisica{
		frame:  frame,
		offset: offset,
	}
	return dir_fisica
}

func (cpu *CPU) ActualizarPagCompleta(entrada_a_actualizar *EntradaCachePag) {
	utils.FormatearUrlYEnviar(cpu.Url_memoria, "/actualizar_entrada_cache", false, "%d %d",
		cpu.Proc_ejecutando.Pid,
		entrada_a_actualizar.frame,
	)

	utils.LoggerConFormato("PID: %d - Memory Update - Página: %d - Frame: %d",
		cpu.Proc_ejecutando.Pid, entrada_a_actualizar.pagina, entrada_a_actualizar.frame)
}
