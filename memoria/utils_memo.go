package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

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
		fmt.Println("error al recibir fetch de cpu")
	}

	aux := strings.Split(request, " ")
	pid, _ := strconv.Atoi(aux[0])
	pc, _ := strconv.Atoi(aux[1])

	elemento_en_memo_sistema, ok := memo.memoria_sistema[pid]
	if !ok || len(elemento_en_memo_sistema) == 0 {
		fmt.Println("No hay mas instrucciones")
		return
	}

	//para pruebas nomas
	for _, linea_a_leer := range elemento_en_memo_sistema {
		fmt.Println(linea_a_leer)
	}

	instruccion := elemento_en_memo_sistema[pc]
	//memo.memoria_sistema[pid] = elemento_en_memo_sistema[1:]
	// if !ok {
	// 	fmt.Println("No se encontro un proceso")
	// 	return
	// }

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(instruccion))
}

func (memo *Memo) VerificarHayLugar(w http.ResponseWriter, r *http.Request) {
	var request string

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("error al recibir los datos desde kernel")
	}

	aux := strings.Split(request, " ")

	pid, _ := strconv.Atoi(aux[0])
	tamanio, _ := strconv.Atoi(aux[1])
	arch_pseudo := aux[2]

	// Lock tam_memo_actual
	if tamanio >= tam_memo_actual { //config_memo.Tamanio_memoria va a volar, hay que chequear en memoria_usuario
		// Unlock tam_memo_actual
		fmt.Println("No hay espacio suficiente para crear el proceso pedido por kernel")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_OK"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Si kernel, hay espacio"))

	memo.CrearNuevoProceso(pid, tamanio, arch_pseudo)
	// Lock tam_memo_actual
	tam_memo_actual -= tamanio
	// Unlock tam_memo_actual
}

func (memo *Memo) CrearNuevoProceso(pid int, tamanio int, arch_pseudo string) {
	_, ok := memo.memoria_sistema[pid]
	if ok {
		fmt.Println("Ya se encuentra creado un proceso")
		return
	}

	// 1er check, cargar arch pseudo
	nuevo_elemento := CargarArchivoPseudocodigo(arch_pseudo)
	memo.memoria_sistema[pid] = nuevo_elemento

	// 2do check, asignarle frames y crear tablas
	memo.InicializarTablaPunterosAsociadosA(pid, tamanio)

}

func Hanshake(w http.ResponseWriter, r *http.Request) {
	var request string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("error al recibir los datos desde kernel")
		return
	}

	respuesta := fmt.Sprintf("%d %d", config_memo.Cant_entradasXpag, config_memo.Tamanio_pag)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(respuesta))
}

func (memo *Memo) InicializarTablaPunterosAsociadosA(pid int, tamanio int) {
	nuevo_puntero_de_tablas := memo.ptrs_raiz_tpag[pid]
	nuevo_puntero_de_tablas = &NivelTPag{
		lv_tabla:   0,
		sgte_nivel: nil,
	}

	if config_memo.Cant_niveles == 0 {
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

		utils.LoggerConFormato("Se asigno correctamente frames a un proceso")
	}
	utils.LoggerConFormato("No se pudo asignar frames a un proceso")
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
		slog.Error("Se intento asignar mas frames de los que habia diponibles") //poner pid que pide mucho si queres
		fmt.Println("Se intento asignar mas frames de los que habia diponibles, hicimo algo mal")
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

func (memo *Memo) CargarDataSwap(pid int, tamanio int) {
	memo.swap.espacio_contiguo[pid].inicio = memo.swap.ultimo_byte
	memo.swap.espacio_contiguo[pid].tamanio = tamanio
	memo.swap.ultimo_byte += tamanio
}

func (memo *Memo) EscribirEnSwap(pid int) {
	file, err := os.Open(config_memo.Path_swap)

	if err != nil {
		fmt.Println("Error en abrir swapfile")
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
}

func (memo *Memo) QuitarDeSwap(pid int) {
	file, err := os.Open(config_memo.Path_swap)

	if err != nil {
		fmt.Println("Error en abrir swapfile")
		return
	}

	inicio_proceso := memo.swap.espacio_contiguo[pid].inicio
	tamanio := memo.swap.espacio_contiguo[pid].tamanio
	file.Seek(int64(inicio_proceso), 1)
	contenido_proc := make([]byte, tamanio)

	bytes_leidos, _ := file.Read(contenido_proc)

	if bytes_leidos != tamanio { //leyamal yo leyo tu leyes el leye
		fmt.Errorf("no leo bien necesito gafas")

	}

	fmt.Println(string(contenido_proc[:bytes_leidos]))

	file.Seek(0, 0)
	file.Close()
	// compromiso de copactar para el segundo cuatri
	//segurisimo

	ptr_tpag_del_pid := memo.ptrs_raiz_tpag[pid]
	if config_memo.Cant_niveles == 0 {
		memo.AsignarFramesAProceso(ptr_tpag_del_pid, tamanio, pid)
		return
	}

	for i := 1; i < config_memo.Cant_niveles; i++ {
		ptr_tpag_del_pid = ptr_tpag_del_pid.sgte_nivel
	}

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

}
