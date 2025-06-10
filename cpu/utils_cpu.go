package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func fetch(url_memo string) string {

	peticion := fmt.Sprintf("%d %d", *pid_ejecutando, *pc_ejecutando)
	fullUrl := fmt.Sprintf("%s/memoria/fetch", url_memo)

	instruccion, _ := utils.EnviarSolicitudHTTPString("POST", fullUrl, peticion)

	return instruccion
}

func decode(instruccion string) (string, []string) {
	l_instruccion := strings.Split(instruccion, " ")
	cod_op := l_instruccion[0]
	operacion := l_instruccion[1:]

	return cod_op, operacion
}

func execute(cod_op string, operacion []string) {

	pid_string := strconv.Itoa(*pid_ejecutando)
	switch cod_op {

	case "NOOP":
		//consume el tiempo de ciclo de instruccion
		slog.Info("PID: %d - Ejecutando: %s", pid_string, cod_op)

	case "WRITE":
		dir_logica, _ := strconv.Atoi(operacion[0])
		datos := operacion[1]

		slog.Info("## PID: %s - Ejecutando: %s - %s - %s", pid_string, cod_op, operacion[0], datos)

		respuesta, dir_fisica, err := requestWRITE(dir_logica, datos)
		if err != nil {
			hay_interrupcion = true
			*pc_ejecutando++
			return
		}

		utils.LoggerConFormato("PID: %d - Acción: ESCRITURA - Dirección Física: [ %d | %d ] - Valor: %s", *pid_ejecutando, dir_fisica.frame, dir_fisica.offset, respuesta)

	case "READ":
		dir_logica, _ := strconv.Atoi(operacion[0])
		tamanio, _ := strconv.Atoi(operacion[1])

		slog.Info("## PID: %s - Ejecutando: %s - %s - %s", pid_string, cod_op, operacion[0], operacion[1])

		//Gestionar mejor el error :p
		valor_leido, dir_fisica, err := requestREAD(dir_logica, tamanio)
		//si el valor leido es un aviso de direccionamiento invalido
		//habilitar un hay_interrupcion
		if err != nil {
			hay_interrupcion = true
			*pc_ejecutando++
			return
		}
		utils.LoggerConFormato("PID: %d - Acción: LEER - Dirección Física: [ %d | %d ] - Valor: %s", *pid_ejecutando, dir_fisica.frame, dir_fisica.offset, valor_leido)

	case "GOTO":
		slog.Info("PID: %s - Ejecutando: %s", pid_string, cod_op)

		nuevo_pc, _ := strconv.Atoi(operacion[0])
		*pc_ejecutando = nuevo_pc

	// Syscalls
	case "IO":
		// ID_CPU PC IO TECLADO 20000

		mensaje_io := fmt.Sprintf("%s %d IO %s %s", id_cpu, *pc_ejecutando, operacion[0], operacion[1])
		enviarSyscall("IO", mensaje_io)
		hay_interrupcion = true
	case "INIT_PROC":
		// ID_CPU PC INIT_PROC proceso1 256

		mensaje_init_proc := fmt.Sprintf("%s %d INIT_PROC %s %s", id_cpu, *pc_ejecutando, operacion[0], operacion[1])
		enviarSyscall("INIT_PROC", mensaje_init_proc)
		hay_interrupcion = false

	case "DUMP_MEMORY":
		// ID_CPU PC DUMP_MEMORY

		mensaje_dump := fmt.Sprintf("%s %d DUMP_MEMORY", id_cpu, *pc_ejecutando)
		enviarSyscall("DUMP_MEMORY", mensaje_dump)
		hay_interrupcion = true

	case "EXIT":
		// ID_CPU PC DUMP_MEMORY
		hay_interrupcion = true
		mensaje_exit := fmt.Sprintf("%s %d EXIT", id_cpu, *pc_ejecutando)
		enviarSyscall("EXIT", mensaje_exit)

	default:
		fmt.Println("Error, ingrese una instruccion valida")
	}

	// Incrementar PC
	if cod_op != "GOTO" {
		*pc_ejecutando++
	}

}

func recibirInterrupt(w http.ResponseWriter, r *http.Request) {
	var respuesta string

	if err := json.NewDecoder(r.Body).Decode(&respuesta); err != nil {
		fmt.Println("error decodificando la respuesta: ", err) //revisar porque no podemos usar Errorf
		return
	}

	if respuesta == "OK" {
		hay_interrupcion = true
		return
	}
}

func chequarTLBActiva() {
	if config_CPU.Cant_entradas_TLB > 0 {
		tlb_activa = true
	}
}

func chequearCachePagsActiva() {
	if config_CPU.Cant_entradas_cache > 0 {
		cache_pags_activa = true
	}
}

var hola int = 0
