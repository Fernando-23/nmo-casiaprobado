package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func iniciarConfiguracionMemo(filePath string) *ConfigMemo {
	var configuracion *ConfigMemo
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&configuracion)

	return configuracion
}

func CargarArchivoPseudocodigo(path string) *[]string {
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

	return &nuevas_instr_pid

}

func Fetch(w http.ResponseWriter, r *http.Request) {
	var request string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("error al recibir fetch de cpu")
	}

	aux := strings.Split(request, " ")
	pid, _ := strconv.Atoi(aux[0])
	pc, _ := strconv.Atoi(aux[1])

	elemento_en_memo_sistema, ok := memoria_sistema[pid]

	if !ok {
		fmt.Println("No se encontro un proceso")
		return
	}

	linea_instruccion := elemento_en_memo_sistema[pc]
	if linea_instruccion == "" {
		fmt.Println("El proceso se encontro en memoria del sistema, pero no tiene ninguna instruccion")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(linea_instruccion))
}

func VerificarHayLugar(w http.ResponseWriter, r *http.Request) {
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

	CrearNuevoProceso(pid, arch_pseudo)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

}
