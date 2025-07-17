package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func InicializarSwap(path string) *DataSwap {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(fmt.Sprintf("No se pudo abrir el archivo de swap: %v", err))
	}

	return &DataSwap{
		ultimo_byte:      0,
		espacio_contiguo: make(map[int]*ProcesoEnSwap),
		espacio_libre:    []*EspacioLibre{},
		SwapFile:         file,
	}
}

func (memo *Memo) CargarDataSwap(pid int, tamanio int) {

	//Primero chequeo si hay un espacio entre medio
	//(parchesito para no compactar basicamente)
	for i := range memo.swap.espacio_libre {
		if tamanio <= memo.swap.espacio_libre[i].tamanio {
			memo.swap.espacio_contiguo[pid] = &ProcesoEnSwap{
				inicio:  memo.swap.espacio_libre[i].inicio,
				tamanio: tamanio,
			}
			memo.swap.espacio_libre = slices.Delete(memo.swap.espacio_libre, i, i+1)
			return
		}
	}
	//Si no encuentra, que escriba al final
	memo.swap.espacio_contiguo[pid] = &ProcesoEnSwap{
		inicio:  memo.swap.ultimo_byte,
		tamanio: tamanio,
	}
	memo.swap.ultimo_byte += tamanio
}

func (memo *Memo) EscribirProcesoEnSwap(pid int) error {
	mutex_lprocs.Lock()
	proc, ok := memo.Procesos[pid]
	mutex_lprocs.Unlock()

	if !ok {
		return fmt.Errorf("Proceso %d no encontrado", pid)
	}

	tamanio := proc.Tamanio

	var inicioSwap int = -1

	mutex_swap.Lock()
	defer mutex_swap.Unlock()

	// // Buscar espacio libre en la lista de huecos
	// for i, espacio := range memo.swap.espacio_libre {
	// 	if espacio.tamanio >= tamanio {
	// 		inicioSwap = espacio.inicio
	// 		if espacio.tamanio > tamanio {
	// 			espacio.inicio += tamanio
	// 			espacio.tamanio -= tamanio
	// 		} else {
	// 			memo.swap.espacio_libre = append(memo.swap.espacio_libre[:i], memo.swap.espacio_libre[i+1:]...)
	// 		}
	// 		break
	// 	}
	// }

	// No había espacio libre suficiente: expandimos
	if inicioSwap == -1 || memo.swap.ultimo_byte == 0 {
		inicioSwap = memo.swap.ultimo_byte
		memo.swap.ultimo_byte += tamanio
	}

	// Escribir contenido del proceso desde memoria principal
	tamPag := memo.Config.Tamanio_pag
	offsetSwap := inicioSwap

	mutex_memoriaPrincipal.Lock()
	defer mutex_memoriaPrincipal.Unlock()

	mutex_framesDisponibles.Lock()

	for i, frame := range memo.Frames {
		if frame.Usado && frame.PidOcupante == pid {
			inicio := i * tamPag
			fin := inicio + tamPag
			contenido := memo.memoria_principal[inicio:fin]

			slog.Debug("Debug - (EscribirProcesoEnSwap) - Se va a intentar escribir un proceso en swap",
				"contenido", contenido,
				"offset", offsetSwap)
			_, err := memo.swap.SwapFile.WriteAt(contenido, int64(offsetSwap))
			if err != nil {
				mutex_framesDisponibles.Unlock()
				return fmt.Errorf("error escribiendo en archivo de swap: %v", err)
			}

			offsetSwap += tamPag

			// Liberar frame
			frame.Usado = false
			frame.PidOcupante = -1
			frame.NumeroPagina = -1
		}
	}
	mutex_framesDisponibles.Unlock()
	// Limpiar referencias en tabla de páginas
	mutex_lprocs.Lock()
	memo.LimpiarReferenciasDeFrames(pid)
	mutex_lprocs.Unlock()
	// Guardar ubicación del proceso en el swap
	memo.swap.espacio_contiguo[pid] = &ProcesoEnSwap{
		inicio:  inicioSwap,
		tamanio: tamanio,
	}

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Bajadas_de_swap)
	mutex_metricas.Unlock()

	mutex_tamanioMemoActual.Lock()
	gb_tam_memo_actual += tamanio
	slog.Debug("Debug - (EscribirProcesoEnSwap) - Nuevo tamanio de memoria actual",
		"gb_tam_memo_actual", gb_tam_memo_actual)
	mutex_tamanioMemoActual.Unlock()

	utils.LoggerConFormato("PID %d - Proceso escrito en swap desde byte %d a %d", pid, inicioSwap, offsetSwap)
	return nil
}

func (memo *Memo) LimpiarReferenciasDeFrames(pid int) {
	proc := memo.Procesos[pid]
	numPaginas := memo.LaCuentitaMaestro(proc.Tamanio)

	for pagina := 0; pagina < numPaginas; pagina++ {
		indices := memo.indicesParaPagina(pagina)
		actual := proc.TablaPagsRaiz

		for nivel := 0; nivel < memo.Config.Cant_niveles-1; nivel++ {
			entry := actual.Entradas[indices[nivel]]
			if entry == nil || entry.SiguienteNivel == nil {
				actual = nil
				break
			}
			actual = entry.SiguienteNivel
		}

		if actual != nil {
			finalEntry := actual.Entradas[indices[len(indices)-1]]
			if finalEntry != nil {
				finalEntry.NumeroDeFrame = nil
			}
		}
	}
}

func (memo *Memo) EscribirEnSwap(w http.ResponseWriter, r *http.Request) {
	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (EscribirEnSwap) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)
	slog.Debug("Debug - (EscribirEnSwap) - Llego peticion escribir en swap", "mensaje", mensaje)

	pid, err := strconv.Atoi(string(mensaje))
	if err != nil {
		slog.Error("Error - (EscribirEnSwap) - Error convirtiendo pid", "input", mensaje)
		http.Error(w, "PID invalido", http.StatusBadRequest)
		return
	}

	if err := memo.EscribirProcesoEnSwap(pid); err != nil {
		slog.Warn("Cuidadito - (EscribirEnSwap) - No se pudo escribir el proceso en swap", "error", err)
		http.Error(w, "No se pudo escribir el proceso en swap", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (memo *Memo) RestaurarProcesoDesdeSwap(pid int) error {
	mutex_lprocs.Lock()
	defer mutex_lprocs.Unlock()
	proc, ok := memo.Procesos[pid]
	if !ok {

		return fmt.Errorf("Proceso %d no encontrado", pid)
	}

	infoSwap, existe := memo.swap.espacio_contiguo[pid]
	if !existe {
		return fmt.Errorf("Proceso %d no esta en swap", pid)
	}

	offsetSwap := infoSwap.inicio // donde inicia el proceso en el swapfile
	bytes_leidos := 0
	tamanio_proc_en_swap := infoSwap.tamanio

	numPaginas := memo.LaCuentitaMaestro(tamanio_proc_en_swap)

	tamPag := memo.Config.Tamanio_pag

	mutex_memoriaPrincipal.Lock()
	defer mutex_memoriaPrincipal.Unlock()

	mutex_framesDisponibles.Lock()
	defer mutex_framesDisponibles.Unlock()

	for pagina := 0; pagina < numPaginas; pagina++ {
		frameIdx := memo.buscarFrameLibre()
		if frameIdx == -1 {
			return fmt.Errorf("no hay frames libres para restaurar proceso %d", pid)
		}

		//Leer datos desde swap
		buffer := make([]byte, tamPag)
		n, err := memo.swap.SwapFile.ReadAt(buffer, int64(offsetSwap))
		if err != nil && err != io.EOF {
			return fmt.Errorf("error leyendo swap para proceso %d: %v", pid, err)
		}
		if n == 0 {
			return fmt.Errorf("no se leyo nada del swap para proceso %d", pid)
		}
		// copiar a memoria principal en frame libre
		inicioMem := frameIdx * tamPag
		copy(memo.memoria_principal[inicioMem:inicioMem+tamPag], buffer[:n])

		//Marcar frame como ocupado
		memo.Frames[frameIdx].Usado = true
		memo.Frames[frameIdx].PidOcupante = pid
		memo.Frames[frameIdx].NumeroPagina = pagina

		memo.AsignarUnFrameATPags(proc, pagina, frameIdx)

		offsetSwap += tamPag
		bytes_leidos += n
	}

	memo.swap.espacio_libre = append(memo.swap.espacio_libre, &EspacioLibre{
		inicio:  infoSwap.inicio,
		tamanio: infoSwap.tamanio,
	})

	delete(memo.swap.espacio_contiguo, pid)

	mutex_tamanioMemoActual.Lock()
	gb_tam_memo_actual -= proc.Tamanio
	slog.Debug("Debug - (RestaurarProcesoDesdeSwap) - Nuevo tamanio de la memoria principal", "gb_tam_memo_actual", gb_tam_memo_actual)
	mutex_tamanioMemoActual.Unlock()

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Subidas_a_memoria)
	mutex_metricas.Unlock()

	slog.Debug("Debug - (RestaurarProcesoDesdeSwap) - Proceso restaurado desde swap", "pid", pid, "bytes_leidos", bytes_leidos)
	return nil
}

func (memo *Memo) AsignarUnFrameATPags(proc *Proceso, nro_pagina int, frameIdx int) {
	indices := memo.indicesParaPagina(nro_pagina)
	actual := proc.TablaPagsRaiz

	for nivel := 0; nivel < memo.Config.Cant_niveles-1; nivel++ {
		idx := indices[nivel]
		if actual.Entradas[idx] == nil {
			actual.Entradas[idx] = &EntradaTablaDePaginas{
				SiguienteNivel: &TablaDePaginas{
					Entradas: make([]*EntradaTablaDePaginas, memo.Config.EntradasPorNivel),
				},
			}
		}
		actual = actual.Entradas[idx].SiguienteNivel
	}

	ultimoIdx := indices[memo.Config.Cant_niveles-1]
	actual.Entradas[ultimoIdx] = &EntradaTablaDePaginas{
		NumeroDeFrame: &frameIdx,
	}

}

func (memo *Memo) QuitarDeSwap(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (QuitarDeSwap) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}

	mensaje := string(body_Bytes)

	slog.Debug("Debug - (QuitarDeSwap) - Llego peticion quitar de swap", "mensaje", mensaje)

	pid, err := strconv.Atoi(mensaje)

	if err != nil {
		slog.Error("Error - (QuitarDeSwap) - Conversion de pid ")
		return
	}

	mutex_swap.Lock()
	defer mutex_swap.Unlock()
	proceso_en_swap, existe := memo.swap.espacio_contiguo[pid]

	if !existe {
		slog.Error("Error - (QuitarDeSwap) - NO existe el proceso en swap", "pid", pid)
		http.Error(w, "Proceso no encontrado en swap", http.StatusNotFound)
		return
	}

	tamanio_proc_en_swap := proceso_en_swap.tamanio

	mutex_tamanioMemoActual.Lock()
	if !HayEspacio(tamanio_proc_en_swap) {
		mutex_tamanioMemoActual.Unlock()
		slog.Debug("Debug - (QuitarDeSwap) - No hay espacio en memoria para sacar un proceso de swap")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_OK"))
		return
	}
	mutex_tamanioMemoActual.Unlock()

	if memo.RestaurarProcesoDesdeSwap(pid) != nil {
		slog.Error("Error - (QuitarDeSwap) - No hay espacio en memoria para sacar un proceso de swap", "error", err)

		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (memo *Memo) EliminarProcesoDeSwap(pid int) {

	mutex_swap.Lock()
	defer mutex_swap.Unlock()

	if proceso_en_swap := memo.swap.espacio_contiguo[pid]; proceso_en_swap != nil {

		slog.Debug("EliminarProcesoDeSwap - espacio liberado en swap",
			"pid", pid,
			"inicio", proceso_en_swap.inicio,
			"tamanio", proceso_en_swap.tamanio)

		delete(memo.swap.espacio_contiguo, pid)

		nueva_instancia_espacio_libre := &EspacioLibre{
			inicio:  proceso_en_swap.inicio,
			tamanio: proceso_en_swap.tamanio,
		}

		memo.swap.espacio_libre = append(memo.swap.espacio_libre, nueva_instancia_espacio_libre)
	}
}

func (memo *Memo) recursivamenteLiberarTabla(tabla *TablaDePaginas) {
	for _, entrada := range tabla.Entradas {
		if entrada == nil {
			continue
		}
		if entrada.SiguienteNivel != nil {
			memo.recursivamenteLiberarTabla(entrada.SiguienteNivel)
		}
	}
}

func (memo *Memo) EliminarProceso(pid int) error {
	mutex_lprocs.Lock()
	defer mutex_lprocs.Unlock()
	proc, ok := memo.Procesos[pid]
	if !ok {
		return fmt.Errorf("Proceso %d no existe", pid)
	}

	slog.Debug("Debug - (EliminarProceso) - Encontre el proceso, voy a proceder a eliminarlo",
		"pid", pid)

	mutex_framesDisponibles.Lock()
	for _, frame := range memo.Frames {
		if frame.Usado && frame.PidOcupante == pid {
			frame.Usado = false
			frame.PidOcupante = -1
			frame.NumeroPagina = -1
		}
	}
	mutex_framesDisponibles.Unlock()

	//liberar espacio de swap si el proceso estaba en swap

	mutex_swap.Lock()

	if ubicacion, existe := memo.swap.espacio_contiguo[pid]; existe {
		memo.swap.espacio_libre = append(memo.swap.espacio_libre, &EspacioLibre{
			inicio:  ubicacion.inicio,
			tamanio: ubicacion.tamanio,
		})
		delete(memo.swap.espacio_contiguo, pid)
	}
	// hay mas memoria, actualizo la global

	mutex_swap.Unlock()
	mutex_tamanioMemoActual.Lock()
	gb_tam_memo_actual += proc.Tamanio
	slog.Debug("Debug - (EliminarProceso) - Nuevo tamanio de memoria principal despues de un EXIT",
		"pid", pid, "gb_tam_memo_actual", gb_tam_memo_actual)
	mutex_tamanioMemoActual.Unlock()

	memo.recursivamenteLiberarTabla(proc.TablaPagsRaiz)

	delete(memo.Procesos, pid)

	return nil
}

func (memo *Memo) ImprimirSwap(pid int) error {
	mutex_swap.Lock()
	ProcesoEnSwap, ok := memo.swap.espacio_contiguo[pid]
	mutex_swap.Unlock()

	if !ok {
		return fmt.Errorf("PID %d no tien entrada en swap", pid)
	}

	inicio := ProcesoEnSwap.inicio
	tamanio := ProcesoEnSwap.tamanio

	buffer := make([]byte, tamanio)

	_, err := memo.swap.SwapFile.ReadAt(buffer, int64(inicio))
	if err != nil {
		return fmt.Errorf("error leyendo swap para PID %d: %v", pid, err)
	}

	fmt.Println("===== DUMP SWAP PID:", pid, "=====")
	fmt.Printf("Desde byte %d hasta %d\n", inicio, inicio+tamanio)
	fmt.Println("=============================")
	fmt.Println(string(buffer))
	fmt.Println("contenido como bytes:")
	fmt.Printf("%v\n", buffer)
	fmt.Println("=============================")

	return nil

}
