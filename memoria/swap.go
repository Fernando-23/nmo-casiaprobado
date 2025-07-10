package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"
)

func (memo *Memo) CrearSwapfile() {
	_, err := os.Create(memo.config_memo.Path_swap)

	if err != nil {
		fmt.Println("Error en crear swapfile")
		return
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

func (memo *Memo) EscribirEnSwap(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (EscribirEnSwap) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición escribir en swap", "mensaje", mensaje)

	pid, err := strconv.Atoi(mensaje)

	if err != nil {
		slog.Error("Error - (EscribirEnSwap) - Error convirtiendo pid")
	}

	file, err := os.OpenFile(memo.config_memo.Path_swap, os.O_RDWR|os.O_CREATE, 0666)

	if err != nil {
		slog.Error("Error - (EscribirEnSwap) - Error en abrir swapfile")
		return
	}
	defer file.Close()

	mutex_lprocs.Lock()
	frames_asignados_a_pid := memo.l_proc[pid].ptr_a_frames_asignados
	tamanio := memo.l_proc[pid].tamanio

	mutex_memoriaPrincipal.Lock()
	for i := range frames_asignados_a_pid {
		ptr_frame_asignado := frames_asignados_a_pid[i]
		inicio := *frames_asignados_a_pid[i] * memo.config_memo.Tamanio_pag
		fin_de_pag := inicio + memo.config_memo.Tamanio_pag //Liam vino de rendir funcional mepa

		contenido_pag := memo.memoria_principal[inicio:fin_de_pag]
		*ptr_frame_asignado = -1
		file.Write(contenido_pag)
	}
	mutex_memoriaPrincipal.Unlock()

	mutex_swap.Lock()
	memo.CargarDataSwap(pid, tamanio)
	mutex_swap.Unlock()

	memo.l_proc[pid].ptr_a_frames_asignados = memo.l_proc[pid].ptr_a_frames_asignados[:0]
	mutex_lprocs.Unlock()

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Bajadas_de_swap)
	mutex_metricas.Unlock()

	mutex_tamanioMemoActual.Lock()
	gb_tam_memo_actual += tamanio
	mutex_tamanioMemoActual.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (memo *Memo) QuitarDeSwap(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (QuitarDeSwap) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición quitar de swap", "mensaje", mensaje)

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

	inicio_proceso := proceso_en_swap.inicio
	tamanio := proceso_en_swap.tamanio

	mutex_tamanioMemoActual.Lock()
	if !HayEspacio(tamanio) {
		mutex_tamanioMemoActual.Unlock()
		slog.Debug("Debug - (QuitarDeSwap) - No hay espacio en memoria para sacar un proceso de swap")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_OK"))
		return
	}
	mutex_tamanioMemoActual.Unlock()

	file, err := os.Open(memo.config_memo.Path_swap)

	if err != nil {
		slog.Error("Error - (QuitarDeSwap) - Error en abrir swapfile")
		http.Error(w, "Error accediendo al archivo swap", http.StatusBadGateway)
		return
	}

	if _, err := file.Seek(int64(inicio_proceso), 1); err != nil {
		slog.Error("Error - (QuitarDeSwap) - Fallo al hacer seek en swap", "error", err)
		http.Error(w, "Error de lectura", http.StatusInternalServerError)
		return
	}

	contenido_proc := make([]byte, tamanio)
	bytes_leidos, err := file.Read(contenido_proc)

	if err != nil {
		slog.Error("Error - (QuitarDeSwap) - Fallo al leer de swap", "error", err)
		http.Error(w, "Error leyendo swap", http.StatusInternalServerError)
		return
	}

	if bytes_leidos != tamanio { //leyamal yo leyo tu leyes el leye
		slog.Error("Error - (QuitarDeSwap) - No leo bien necesito gafas")
		http.Error(w, "Error leyendo swap", http.StatusInternalServerError)
		return
	}

	slog.Debug("Debug - (QuitarDeSwap) - Contenido leido de swap",
		"pid", pid,
		"bytes", bytes_leidos,
	)

	file.Seek(0, 0)
	file.Close()
	// compromiso de copactar para el segundo cuatri
	//segurisimo

	//TEMA ASIGNAR FRAMES AL PROCESO SACADO DE SWAP

	mutex_tablaPaginas.Lock()
	defer mutex_tablaPaginas.Unlock()
	ptr_tpag_del_pid := memo.ptrs_raiz_tpag[pid]
	if memo.config_memo.Cant_niveles == 0 {
		memo.EscribirDeSwapAMemoriaPrincipal(pid, tamanio, ptr_tpag_del_pid, w)
		return
	}

	for i := 1; i < memo.config_memo.Cant_niveles; i++ {
		ptr_tpag_del_pid = ptr_tpag_del_pid.sgte_nivel
	}

	memo.EscribirDeSwapAMemoriaPrincipal(pid, tamanio, ptr_tpag_del_pid, w)

}

func (memo *Memo) EscribirDeSwapAMemoriaPrincipal(pid int, tamanio int, ptr_tpag_del_pid *NivelTPag, w http.ResponseWriter) {

	memo.AsignarFramesAProceso(ptr_tpag_del_pid, tamanio, pid)

	mutex_lprocs.Lock()
	frames_asignados_a_pid := memo.l_proc[pid].ptr_a_frames_asignados
	var contenido_proc_dividido []byte

	mutex_memoriaPrincipal.Lock()
	for i := range frames_asignados_a_pid {
		//tema contenido
		inicio_contenido := i * gb_tamanio_pag
		fin_contenido := inicio_contenido + gb_tamanio_pag
		pagina_a_escribir := contenido_proc_dividido[inicio_contenido:fin_contenido]

		//tema memoria
		frame := *frames_asignados_a_pid[i]
		inicio_memo := frame * gb_tamanio_pag
		fin_memo := inicio_memo + gb_tamanio_pag

		copy(memo.memoria_principal[inicio_memo:fin_memo], pagina_a_escribir)
	}
	mutex_memoriaPrincipal.Unlock()
	mutex_lprocs.Unlock()

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Subidas_a_memoria)
	mutex_metricas.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (memo *Memo) EliminarProcesoDeSwap(pid int) {

	mutex_swap.Lock()
	defer mutex_swap.Unlock()

	if proceso_en_swap := memo.swap.espacio_contiguo[pid]; proceso_en_swap != nil {

		delete(memo.swap.espacio_contiguo, pid)

		nueva_instancia_espacio_libre := &EspacioLibre{
			inicio:  proceso_en_swap.inicio,
			tamanio: proceso_en_swap.tamanio,
		}

		memo.swap.espacio_libre = append(memo.swap.espacio_libre, nueva_instancia_espacio_libre)
	}
}
