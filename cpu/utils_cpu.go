package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

type ConfigCPU struct {
	Puerto_CPU      int    `json:"port_cpu"`
	Ip_CPU          string `json:"ip_cpu"`
	Ip_Memoria      string `json:"ip_memory"`
	Puerto_Memoria  int    `json:"port_memory"`
	Ip_Kernel       string `json:"ip_kernel"`
	Puerto_Kernel   int    `json:"port_kernel"`
	Entrada_TLB     int    `json:"tlb_entries"`
	Reemplazo_TLB   string `json:"tlb_replacement"`
	Entrada_Cache   int    `json:"cache_entries"`
	Reemplazo_Cache string `json:"cache_replacement"`
	Delay_Cache     int    `json:"cache_delay"`
	Log_level       string `json:"log_level"`
}

var (
	config_CPU       *ConfigCPU
	url_cpu          string
	url_kernel       string
	url_memo         string
	hay_interrupcion *bool
	id_cpu           string
	pid_ejecutando   *int
	pc_ejecutando    *int
	sem_datos_kernel sync.Mutex
)

func iniciarConfiguracionIO(filePath string) *ConfigCPU {
	var configuracion *ConfigCPU
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&configuracion)

	return configuracion
}

func fetch() string {

	peticion := fmt.Sprintf("%d %d", *pid_ejecutando, *pc_ejecutando)
	fullUrl := fmt.Sprintf("http://%s/memoria/fetch", url_memo)

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
		direccion, _ := strconv.Atoi(operacion[0])
		datos := operacion[1]

		slog.Info("## PID: %s - Ejecutando: %s - %s - %s", pid_string, cod_op, operacion[0], datos)

		respuesta, _ := requestWRITE(direccion, datos)
		fmt.Println(respuesta)

		slog.Info("PID: " + pid_string + " - Acción: ESCRIBIR - Dirección Física: " + operacion[0] + " - Valor: " + datos)

	case "READ":
		direccion, _ := strconv.Atoi(operacion[0])
		tamanio, _ := strconv.Atoi(operacion[1])

		slog.Info("## PID: %s - Ejecutando: %s - %s - %s", pid_string, cod_op, operacion[0], operacion[1])

		//Gestionar mejor el error :p
		valor_leido, _ := requestREAD(direccion, tamanio)
		slog.Info("PID: " + pid_string + " - Acción: LEER - Dirección Física: " + operacion[0] + " - Valor: " + valor_leido)

	case "GOTO":
		slog.Info("PID: %s - Ejecutando: %s", pid_string, cod_op)

		nuevo_pc, _ := strconv.Atoi(operacion[0])
		*pc_ejecutando = nuevo_pc

	// Syscalls
	case "IO":
		// ID_CPU PC IO TECLADO 20000

		mensaje_io := fmt.Sprintf("%s %d IO %s %s", id_cpu, *pc_ejecutando, operacion[0], operacion[1])
		enviarSyscall("IO", mensaje_io)
		*hay_interrupcion = true
	case "INIT_PROC":
		// ID_CPU PC INIT_PROC proceso1 256

		mensaje_init_proc := fmt.Sprintf("%s %d INIT_PROC %s %s", id_cpu, *pc_ejecutando, operacion[0], operacion[1])
		enviarSyscall("INIT_PROC", mensaje_init_proc)
		*hay_interrupcion = true

	case "DUMP_MEMORY":
		// ID_CPU PC DUMP_MEMORY

		mensaje_dump := fmt.Sprintf("%s %d DUMP_MEMORY", id_cpu, *pc_ejecutando)
		enviarSyscall("DUMP_MEMORY", mensaje_dump)
		*hay_interrupcion = true

	case "EXIT":
		// ID_CPU PC DUMP_MEMORY

		mensaje_exit := fmt.Sprintf("%s %d EXIT", id_cpu, *pc_ejecutando)
		enviarSyscall("EXIT", mensaje_exit)
		*hay_interrupcion = true

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
		*hay_interrupcion = true
		return
	}

}
