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

	if tamanio >= config_memo.Tamanio_memoria { //config_memo.Tamanio_memoria va a volar, hay que chequear en memoria_usuario
		fmt.Println("No hay espacio suficiente para crear el proceso pedido por kernel")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_OK"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Si kernel, hay espacio"))

	memo.CrearNuevoProceso(pid, arch_pseudo)
	tam_memo_actual -= tamanio
}

func (memo *Memo) CrearNuevoProceso(pid int, arch_pseudo string) {
	_, ok := memo.memoria_sistema[pid]
	if ok {
		fmt.Println("Ya se encuentra creado un proceso")
		return
	}

	nuevo_elemento := CargarArchivoPseudocodigo(arch_pseudo)
	memo.memoria_sistema[pid] = nuevo_elemento
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
		return
	}

	for i := 1; i < config_memo.Cant_niveles; i++ {
		nuevo_puntero_de_tablas.sgte_nivel = &NivelTPag{
			lv_tabla:   i,
			sgte_nivel: nil,
		}
	}

	nuevo_puntero_de_tablas.sgte_nivel = &NivelTPag{
		lv_tabla:   config_memo.Cant_niveles,
		sgte_nivel: nil,
	}

	memo.AsignarFramesAProceso(nuevo_puntero_de_tablas, tamanio, pid)
}

func (memo *Memo) AsignarFramesAProceso(tpags_final *NivelTPag, tamanio int, pid int) {
	if memo.HayFramesLibresPara(tamanio) {
		tpags_final.entradas = make([]*int, config_memo.Cant_entradasXpag)
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

	for marcos := 0; frames_a_reservar != 0 && frames_disponibles > 0; marcos++ {
		if memo.tabla_frames[marcos] < 0 {
			memo.tabla_frames[marcos] = pid
			frames_a_reservar--
			frames_disponibles--
		}

		if frames_a_reservar != 0 {
			slog.Error("Se intento asignar mas frames de los que habia diponibles") //poner pid que pide mucho si queres
		}
	}
}

func (memo *Memo) HayFramesLibresPara(tamanio int) bool {
	return frames_disponibles > tamanio
}
