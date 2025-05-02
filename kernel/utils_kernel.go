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

func crearPcb(pid *int, tamanio int) *PCB {
	pcb := new(PCB)
	pcb.Pid = *pid
	pcb.tamanio = tamanio
	incrementarPid(pid)
	//pcb.Pc = 0
	return pcb
}

//

func incrementarPid(pid *int) {
	*pid++
}

func FIFO(l_estado *[]*PCB, pcb *PCB) { //FIFO
	*l_estado = append(*l_estado, pcb)
}

func cambiarMetricasEstado(pPcb *PCB, posEstado int) {
	pPcb.Me[posEstado]++ //ver si puede quedar mas lindo
}

func cambiarMetricasTiempo(pPcb *PCB, posEstado int) {
	tiempoActual := time.Now()
	(pPcb).Mt[posEstado] = tiempoActual.Sub(pPcb.contador)
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

func iniciarProceso(pid *int, tamanio int, l_new *[]*PCB) {
	pcb := crearPcb(pid, tamanio)
	pcb.contador = time.Now()

	cambiarMetricasEstado(pcb, EstadoNew)
	FIFO(l_new, pcb) //meter en la cola new no hay planificacion para meter en la cola new

}

func planiLargoPlazo(pid *int, tamanio int, archivoPseudo string, l_new *[]*PCB, l_ready *[]*PCB) error { //fijarte si podes hacer que entre a la cola de new y que prg dsp por el sig

	iniciarProceso(pid, tamanio, l_new)
	//log.Info("## (%d) Se crea el proceso - Estado: NEW", pcb.Pid)
	hay_espacio, err := hayEspacio(*pid, tamanio, archivoPseudo)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return err
	}
	if hay_espacio {
		if config_kernel.Ready_ingress_algorithm == "PMCP" || len(*l_new) == 1 {
			// Cambiar de estado del proceso de NEW a READY
			cambiarDeEstado(l_new, l_ready, 0, EstadoReady)
			//log.Info("## (%d) Pasa del estado NEW al estado READY", *pid)
		}
	}
	return nil //dudoso
}

func AsignarCPU(pid int) int {
	
	for id, cpu := range cpuLibres {
		if cpu.Esta_libre {
			cpu.Pid = pid
			cpu.Esta_libre = false
			cpuLibres[id] = cpu
			return id
		}
	}
	return -1
}

func planiCortoPlazo(l_ready *[]*PCB, idCPU int) {
	
	if len(*l_ready) == 0 {
		fmt.Println("No hay procesos en READY")
		return
	}
	// Ordenar la cola READY por estimación o FIFO
	indice := 0 //fifo
	pcb := &l_ready[indice]


	// Cambiar de estado: READY → EXECUTE
	//mutex
	//defer
	cpu_pos := AsignarCPU(pcb.Pid)
	if cpu_pos != -1{
		cambiarDeEstado(&l_ready,&l_execute, indice, EstadoReady)
		
		fmt.Println("Proceso a execute")
	}
	
	fmt.Println("No hay ninguna CPU disponible")
	// Marcar el CPU como ocupado
	
	cpusLibres[idCPU].Esta_libre = false

}

func enviarProcessExe(cpu CPU, proceso *PCB) (string, error) {
	//ENVIAR PROCESO A CPU
	procecssEjec := fmt.Sprintf("%d %d", proceso.Pid, proceso.Pc)
	url := fmt.Sprintf("http://%s:%d/pid", cpu.IP, cpu.Puerto, procecssEjec)

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

func recibirSyscallCPU(w http.ResponseWriter, r *http.Request) []string {
	var respuesta string
	var aux []string //y bueno
	//Lo mas hardcodeado que vi
	if err := json.NewDecoder(r.Body).Decode(&respuesta); err != nil {
		fmt.Errorf("error creando la solicitud: %w", err)
		return aux
	}

	syscall := strings.Split(respuesta, " ")
	return syscall
}

func gestionarSyscalls(syscall []string, pid int, url string) {
	cod_op := syscall[0]

	switch cod_op {
	case "IO":
		//IO AURICULARES 9000
		nombre := syscall[1]
		tiempo, _ := strconv.Atoi(syscall[2])

		manejarIO(nombre, pid, tiempo)
		//manejarIO
		//validar que exista la io
		//enviar mensaje a io

	case "INIC_PROC":
		arch_inst := syscall[1]
		tamanio, _ := strconv.Atoi(syscall[2])

	case "DUMP_MEMORY":
		mensaje_DUMP_MEMORY := fmt.Sprintf("DUMP_MEMORY %d", pid)
		utils.EnviarSolicitudHTTPString("POST", url, mensaje_DUMP_MEMORY)

	case "EXIT":
		//finalizarProc

	}
}

func agregarAColaBlocked(nombre_io string,io IO){
	pcb := _execute
	
	cambiarDeEstado(,,,"")
}

func manejarIO(nombre_io string, pid int, duracion int) error {

	// Validacion de si existe  IO
	ioMutex.RLock()
	io_seleccionada, ok := IOs[nombre_io]
	ioMutex.RUnlock()
	if !ok {
		return fmt.Errorf("la IO %s no está registrada", nombre_io)
	}

	if !io_seleccionada. {

	}

	//agregarACola(blocked)
	datos_a_enviar := fmt.Sprintf("%d %d", pid, duracion)
	fullURL := fmt.Sprintf("%s/io_request", io_seleccionada.Url)                     // Cambiar el formato en io en un futuro
	respuesta, _ := utils.EnviarSolicitudHTTPString("POST", fullURL, datos_a_enviar) // La respuesta es el pid
	pid_aux := strconv.Itoa(pid)

	if respuesta != pid_aux {
		return nil //momentaneo
	}

	//sacarDeBlockAReady(pid)
	return nil

}

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

	url := fmt.Sprintf("http://%s:%s/", aux[1], aux[2])

	nueva_ID_CPU, _ := strconv.Atoi(aux[0])

	nueva_cpu := CPU{
		ID:         nueva_ID_CPU,
		Url:        url,
		Pid:-1,
		Esta_libre: true}

	// Agrego y sincronizo el nuevo CPU
	mutex_cpus_libres.Lock()
	defer mutex_cpus_libres.Unlock()

	cpuLibres[nueva_cpu.ID] = nueva_cpu

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
	// Si no existe una io con ese nombre, lo agrego nuevito

	if _, ok := IOs[nombre_io]; !ok {
		// Agrego y sincronizo el nuevo dispositivo io

		IOs[nombre_io] = &IO{
			Urls:           []string{url},
			CantInstancias: 1,
		}
	} else {
		// Sino, actualizo los valores en esa posicion
		IOs[nombre_io].CantInstancias++
		IOs[nombre_io].Urls = append(IOs[nombre_io].Urls, url)
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
