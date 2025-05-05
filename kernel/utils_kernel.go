package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	utils "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

// var cola_susp_block []
// var cola_susp_ready []

// array de arrays que contenga a todas las colas

func iniciarConfiguracionKernel(filePath string) *ConfigKernel {
	var configuracion *ConfigKernel
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&configuracion)

	return configuracion
}

func detenerKernel() {

	fmt.Println("Ingrese ENTER para empezar con la planificacion")

	bufio.NewReader(os.Stdin).ReadBytes('\n')

	fmt.Println("Empezando con la planificacion")

}

func crearPcb(tamanio int) *PCB {
	pcb := new(PCB)
	pcb.Pid = pid
	pcb.tamanio = tamanio
	if config_kernel.Algoritmo_Plani == "" {
		crearSJF(pcb)
	} else {
		pcb.SJF = nil
	}

	incrementarPid()
	//pcb.Pc = 0
	return pcb
}

//

func incrementarPid() {
	pid++
}

func FIFO(l_estado *[]*PCB, pcb *PCB) { //FIFO
	*l_estado = append(*l_estado, pcb)
}

func cambiarMetricasEstado(pPcb *PCB, posEstado int) {
	pPcb.Me[posEstado]++ //ver si puede quedar mas lindo
	pPcb.contador = time.Now()
}

func duracionEnEstado(pPcb *PCB, posEstado int) time.Duration {
	tiempoActual := time.Now()
	return tiempoActual.Sub(pPcb.contador)
}
func cambiarMetricasTiempo(pPcb *PCB, posEstado int) {
	pPcb.Mt[posEstado] += duracionEnEstado(pPcb, posEstado)

}

func cambiarDeEstado(l_est_origen *[]*PCB, l_est_destino *[]*PCB, indice int, estado int) {

	if indice < 0 || indice >= len(*l_est_origen) {
		fmt.Println("Índice fuera de rango")
		return
	}

	// Obtener el elemento
	pcb := (*l_est_origen)[indice]

	fmt.Println("Lista new antes:", l_est_origen)

	// Actualizar métricas
	cambiarMetricasTiempo(pcb, pcb.estado)
	cambiarMetricasEstado(pcb, estado)

	// Cambiar estado del PCB
	pcb.estado = estado

	// Intercambio de PCB entre listas
	*l_est_destino = append(*l_est_destino, pcb)
	*l_est_origen = append((*l_est_origen)[:indice], (*l_est_origen)[indice+1:]...)

	//Log de los cambios
	fmt.Println("Lista new:", l_est_origen)
	fmt.Println("Lista ready:", l_est_destino)
}

//agregarACola(){
//
//}

func crearSJF(pcb *PCB) {
	sjf := new(SJF)
	sjf.Estimado_anterior = config_kernel.Estimacion_Inicial
	sjf.Real_anterior = 0 //no ejecuto valor igual a 0
	pcb.SJF = sjf
}

func actualizarEstimacionSJF(pcb *PCB, tiempoEnExecute time.Duration) {
	if pcb == nil || pcb.SJF == nil {
		return
	}
	realAnterior := float64(tiempoEnExecute)
	alpha := config_kernel.Alfa
	sjf := pcb.SJF

	sjf.Estimado_anterior = alpha*realAnterior + (1-alpha)*sjf.Estimado_anterior
}

func iniciarProceso(tamanio int) {
	pcb := crearPcb(tamanio)
	pcb.contador = time.Now()

	cambiarMetricasEstado(pcb, EstadoNew)
	FIFO(&l_new, pcb) //meter en la cola new no hay planificacion para meter en la cola new

}

func planiLargoPlazo(tamanio int, archivoPseudo string) error { //fijarte si podes hacer que entre a la cola de new y que prg dsp por el sig

	iniciarProceso(tamanio)
	//log.Info("## (%d) Se crea el proceso - Estado: NEW", pcb.Pid)
	hay_espacio, err := hayEspacio(tamanio, archivoPseudo)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return err
	}
	if hay_espacio {
		if config_kernel.Ready_ingress_algorithm == "PMCP" || len(l_new) == 1 {
			// Cambiar de estado del proceso de NEW a READY
			cambiarDeEstado(&l_new, &l_ready, 0, EstadoReady)
			//log.Info("## (%d) Pasa del estado NEW al estado READY", *pid)
		}
	}
	return nil //dudoso
}

func buscarCPU() *CPU {

	for id, cpu := range cpuLibres {
		if cpu.Esta_libre {
			return cpu // La primera CPU que esta libre
		}
	}
	return nil // No hay CPU libre
}
func planiCortoPlazo() {

	if config_kernel.Algoritmo_Plani == "SJF" {
		sort.Sort(PorSJF(l_ready)) //SJF distinto de nil
	}
}
func enviarProcesoAExecute() {

	if len(l_ready) == 0 {
		fmt.Println("No hay procesos en READY")
		return
	}

	planiCortoPlazo()

	//Tomamos el primer PCB tras el sort si era SJF y sino FIFO
	indice := 0
	pcb := (l_ready)[indice]

	// intentamos asignarle cpu
	cpu_seleccionada := buscarCPU()

	if cpu_seleccionada == nil { //no hay cpu libre
		fmt.Printf("No hay cpu libre")
		return
	}

	cpu_seleccionada.Pid = pcb.Pid
	cpu_seleccionada.Pc = pcb.Pc
	handleDispatch(cpu_seleccionada)
	cambiarDeEstado(&l_ready, &l_execute, indice, EstadoReady)

	cpu_seleccionada.Esta_libre = false

	fmt.Printf("Proceso %d a Execute en CPU %d\n", pcb.Pid, cpu_seleccionada.ID)
}

// post /dispach hola soy kernel aca tenes el pid y pc (no el tid) /interrupt sali chvom /syscalll ahola soy cpu termine
// /cpu

func handleDispatch(cpu_seleccionada *CPU) {

	fullURL := fmt.Sprintf("%sdispatch", cpu_seleccionada.Url)

	datos := fmt.Sprintf("%d %d", cpu_seleccionada.Pid, cpu_seleccionada.Pc)
	utils.EnviarSolicitudHTTPString("POST", fullURL, datos)
}

func enviarProcessExe(cpu CPU, proceso *PCB) (string, error) {
	//ENVIAR PROCESO A CPU
	procecssEjec := fmt.Sprintf("%d %d", proceso.Pid, proceso.Pc)
	url := fmt.Sprintf("http://%s/pid", cpu.Url, procecssEjec)

	respuestaMemo, err := http.Get(url)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	bodyByte, err := io.ReadAll(respuestaMemo.Body)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	fmt.Println(string(bodyByte))

	return string(bodyByte), nil
}

/*func ingresarColaNew(pid *int) {
	crearPcb(*pid)

	//inicio := time.Now()
	cola_new = append(cola_new, pcb)

}*/ //obsolteta aparentemente

func modificarEstado(pcb *PCB, pos int) {

	pcb.Me[pos]++

}

func solicitudMemo(pidAPreguntar int, tamanio int, archivoPseudo string) (string, error) {
	preguntaMemo := solicitudIniciarProceso{
		Pid:           pidAPreguntar,
		ArchivoPseudo: archivoPseudo,
		Tamanio:       tamanio,
	}
	url := fmt.Sprintf("http://%s:%d/", config_kernel.Ip_kernel, config_kernel.Puerto_Memoria)
	body, err := json.Marshal(preguntaMemo)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	respuesta, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	var respuestaMemo string
	err = json.NewDecoder(respuesta.Body).Decode(&respuestaMemo)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	return respuestaMemo, nil

}

func solicitudEliminarProceso(pid int) (string, error) {

	url := fmt.Sprintf("http://%s:%d/%d", config_kernel.Ip_kernel, config_kernel.Puerto_Memoria, pid)

	respuestaMemo, err := http.Get(url)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	bodyByte, err := io.ReadAll(respuestaMemo.Body)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	fmt.Println(string(bodyByte))

	return string(bodyByte), nil

}

func hayEspacio(pid int, tamanio int, archivoPseudo string) (bool, error) {

	mensaje, err := solicitudMemo(pid, tamanio, archivoPseudo)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return false, err
	}

	if mensaje == "okay" {
		return true, nil
	}
	return false, nil
	//mux.HandleFunc("/hay_espacio", hayEspacio())

}

func recibirSyscallCPU(w http.ResponseWriter, r *http.Request) {
	var respuesta string

	if err := json.NewDecoder(r.Body).Decode(&respuesta); err != nil {
		fmt.Errorf("error creando la solicitud: %w", err)
		return
	}

	syscall := strings.Split(respuesta, " ")
	gestionarSyscalls(syscall)
}

func gestionarSyscalls(syscall []string) {

	id_cpu, _ := strconv.Atoi(syscall[IdCPU])
	pc, _ := strconv.Atoi(syscall[PC])
	cpu_ejecutando := cpuLibres[id_cpu]
	cod_op := syscall[CodOp]

	switch cod_op {
	case "IO":
		// 2 20 IO AURICULARES 9000

		nombre := syscall[3]
		tiempo, _ := strconv.Atoi(syscall[4])

		manejarIO(nombre, cpu_ejecutando.Pid, tiempo)
		//manejarIO
		//validar que exista la io
		//enviar mensaje a io

	case "INIC_PROC":
		// 2 20 INIT_PROC proceso1 256

		nombre_arch := syscall[3]
		tamanio, _ := strconv.Atoi(syscall[4])
		planiLargoPlazo(tamanio, nombre_arch)
		cpu_ejecutando.Pc = pc //Actualizar pc para cpu
		handleDispatch(cpu_ejecutando)

	case "DUMP_MEMORY":
		// 2 30 DUMP_MEMORY

		mensaje_DUMP_MEMORY := fmt.Sprintf("DUMP_MEMORY %d", cpu_ejecutando.Pid)
		utils.EnviarSolicitudHTTPString("POST", cpu_ejecutando.Url, mensaje_DUMP_MEMORY)

	case "EXIT":
		// 2 EXIT
		//finalizarProc

	}
}

func agregarAColaBlocked(nombre_io string, io IO) {
}

func buscarIOLibre(nombre string) *IO {

	ioMutex.RLock()
	defer ioMutex.RUnlock()

	if iosDispo, ok := ios[nombre]; ok {
		for _, instancia := range iosDispo.io {
			if instancia.Esta_libre {
				return instancia
			}
		}
	}

	return nil

}

func buscarPorPid(lista *[]*PCB, pid int) int {
	for pos, pcb := range *lista {
		if pcb.Pid == pid {
			return pos
		}
	}
	return -1
}

func manejarIO(nombre_io string, pid int, duracion int) {

	io, existeIO := ios[nombre_io]
	if !existeIO {
		return // mandar a exit falta!!
	}
	pos := buscarPorPid(&l_ready, pid)
	cambiarDeEstado(&l_execute, &l_block, pos, EstadoBlock)

	IO_Seleccionada := buscarIOLibre(nombre_io)

	if IO_Seleccionada == nil { //no hay io libre
		io.procEsperandoPorIO = append(io.procEsperandoPorIO, pid)
	}

	IO_Seleccionada.Pid = pid
	//enviar a io
	IO_Seleccionada.Esta_libre = false

	enviarProcesoAIO(IO_Seleccionada, duracion)

	return

}

func enviarProcesoAIO(io_seleccionada *IO, duracion int) {

	fullURL := fmt.Sprintf("%sio/tarea", io_seleccionada.Url)

	datos := fmt.Sprintf("%d %d", io_seleccionada.Pid, duracion)
	utils.EnviarSolicitudHTTPString("POST", fullURL, datos)
}

func recibirRespuestaIO(w http.ResponseWriter, r *http.Request) {
	var respuesta string
	if err := json.NewDecoder(r.Body).Decode(&respuesta); err != nil {
		fmt.Errorf("Error recibiendo la solicitud:%w", err)
		return
	}

	data := strings.Split(respuesta, " ")

	cod_op := data[0]    // cod_op
	nombre_io := data[1] // nombre_io
	pid_io, _ := strconv.Atoi(data[2])

	switch cod_op {
	case "FIN_IO":
		pos := buscarPorPid(&l_block, pid_io)
		cambiarDeEstado(&l_block, &l_ready, pos, EstadoReady)

	case "DESCONEXION":
	}

}

// func sacarProcesoIO()

func conectarNuevaCPU(w http.ResponseWriter, r *http.Request) { // Handshake
	var cpu_string string
	if err := json.NewDecoder(r.Body).Decode(&cpu_string); err != nil {
		fmt.Errorf("Error recibiendo la solicitud:%w", err)
		return
	}

	aux := strings.Split(cpu_string, " ") //ID IP PUERTO

	if len(aux) != 3 {
		http.Error(w, "Formato inválido", http.StatusBadRequest)
		return
	}

	ip := aux[1]
	puerto := aux[2]
	url := fmt.Sprintf("http://%s:%s/", ip, puerto)

	nueva_ID_CPU, err := strconv.Atoi(aux[0])

	if err != nil {
		http.Error(w, "Formato invalido. Esperando: 'ID IP PUERTO'", http.StatusBadRequest)
	}
	nueva_cpu := CPU{
		ID:         nueva_ID_CPU,
		Url:        url,
		Pid:        -1,
		Esta_libre: true}

	// Agrego y sincronizo el nuevo CPU
	mutex_cpus_libres.Lock()
	defer mutex_cpus_libres.Unlock()

	cpuLibres[nueva_cpu.ID] = &nueva_cpu

}

func conectarNuevaIO(w http.ResponseWriter, r *http.Request) { // Handshake
	var io_string string
	if err := json.NewDecoder(r.Body).Decode(&io_string); err != nil {
		fmt.Errorf("Error recibiendo la solicitud:%w", err)
		return
	}

	aux := strings.Split(io_string, " ") // NOMBRE IP PUERTO

	if len(aux) != 3 {
		http.Error(w, "Formato inválido", http.StatusBadRequest)
		return
	}

	nombre_io := aux[0] //Para mas claridad :p
	url := fmt.Sprintf("http://%s:%s/", aux[1], aux[2])

	mutex_ios.Lock()

	defer mutex_ios.Unlock()

	nuevaIO := &IO{
		Url:        url,
		Pid:        -1,
		Esta_libre: true,
	}
	// Si no existe una io con ese nombre, lo agrego nuevito

	if _, ok := ios[nombre_io]; !ok {
		// Agrego y sincronizo el nuevo dispositivo io

		ios[nombre_io] = &IOS{
			io:                 []*IO{nuevaIO},
			procEsperandoPorIO: []int{},
		}
	} else {
		// Sino, actualizo los valores en esa posicion
		ios[nombre_io].io = append(ios[nombre_io].io, nuevaIO)
	}

}

// Ej 3 CPUs
// cpus := []CPU{
// 	{ID: 1, URL: "http://localhost:5001"},
// 	{ID: 2, URL: "http://localhost:5002"},
// 	{ID: 3, URL: "http://localhost:5003"},
// }

// Al principio todas están libres
// for _, cpu := range cpus {
// 	cpusLibres <- cpu
// }
