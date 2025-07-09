package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func CargarArchivoPseudocodigo(path string) []string {
	archivo, err := os.Open(path)

	if err != nil {
		fmt.Println(err)
	}
	defer archivo.Close()

	var nuevas_instr_pid []string

	scaner := bufio.NewScanner(archivo)
	scaner.Split(bufio.ScanLines)

	for scaner.Scan() {
		nuevas_instr_pid = append(nuevas_instr_pid, scaner.Text())
	}

	return nuevas_instr_pid

}

func (memo *Memo) Fetch(w http.ResponseWriter, r *http.Request) {
	var request string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		slog.Error("Error - (Fetch) - Recibiendo fetch de cpu")
	}

	aux := strings.Split(request, " ")

	pid, err := strconv.Atoi(aux[0])
	if err != nil {
		slog.Error("Error - (Fetch) - Transformando pid",
			"pid", pid)
		w.Write([]byte("TODO MAL"))
		return
	}

	pc, err := strconv.Atoi(aux[1])
	if err != nil {
		slog.Error("Error - (Fetch) - Transformando pc",
			"pc", pc)
		w.Write([]byte("TODO MAL"))
		return
	}

	elemento_en_memo_sistema, ok := memo.memoria_sistema[pid]

	if !ok || len(elemento_en_memo_sistema) == 0 {
		slog.Debug("No hay mas instrucciones")
		w.Write([]byte("TODO MAL")) //la ultima deberia ser un exit
		return
	}

	//para pruebas nomas
	for _, linea_a_leer := range elemento_en_memo_sistema {
		slog.Debug(linea_a_leer)
	}

	instruccion := elemento_en_memo_sistema[pc]

	slog.Debug("Instruccion a enviar",
		"instruccion", instruccion,
	)

	memo.IncrementarMetrica(pid, Cant_instr_solicitadas)
	utils.LoggerConFormato("## PID: %d - Obtener instrucción: %d - Instrucción: %s", pid, pc, instruccion)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(instruccion))
}

func (memo *Memo) VerificarHayLugar(w http.ResponseWriter, r *http.Request) {
	var request string

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		slog.Error("Error - (VerificarHayLugar) - Solicitud invalida")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	aux := strings.Split(request, " ")

	pid, _ := strconv.Atoi(aux[0])
	tamanio, _ := strconv.Atoi(aux[1])
	arch_pseudo := aux[2]

	// Lock tam_memo_actual
	if !HayEspacio(tamanio) {
		// Unlock tam_memo_actual
		slog.Debug("No hay espacio suficiente para crear el proceso pedido por kernel")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_OK"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	memo.CrearNuevoProceso(pid, tamanio, arch_pseudo)
	// Lock tam_memo_actual
	tam_memo_actual -= tamanio
	// Unlock tam_memo_actual
}

func HayEspacio(tamanio int) bool {
	return tamanio >= tam_memo_actual
}

func (memo *Memo) CrearNuevoProceso(pid int, tamanio int, arch_pseudo string) {
	_, ok := memo.memoria_sistema[pid]
	if ok {
		slog.Error("Error - (CrearNuevoProceso) - Ya se encuentra creado este proceso",
			"pid", pid)
		return
	}

	// 1er check, cargar arch pseudo
	nuevo_elemento := CargarArchivoPseudocodigo(arch_pseudo)
	memo.memoria_sistema[pid] = nuevo_elemento

	// 2do check, asignarle frames y crear tablas
	memo.InicializarTablaPunterosAsociadosA(pid, tamanio)

	// 3er check, inicializar metrica
	memo.InicializarMetricasPor(pid)

	utils.LoggerConFormato("PID: %d - Proceso Creado - Tamaño: %d", pid, tamanio)
}
func (memo *Memo) InicializarMetricasPor(pid int) {
	memo.metricas[pid] = make([]int, cant_metricas)
}

func Hanshake(w http.ResponseWriter, r *http.Request) {
	var string_modulo string
	body_bytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Error leyendo la solicitud:", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}

	string_modulo = string(body_bytes)

	responderHandshakeA(string_modulo, w)

}

func responderHandshakeA(modulo string, w http.ResponseWriter) {
	switch modulo {
	case "CPU":
		respuesta := fmt.Sprintf("%d %d %d", config_memo.Cant_niveles, config_memo.Cant_entradasXpag, config_memo.Tamanio_pag)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(respuesta))

	case "Kernel":
		w.Write([]byte("OK"))
	default:
		w.Write([]byte("NO_OK"))
	}

}

func (memo *Memo) InicializarTablaPunterosAsociadosA(pid int, tamanio int) {

	memo.ptrs_raiz_tpag[pid] = &NivelTPag{
		lv_tabla:   0,
		sgte_nivel: nil,
	}

	nuevo_puntero_de_tablas := memo.ptrs_raiz_tpag[pid]

	if config_memo.Cant_niveles == 0 {
		nuevo_puntero_de_tablas.entradas = make([]*int, config_memo.Cant_entradasXpag)
		memo.AsignarFramesAProceso(nuevo_puntero_de_tablas, tamanio, pid)
		return
	}

	for i := 1; i <= config_memo.Cant_niveles; i++ {
		nuevo_puntero_de_tablas.sgte_nivel = &NivelTPag{
			lv_tabla:   i,
			sgte_nivel: nil,
		}

		if i != config_memo.Cant_niveles {
			nuevo_puntero_de_tablas = nuevo_puntero_de_tablas.sgte_nivel
		}
	}

	nuevo_puntero_de_tablas.entradas = make([]*int, config_memo.Cant_entradasXpag)
	memo.AsignarFramesAProceso(nuevo_puntero_de_tablas, tamanio, pid)
}

func (memo *Memo) AsignarFramesAProceso(tpags_final *NivelTPag, tamanio int, pid int) {
	if memo.HayFramesLibresPara(tamanio) {
		frames_a_reservar := memo.LaCuentitaMaestro(tamanio, config_memo.Tamanio_pag)
		memo.ModificarEstadoFrames(frames_a_reservar, pid)

		slog.Debug("Debug - (AsignarFramesAProceso) -  Se asigno correctamente frames al proceso",
			"pid", pid,
		)
	}
	slog.Error("Error - (AsignarFramesAProceso) - No se pudo asignar frames a un proceso")
}

// Traeme la dolorosa, la juguetona pa
func (memo *Memo) LaCuentitaMaestro(tamanio_proc int, tamanio_frame int) int {
	la_dolorosa := tamanio_proc / tamanio_frame
	if (tamanio_proc % tamanio_frame) != 0 {
		return la_dolorosa + 1
	}

	return la_dolorosa
}

func (memo *Memo) ModificarEstadoFrames(frames_a_reservar int, pid int) {

	i := 0
	for frame := 0; frames_a_reservar != 0 && frames_disponibles > 0; frame++ {
		if memo.tabla_frames[frame] < 0 {
			memo.tabla_frames[frame] = frame
			memo.l_proc[pid].ptr_a_frames_asignados[i] = &memo.tabla_frames[frame]
			i++

			frames_a_reservar--
			frames_disponibles--
		}
	}

	if frames_a_reservar != 0 {
		slog.Error("Error - (ModificarEstadoFrames) - Se intento asignar mas frames de los que habia diponibles") //poner pid que pide mucho si queres
		return
	}
}

func (memo *Memo) HayFramesLibresPara(tamanio int) bool {
	return frames_disponibles > tamanio
}

func (memo *Memo) CrearSwapfile() {
	_, err := os.Create(config_memo.Path_swap)

	if err != nil {
		fmt.Println("Error en crear swapfile")
		return
	}
}

func (memo *Memo) InicializarTablaFramesGlobal(cant_frames_memo int) {
	for i := range cant_frames_memo {
		memo.tabla_frames[i] = -1
	}
}

// Acceso a espacio de usuario
func (memo *Memo) buscarEnTablaAsociadoAProceso(w http.ResponseWriter, r *http.Request) {
	var request string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("error al recibir peticion de busqueda en tpags")
	}

	aux := strings.Split(request, " ")
	pid, _ := strconv.Atoi(aux[0])
	nivel_actual_solicitado, _ := strconv.Atoi(aux[1])
	entrada, _ := strconv.Atoi(aux[2])

	if nivel_actual_solicitado != config_memo.Cant_niveles {
		hacerRetardo()

		memo.IncrementarMetrica(pid, Accesos_a_tpags)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("SEGUI"))
		return
	}

	tabla_asociada_proceso := memo.ptrs_raiz_tpag[pid]

	for range config_memo.Cant_niveles {
		tabla_asociada_proceso = tabla_asociada_proceso.sgte_nivel
	}

	memo.IncrementarMetrica(pid, Accesos_a_tpags)

	respuesta := strconv.Itoa(*tabla_asociada_proceso.entradas[entrada])

	hacerRetardo()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(respuesta))
}

func hacerRetardo() {
	time.Sleep(time.Duration(config_memo.Delay_memoria) * time.Millisecond)
}

func (memo *Memo) LeerEnMemoria(w http.ResponseWriter, r *http.Request) {
	var request string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("error al recibir peticion de READ")
	}

	aux := strings.Split(request, " ")
	pid, _ := strconv.Atoi(aux[0])
	frame, _ := strconv.Atoi(aux[1])
	offset, _ := strconv.Atoi(aux[2])
	tamanio_a_leer, _ := strconv.Atoi(aux[3])

	if !memo.ConfirmacionFrameMio(pid, frame) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_ES_MI_FRAME_PADRE_NUESTRO...AMEN"))
		return
	}

	base := (frame * config_memo.Tamanio_pag) + offset
	contenido_leido := memoria_principal[base:tamanio_a_leer]
	if !memo.SigoEnMiFrame(pid, frame, base, tamanio_a_leer) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ME_PASE_DE_LA_RAYA_AMEN"))
		return
	}

	memo.IncrementarMetrica(pid, Cant_read)

	utils.LoggerConFormato("## PID: %d - Lectura - Dir. Física: [ %d |  %d  ] - Tamaño: %d", pid, frame, offset, tamanio_a_leer)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(contenido_leido))
}

func (memo *Memo) EscribirEnMemoria(w http.ResponseWriter, r *http.Request) {
	var request string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("error al recibir peticion de WRITE")
	}

	aux := strings.Split(request, " ")
	pid, _ := strconv.Atoi(aux[0])
	frame, _ := strconv.Atoi(aux[1])
	offset, _ := strconv.Atoi(aux[2])
	datos_a_escribir := []byte(aux[3])

	if !memo.ConfirmacionFrameMio(pid, frame) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_ES_MI_FRAME_PADRE_NUESTRO...AMEN"))
		return
	}

	base := (frame * config_memo.Tamanio_pag) + offset
	tamanio_a_escribir := len(datos_a_escribir)

	if !memo.SigoEnMiFrame(pid, frame, base, tamanio_a_escribir) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ME_PASE_DE_LA_RAYA_AMEN"))
		return
	}

	memo.IncrementarMetrica(pid, Cant_write)
	copy(memoria_principal[base:], datos_a_escribir)
	utils.LoggerConFormato("## PID: %d - Escritura - Dir. Física: [ %d |  %d  ] - Tamaño: %d ", pid, frame, offset, tamanio_a_escribir)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (memo *Memo) ConfirmacionFrameMio(pid int, frame int) bool {
	cant_frames_asignados_a_pid := len(memo.l_proc[pid].ptr_a_frames_asignados)

	for i := range cant_frames_asignados_a_pid {
		if *memo.l_proc[pid].ptr_a_frames_asignados[i] == frame {
			return true
		}
	}
	return false

}

func (memo *Memo) SigoEnMiFrame(pid int, frame int, base int, tamanio_a_escribir int) bool {
	fin := base + tamanio_a_escribir
	if config_memo.Tamanio_pag >= fin { //    100 4 96
		return true
	}
	return false
}

func (memo *Memo) DumpMemory(w http.ResponseWriter, r *http.Request) {
	var request string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("error al recibir peticion de WRITE")
	}

	pid, _ := strconv.Atoi(request)

	timestamp := time.Now().Format(time.RFC3339) //me lo ensenio mi amigo luchin guita facil
	nombre := fmt.Sprintf("%s%d-%s.dmp", config_memo.Path_dump, pid, timestamp)
	file, err := os.Create(nombre)

	if err != nil {
		panic(err)
	}
	defer file.Close()

	frames_asignados_a_pid := memo.l_proc[pid].ptr_a_frames_asignados
	tamanio := memo.l_proc[pid].tamanio

	err = file.Truncate(int64(tamanio))

	if err != nil {
		panic(err)
	}

	for i := range frames_asignados_a_pid {
		inicio := *frames_asignados_a_pid[i] * config_memo.Tamanio_pag
		fin_de_pag := inicio + config_memo.Tamanio_pag //Liam vino de rendir funcional mepa

		contenido_pag := memoria_principal[inicio:fin_de_pag]
		file.Write(contenido_pag)
	}

	utils.LoggerConFormato("## PID: %d - Memory Dump solicitado", pid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

}

func (memo *Memo) EliminarProceso(pid int) bool {
	proceso_existe := memo.l_proc[pid]

	if proceso_existe != nil {
		return false
	}

	frames_asignados_a_pid := memo.l_proc[pid].ptr_a_frames_asignados
	for i := range frames_asignados_a_pid {
		ptr_frame_asignado := frames_asignados_a_pid[i]
		*ptr_frame_asignado = -1
	}

	memo.EliminarProcesoDeSwap(pid)

	return true
}

func (memo *Memo) FinalizarProceso(w http.ResponseWriter, r *http.Request) {
	var request string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		slog.Error("Error - (FinalizarProceso) - Error al recibir peticion de WRITE")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pid, _ := strconv.Atoi(request)

	eliminado_correctamente := memo.EliminarProceso(pid)

	if !eliminado_correctamente {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_OK"))
		return
	}

	mt_a_log := memo.metricas[pid]
	utils.LoggerConFormato(
		"## PID: %d - Proceso Destruido - Métricas - Acc.T.Pag: %d; Inst.Sol.: %d; SWAP: %d; Mem.Prin.: %d; Lec.Mem.: %d; Esc.Mem.: %d",
		pid, mt_a_log[Accesos_a_tpags], mt_a_log[Cant_instr_solicitadas], mt_a_log[Bajadas_de_swap],
		mt_a_log[Subidas_a_memoria], mt_a_log[Cant_read], mt_a_log[Cant_write])
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (memo *Memo) IncrementarMetrica(pid int, cod_metrica int) {
	memo.metricas[pid][cod_metrica]++
}
