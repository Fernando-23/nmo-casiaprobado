package main

import (
	"bufio"
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
	path_completo := "/home/utnso/pruebas/" + path
	archivo, err := os.Open(path_completo)

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

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (Fetch) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición fetch", "mensaje", mensaje)

	aux := strings.Split(mensaje, " ")

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

	mutex_memoriaSistema.Lock()
	elemento_en_memo_sistema, ok := memo.memoria_sistema[pid]
	mutex_memoriaSistema.Unlock()

	if !ok || len(elemento_en_memo_sistema) == 0 {
		slog.Debug("No hay mas instrucciones")
		w.Write([]byte("TODO MAL")) //la ultima deberia ser un exit
		return
	}

	//para pruebas nomas
	//==================================================
	//==================================================
	//--------------ESTO LO VAMOS A BORRAR--------------
	//
	//==================================================
	//==================================================
	// for _, linea_a_leer := range elemento_en_memo_sistema {
	// 	slog.Debug(linea_a_leer)
	// }

	instruccion := elemento_en_memo_sistema[pc]

	slog.Debug("Instruccion a enviar",
		"instruccion", instruccion,
	)

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Cant_instr_solicitadas)
	mutex_metricas.Unlock()
	//==================== LOG OBLIGATORIO ====================
	utils.LoggerConFormato("## PID: %d - Obtener instrucción: %d - Instrucción: %s", pid, pc, instruccion)
	//=========================================================

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(instruccion))
}

func (memo *Memo) VerificarHayLugar(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (VerificarHayLugar) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición verificar hay lugar", "mensaje", mensaje)

	aux := strings.Split(mensaje, " ")

	pid, _ := strconv.Atoi(aux[0])
	tamanio, _ := strconv.Atoi(aux[1])
	arch_pseudo := aux[2]

	mutex_tamanioMemoActual.Lock()
	defer mutex_tamanioMemoActual.Unlock()
	if !HayEspacio(tamanio) {

		slog.Debug("No hay espacio suficiente para crear el proceso pedido por kernel")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_OK"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	memo.CrearNuevoProceso(pid, tamanio, arch_pseudo)
	gb_tam_memo_actual -= tamanio

}

func HayEspacio(tamanio int) bool {
	return gb_tam_memo_actual >= tamanio
}

func (memo *Memo) CrearNuevoProceso(pid int, tamanio int, arch_pseudo string) {

	mutex_memoriaSistema.Lock()
	defer mutex_memoriaSistema.Unlock()
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
	mutex_metricas.Lock()
	memo.InicializarMetricasPor(pid)
	mutex_metricas.Unlock()

	utils.LoggerConFormato("PID: %d - Proceso Creado - Tamaño: %d", pid, tamanio)
}
func (memo *Memo) InicializarMetricasPor(pid int) {
	memo.metricas[pid] = make([]int, cant_metricas)
}

func (memo *Memo) Hanshake(w http.ResponseWriter, r *http.Request) {
	var string_modulo string
	body_bytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Error leyendo la solicitud:", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	string_modulo = string(body_bytes)

	memo.ResponderHandshakeA(string_modulo, w)

}

func (memo *Memo) ResponderHandshakeA(modulo string, w http.ResponseWriter) {
	switch modulo {
	case "CPU":
		respuesta := fmt.Sprintf("%d %d %d", memo.config_memo.Cant_niveles, memo.config_memo.Cant_entradasXpag, memo.config_memo.Tamanio_pag)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(respuesta))

	case "Kernel":
		w.Write([]byte("OK"))
	default:
		w.Write([]byte("NO_OK"))
	}

}

func (memo *Memo) InicializarTablaPunterosAsociadosA(pid int, tamanio int) {

	mutex_tablaPaginas.Lock()
	defer mutex_tablaPaginas.Unlock()

	memo.ptrs_raiz_tpag[pid] = &NivelTPag{
		lv_tabla:   0,
		sgte_nivel: nil,
	}

	nuevo_puntero_de_tablas := memo.ptrs_raiz_tpag[pid]

	if memo.config_memo.Cant_niveles == 0 {
		nuevo_puntero_de_tablas.entradas = make([]*int, memo.config_memo.Cant_entradasXpag)
		memo.AsignarFramesAProceso(nuevo_puntero_de_tablas, tamanio, pid)
		return
	}

	for i := 1; i <= memo.config_memo.Cant_niveles; i++ {
		nuevo_puntero_de_tablas.sgte_nivel = &NivelTPag{
			lv_tabla:   i,
			sgte_nivel: nil,
		}

		if i != memo.config_memo.Cant_niveles {
			nuevo_puntero_de_tablas = nuevo_puntero_de_tablas.sgte_nivel
		}
	}

	nuevo_puntero_de_tablas.entradas = make([]*int, memo.config_memo.Cant_entradasXpag)
	memo.AsignarFramesAProceso(nuevo_puntero_de_tablas, tamanio, pid)
}

func (memo *Memo) AsignarFramesAProceso(tpags_final *NivelTPag, tamanio int, pid int) {
	mutex_framesDisponibles.Lock()
	defer mutex_framesDisponibles.Unlock()
	if HayFramesLibresPara(tamanio) {

		frames_a_reservar := LaCuentitaMaestro(tamanio, memo.config_memo.Tamanio_pag)
		mutex_bitmap.Lock()
		memo.ModificarEstadoFrames(frames_a_reservar, pid)
		mutex_bitmap.Unlock()
		slog.Debug("Debug - (AsignarFramesAProceso) -  Se asigno correctamente frames al proceso",
			"pid", pid,
		)
		return
	}
	slog.Error("Error - (AsignarFramesAProceso) - No se pudo asignar frames a un proceso")
}

// Traeme la dolorosa, la juguetona pa
func LaCuentitaMaestro(tamanio_proc int, tamanio_frame int) int {
	la_dolorosa := tamanio_proc / tamanio_frame
	if (tamanio_proc % tamanio_frame) != 0 {
		return la_dolorosa + 1
	} else if tamanio_proc == 0 {
		return 1
	}

	return la_dolorosa
}

func (memo *Memo) ModificarEstadoFrames(frames_a_reservar int, pid int) {

	i := 0
	for frame := 0; frames_a_reservar != 0 && gb_frames_disponibles > 0; frame++ {
		if memo.bitmap[frame] < 0 {
			memo.bitmap[frame] = frame
			memo.l_proc[pid].ptr_a_frames_asignados[i] = &memo.bitmap[frame]
			i++

			frames_a_reservar--
			gb_frames_disponibles--
		}
	}

	if frames_a_reservar != 0 {
		slog.Error("Error - (ModificarEstadoFrames) - Se intento asignar mas frames de los que habia diponibles") //poner pid que pide mucho si queres
		return
	}
}

func HayFramesLibresPara(tamanio int) bool {
	return gb_frames_disponibles > tamanio
}

func (memo *Memo) InicializarTablaFramesGlobal(cant_frames_memo int) {
	for i := 0; i < cant_frames_memo; i++ {
		memo.bitmap[i] = -1 // -1 = libre
	}
}

// Acceso a espacio de usuario
func (memo *Memo) buscarEnTablaAsociadoAProceso(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (buscarEnTablaAsociadoAProceso) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición buscar en tabla asociado a proceso", "mensaje", mensaje)

	aux := strings.Split(mensaje, " ")
	pid, err1 := strconv.Atoi(aux[0])
	nivel_actual_solicitado, err2 := strconv.Atoi(aux[1])
	entrada, err3 := strconv.Atoi(aux[2])

	if err1 != nil || err2 != nil || err3 != nil {
		slog.Error("Error - (buscarEnTablaAsociadoAProceso) - Conversiones a int")
		return
	}

	if nivel_actual_solicitado != memo.config_memo.Cant_niveles {

		memo.HacerRetardo()

		mutex_metricas.Lock()
		memo.IncrementarMetrica(pid, Accesos_a_tpags)
		mutex_metricas.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("SEGUI"))
		return
	}

	mutex_tablaPaginas.Lock()
	tabla_asociada_proceso := memo.ptrs_raiz_tpag[pid]

	for i := 0; i < memo.config_memo.Cant_niveles; i++ {
		tabla_asociada_proceso = tabla_asociada_proceso.sgte_nivel
	}

	respuesta := strconv.Itoa(*tabla_asociada_proceso.entradas[entrada])
	mutex_tablaPaginas.Unlock()

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Accesos_a_tpags)
	mutex_metricas.Unlock()

	memo.HacerRetardo()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(respuesta))
}

func (memo *Memo) HacerRetardo() {
	time.Sleep(time.Duration(memo.config_memo.Delay_memoria) * time.Millisecond)
}

func (memo *Memo) LeerEnMemoria(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (LeerEnMemoria) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición leer en memoria", "mensaje", mensaje)

	aux := strings.Split(mensaje, " ")
	pid, err1 := strconv.Atoi(aux[0])
	frame, err2 := strconv.Atoi(aux[1])
	offset, err3 := strconv.Atoi(aux[2])
	tamanio_a_leer, err4 := strconv.Atoi(aux[3])

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		slog.Error("Error - (LeerEnMemoria) - Conversiones a int")
		return
	}

	mutex_lprocs.Lock()
	if !memo.ConfirmacionFrameMio(pid, frame) {
		mutex_lprocs.Unlock()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_ES_MI_FRAME_PADRE_NUESTRO...AMEN"))
		return
	}
	mutex_lprocs.Unlock()

	base := (frame * memo.config_memo.Tamanio_pag) + offset

	mutex_memoriaPrincipal.Lock()
	contenido_leido := memo.memoria_principal[base:tamanio_a_leer]
	mutex_memoriaPrincipal.Unlock()

	if !memo.SigoEnMiFrame(pid, frame, base, tamanio_a_leer) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ME_PASE_DE_LA_RAYA_AMEN"))
		return
	}

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Cant_read)
	mutex_metricas.Unlock()

	utils.LoggerConFormato("## PID: %d - Lectura - Dir. Física: [ %d |  %d  ] - Tamaño: %d", pid, frame, offset, tamanio_a_leer)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(contenido_leido))
}

func (memo *Memo) EscribirEnMemoria(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (EscribirEnMemoria) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición escribir en memoria", "mensaje", mensaje)

	aux := strings.Split(mensaje, " ")
	pid, err1 := strconv.Atoi(aux[0])
	frame, err2 := strconv.Atoi(aux[1])
	offset, err3 := strconv.Atoi(aux[2])

	if err1 != nil || err2 != nil || err3 != nil {
		slog.Error("Error - (EscribirEnMemoria) - Conversiones a int")
		return
	}

	datos_a_escribir := []byte(aux[3])

	mutex_lprocs.Lock()

	if !memo.ConfirmacionFrameMio(pid, frame) {
		mutex_lprocs.Unlock()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_ES_MI_FRAME_PADRE_NUESTRO...AMEN"))
		return
	}
	mutex_lprocs.Unlock()

	base := (frame * memo.config_memo.Tamanio_pag) + offset
	tamanio_a_escribir := len(datos_a_escribir)

	if !memo.SigoEnMiFrame(pid, frame, base, tamanio_a_escribir) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ME_PASE_DE_LA_RAYA_AMEN"))
		return
	}

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Cant_write)
	mutex_metricas.Unlock()

	mutex_memoriaPrincipal.Lock()
	copy(memo.memoria_principal[base:], datos_a_escribir)
	mutex_memoriaPrincipal.Unlock()

	utils.LoggerConFormato("## PID: %d - Escritura - Dir. Física: [ %d |  %d  ] - Tamaño: %d ", pid, frame, offset, tamanio_a_escribir)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (memo *Memo) ConfirmacionFrameMio(pid int, frame int) bool {
	for i := range memo.l_proc[pid].ptr_a_frames_asignados {
		if *memo.l_proc[pid].ptr_a_frames_asignados[i] == frame {
			return true
		}
	}
	return false

}

func (memo *Memo) SigoEnMiFrame(pid int, frame int, base int, tamanio_a_escribir int) bool {
	fin := base + tamanio_a_escribir
	if memo.config_memo.Tamanio_pag >= fin { //    100 4 96
		return true
	}
	return false
}

func (memo *Memo) DumpMemory(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (DumpMemory) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición dump memory", "mensaje", mensaje)

	pid, err := strconv.Atoi(mensaje)

	if err != nil {
		slog.Error("Error - (DumpMemory) - Conversiones a int")
		return
	}

	timestamp := time.Now().Format(time.RFC3339) //me lo ensenio mi amigo luchin guita facil
	nombre := fmt.Sprintf("%s%d-%s.dmp", memo.config_memo.Path_dump, pid, timestamp)
	file, err := os.Create(nombre)

	if err != nil {
		panic(err)
	}
	defer file.Close()

	mutex_lprocs.Lock()
	defer mutex_lprocs.Unlock()

	frames_asignados_a_pid := memo.l_proc[pid].ptr_a_frames_asignados
	tamanio := memo.l_proc[pid].tamanio

	err = file.Truncate(int64(tamanio))

	if err != nil {
		panic(err)
	}

	mutex_memoriaPrincipal.Lock()
	for i := range frames_asignados_a_pid {
		inicio := *frames_asignados_a_pid[i] * memo.config_memo.Tamanio_pag
		fin_de_pag := inicio + memo.config_memo.Tamanio_pag //Liam vino de rendir funcional mepa

		contenido_pag := memo.memoria_principal[inicio:fin_de_pag]
		file.Write(contenido_pag)
	}

	mutex_memoriaPrincipal.Unlock()

	utils.LoggerConFormato("## PID: %d - Memory Dump solicitado", pid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

}

func (memo *Memo) EliminarProceso(pid int) bool {

	mutex_lprocs.Lock()
	proceso_existe := memo.l_proc[pid]

	if proceso_existe != nil {
		return false
	}

	frames_asignados_a_pid := memo.l_proc[pid].ptr_a_frames_asignados

	/*if frames_asignados_a_pid == nil {
		slog.Error("hola roberta")
		return
	}*/
	for i := range frames_asignados_a_pid {
		ptr_frame_asignado := frames_asignados_a_pid[i]
		*ptr_frame_asignado = -1
	}
	mutex_lprocs.Unlock()

	memo.EliminarProcesoDeSwap(pid)

	return true
}

func (memo *Memo) FinalizarProceso(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (FinalizarProceso) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición finalizar proceso", "mensaje", mensaje)

	pid, err := strconv.Atoi(mensaje)

	if err != nil {
		slog.Error("Error - (FinalizarProceso) - Conversiones a int")
		return
	}

	eliminado_correctamente := memo.EliminarProceso(pid)

	if !eliminado_correctamente {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_OK"))
		return
	}

	mutex_metricas.Lock()
	defer mutex_metricas.Unlock()

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
