package main

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"
)

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

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición escribir en swap", "mensaje", mensaje)

	pid, err := strconv.Atoi(mensaje)

	if err != nil {
		slog.Error("Error - (EscribirEnSwap) - Error convirtiendo pid")
	}

	file, err := os.OpenFile(config_memo.Path_swap, os.O_RDWR|os.O_CREATE, 0666)

	if err != nil {
		slog.Error("Error - (EscribirEnSwap) - Error en abrir swapfile")
		return
	}
	defer file.Close()

	frames_asignados_a_pid := memo.l_proc[pid].ptr_a_frames_asignados
	tamanio := memo.l_proc[pid].tamanio

	for i := range frames_asignados_a_pid {
		ptr_frame_asignado := frames_asignados_a_pid[i]
		inicio := *frames_asignados_a_pid[i] * config_memo.Tamanio_pag
		fin_de_pag := inicio + config_memo.Tamanio_pag //Liam vino de rendir funcional mepa

		contenido_pag := memoria_principal[inicio:fin_de_pag]
		*ptr_frame_asignado = -1
		file.Write(contenido_pag)
	}

	memo.CargarDataSwap(pid, tamanio)

	memo.l_proc[pid].ptr_a_frames_asignados = memo.l_proc[pid].ptr_a_frames_asignados[:0]

	memo.IncrementarMetrica(pid, Bajadas_de_swap)

	// lock mutex_tamanioMemoActual.Lock()
	tam_memo_actual += tamanio
	// unlock mutex_tamanioMemoActual.Unlock()
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
	proceso_en_swap, existe := memo.swap.espacio_contiguo[pid]
	if !existe {
		slog.Error("Error - (QuitarDeSwap) - NO existe el proceso en swap", "pid", pid)
		http.Error(w, "Proceso no encontrado en swap", http.StatusNotFound)
		return
	}

	inicio_proceso := proceso_en_swap.inicio
	tamanio := proceso_en_swap.tamanio

	if !HayEspacio(tamanio) {
		slog.Debug("Debug - (QuitarDeSwap) - No hay espacio en memoria para sacar un proceso de swap")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_OK"))
		return
	}

	file, err := os.Open(config_memo.Path_swap)

	if err != nil {
		slog.Error("Error - (QuitarDeSwap) - Error en abrir swapfile")
		http.Error(w, "Error accediendo al archivo swap", http.StatusBadGateway)
		return
	}

	defer file.Close()

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
	ptr_tpag_del_pid := memo.ptrs_raiz_tpag[pid]
	if config_memo.Cant_niveles == 0 {
		//lock memoria_principal
		memo.EscribirDeSwapAMemoriaPrincipal(pid, tamanio, ptr_tpag_del_pid, w)
		//unlock memoria_principal
		return
	}

	for i := 1; i < config_memo.Cant_niveles; i++ {
		ptr_tpag_del_pid = ptr_tpag_del_pid.sgte_nivel
	}

	//lock memoria_principal
	memo.EscribirDeSwapAMemoriaPrincipal(pid, tamanio, ptr_tpag_del_pid, w)
	//unlock memoria_principal

}

func (memo *Memo) EscribirDeSwapAMemoriaPrincipal(pid int, tamanio int, ptr_tpag_del_pid *NivelTPag, w http.ResponseWriter) {

	memo.AsignarFramesAProceso(ptr_tpag_del_pid, tamanio, pid)

	frames_asignados_a_pid := memo.l_proc[pid].ptr_a_frames_asignados
	var contenido_proc_dividido []byte

	for i := range frames_asignados_a_pid {
		//tema contenido
		inicio_contenido := i * tamanio_pag
		fin_contenido := inicio_contenido + tamanio_pag
		pagina_a_escribir := contenido_proc_dividido[inicio_contenido:fin_contenido]

		//tema memoria
		frame := *frames_asignados_a_pid[i]
		inicio_memo := frame * tamanio_pag
		fin_memo := inicio_memo + tamanio_pag

		copy(memoria_principal[inicio_memo:fin_memo], pagina_a_escribir)
	}
	memo.IncrementarMetrica(pid, Subidas_a_memoria)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (memo *Memo) EliminarProcesoDeSwap(pid int) {

	proceso_en_swap := memo.swap.espacio_contiguo[pid]

	delete(memo.swap.espacio_contiguo, pid)

	nueva_instancia_espacio_libre := &EspacioLibre{
		inicio:  proceso_en_swap.inicio,
		tamanio: proceso_en_swap.tamanio,
	}

	memo.swap.espacio_libre = append(memo.swap.espacio_libre, nueva_instancia_espacio_libre)

}
