package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
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

func HanshakeKernel(w http.ResponseWriter, r *http.Request) {

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (memo *Memo) InicializarTablaPaginas(pid int, tamanio int) {
	nueva_tabla_asociado_al_pid_pasado := memo.tabla_global_nivel0[pid]
	nueva_tabla_asociado_al_pid_pasado = &Tabla{
		nivel_tabla: 0,
		nro_marco:   0,
		offset:      tamanio - 1,
		// El tamanio esta DUDOSISIMO, es mas que nada un ej pero 100% hay que cambiarlo jasdaj
		sgte_tabla: nil,
	}

	if config_memo.Cant_niveles == 0 {
		return
	}

	for i := 1; i <= config_memo.Cant_niveles; i++ {
		nueva_tabla_asociado_al_pid_pasado.sgte_tabla = &Tabla{
			nivel_tabla: i,
			nro_marco:   0,
			offset:      tamanio - 1,
			sgte_tabla:  nil,
		}
	}
	//aca habria algo igual para la cantidad de entradas, que me imagino que es el marco :P

}

//hola soy santi, soy el mejor de todos y para nada fer escribio esto mientras YO, Santi, estaba distraido a la 1:30 AM
//me debes 9 lucas santi paga lo que debes
