package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	utils "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

// array de arrays que contenga a todas las colas

func IniciarConfiguracion[T any](ruta string, estructuraDeConfig *T) error {
	configFile, err := os.Open(ruta)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de configuracion: %w", err)
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	if err := jsonParser.Decode(estructuraDeConfig); err != nil {
		return fmt.Errorf("error al decodificar la configuracion %w", err)
	}
	return nil

}

func detenerKernel() {

	fmt.Println("Ingrese ENTER para empezar con la planificacion")

	bufio.NewReader(os.Stdin).ReadBytes('\n')

	fmt.Println("Empezando con la planificacion")

}

func (k *Kernel) InicializarEstados() {
	k.procesoPorEstado = make(map[int][]*PCB)

	// Inicializamos todos los estados del map
	for i := 0; i <= EstadoExit; i++ {
		k.procesoPorEstado[i] = []*PCB{}
	}
}

func (k *Kernel) HandshakeMemoria() error {
	url := fmt.Sprintf("http://%s:%d/memoria/handshake", k.ConfigKernel.Ip_memoria, k.ConfigKernel.Puerto_Memoria)
	respuesta, err := utils.EnviarSolicitudHTTPString("GET", url, nil)
	if err != nil {
		return fmt.Errorf("memoria no responde: %w", err)
	}

	if respuesta != "OK" {
		return fmt.Errorf("respuesta inesperada de memoria: %s", respuesta)
	}

	return nil
}

//en memo
/*
func handlecheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
*/
func (k *Kernel) CrearPCB(tamanio int, arch_pseudo string) *PCB {
	pcb := &PCB{
		Pid:         k.pidActual,
		Tamanio:     tamanio,
		Arch_pseudo: arch_pseudo,
	}

	if k.ConfigKernel.Algoritmo_Plani == "SJF" {
		k.CrearSJF(pcb)
	} else {
		pcb.SJF = nil
	}

	k.pidActual++
	//pcb.Pc = 0
	return pcb
}

func FIFO(l_estado *[]*PCB, pcb *PCB) { //FIFO
	*l_estado = append(*l_estado, pcb)
}

func cambiarMetricasEstado(pPcb *PCB, posEstado int) {
	pPcb.Me[posEstado]++ //ver si puede quedar mas lindo
	pPcb.contador = time.Now()
}

func duracionEnEstado(pPcb *PCB) time.Duration {
	tiempoActual := time.Now()
	return tiempoActual.Sub(pPcb.contador)
}

func cambiarMetricasTiempo(pPcb *PCB, posEstado int) {
	pPcb.Mt[posEstado] += duracionEnEstado(pPcb)

}

func (k *Kernel) AgregarAEstado(estado int, pcb *PCB) {
	k.procesoPorEstado[estado] = append(k.procesoPorEstado[estado], pcb)
}

func (k *Kernel) QuitarDeEstado(estado int, pid int) {
	procesos := k.procesoPorEstado[estado]
	for i, pcb := range procesos {
		if pcb.Pid == pid {
			// Quitar el proceso de la lista del estado
			k.procesoPorEstado[estado] = append(procesos[:i], procesos[i+1:]...)
			break
		}
	}
}

func (k *Kernel) MoverDeEstadoPorPid(estadoActual, estadoNuevo int, pid int) {
	// Buscar el puntero al PCB en el estado actual
	procesos := k.procesoPorEstado[estadoActual]
	var pcb *PCB
	encontrado := false
	for i, p := range procesos {
		if p.Pid == pid {
			pcb = p
			// Eliminar el puntero del estado actual
			k.procesoPorEstado[estadoActual] = append(procesos[:i], procesos[i+1:]...)
			encontrado = true
			break
		}
	}

	if !encontrado {
		fmt.Printf("Proceso %d no encontrado en el estado %d\n", pid, estadoActual)
		return
	}

	// Actualizar metricas
	cambiarMetricasTiempo(pcb, pcb.estado)
	cambiarMetricasEstado(pcb, estadoNuevo)

	// Cambiar el estado del proceso
	pcb.estado = estadoNuevo

	// Agregar el puntero al PCB al nuevo estado
	k.AgregarAEstado(estadoNuevo, pcb)
}

func (k *Kernel) QuitarPrimerElemento(estado int) *PCB {
	pcbs := k.procesoPorEstado[estado]
	primer_elemento := pcbs[0]
	k.procesoPorEstado[estado] = pcbs[1:]

	return primer_elemento

}

func (k *Kernel) MoverDeEstado(estadoActual, estadoNuevo int) {
	// Buscar el puntero al PCB en el estado actual
	pcbs := k.procesoPorEstado[estadoActual]
	if len(pcbs) == 0 {
		fmt.Printf("No hay procesos en el estado %d\n", estadoActual)
		return
	}
	encontrado := false
	primera_pos := 0
	pcb_a_mover := pcbs[primera_pos]
	// Elimina el pcb del estado actual
	k.procesoPorEstado[estadoActual] = pcbs[1:]
	encontrado = true

	if !encontrado {
		fmt.Printf("Proceso %d no encontrado en el estado %d\n", pcb_a_mover.Pid, estadoActual)
		return
	}

	// Actualizar métricas
	cambiarMetricasTiempo(pcb_a_mover, pcb_a_mover.estado)
	cambiarMetricasEstado(pcb_a_mover, estadoNuevo)

	// Cambiar el estado del proceso
	pcb_a_mover.estado = estadoNuevo

	// Agregar el puntero al PCB al nuevo estado
	k.AgregarAEstado(estadoNuevo, pcb_a_mover)
}

func (k *Kernel) CrearSJF(pcb *PCB) {
	sjf := &SJF{
		Estimado_anterior: k.ConfigKernel.Estimacion_Inicial,
		Estimado_actual:   k.ConfigKernel.Estimacion_Inicial,
		Real_anterior:     0, //no ejecuto valor igual a 0
	}
	pcb.SJF = sjf
}

func (k *Kernel) actualizarEstimacionSJF(pcb *PCB, tiempoEnExecute time.Duration) {
	if pcb == nil || pcb.SJF == nil {
		return
	}
	real_anterior := float64(tiempoEnExecute.Milliseconds())
	alpha := k.ConfigKernel.Alfa
	sjf := pcb.SJF
	aux := sjf.Estimado_actual
	sjf.Estimado_actual = (alpha * real_anterior) + ((1 - alpha) * sjf.Estimado_anterior)
	sjf.Estimado_anterior = aux
}

func (k *Kernel) IniciarProceso(tamanio int, arch_pseudo string) *PCB {
	pcb := k.CrearPCB(tamanio, arch_pseudo)
	pcb.contador = time.Now()

	cambiarMetricasEstado(pcb, EstadoNew)
	//k.AgregarAEstado(EstadoNew, pcb) //meter en la cola new no hay planificacion para meter en la cola new
	return pcb
}

func (k *Kernel) ListaNewSoloYo() (bool, error) {
	lista_new := k.procesoPorEstado[EstadoNew]
	if len(lista_new) == 1 {
		fmt.Println("Soy el primer elemento", lista_new[0].Pid)
		primer_elemento := lista_new[0]
		hay_espacio, err := k.MemoHayEspacio(primer_elemento.Pid, primer_elemento.Tamanio, primer_elemento.Arch_pseudo)
		if err != nil {
			log.Printf("Error codificando mensaje: %s", err.Error())
			return true, err
		}
		if hay_espacio {
			// Cambiar de estado del proceso de NEW a READY
			k.MoverDeEstado(EstadoNew, EstadoReady)

			//log.Info("## (%d) Pasa del estado NEW al estado READY", *pid)
		}
		return true, nil
	}
	return false, nil
}

func (k *Kernel) PlanificarLargoPorLista(codLista int) (bool, error) {
	if (k.procesoPorEstado[codLista]) != nil {
		if k.ConfigKernel.Ready_ingress_algorithm == "PCMP" {
			sort.Sort(PorTamanio(k.procesoPorEstado[codLista]))
		}
		pcb := k.procesoPorEstado[codLista][0]
		hay_espacio, err := k.MemoHayEspacio(pcb.Pid, pcb.Tamanio, pcb.Arch_pseudo)

		if err != nil {
			log.Printf("Error codificando mensaje: %s", err.Error())
			return true, err
		}

		if hay_espacio {
			// Cambiar de estado del proceso de NEW a READY
			k.MoverDeEstado(codLista, EstadoReady)

			//log.Info("## (%d) Pasa del estado NEW al estado READY", *pid)
		}
		return true, nil

	}
	fmt.Println("No hay elementos en Ready Suspended")
	return false, nil
}

func (k *Kernel) PlaniLargoPlazo() error {
	//fijarte si podes hacer que entre a la cola de new y que prg dsp por el sig
	// LSR: Lista Suspendido Ready
	hay_elementosLSR, err := k.PlanificarLargoPorLista(EstadoReadySuspended)
	if err != nil {
		return err
	}

	if !hay_elementosLSR {
		_, err := k.PlanificarLargoPorLista(EstadoNew)

		if err != nil {
			return err
		}
	}

	return nil
}

func (k *Kernel) BolicheMomento(pcb_creado *PCB) { //Plani largo plazo para procesos recien creados

	if k.ConfigKernel.Ready_ingress_algorithm == "PMCP" || len(k.procesoPorEstado[EstadoNew]) == 1 { //Si es PMCP o soy el primero
		hay_espacio, _ := k.MemoHayEspacio(pcb_creado.Pid, pcb_creado.Tamanio, pcb_creado.Arch_pseudo)
		if hay_espacio {
			k.MoverDeEstadoPorPid(EstadoNew, EstadoReady, pcb_creado.Pid)
		}
	}
}

func (k *Kernel) ObtenerCPULibre() *CPU {

	for _, cpu := range k.cpusLibres {
		if cpu.Esta_libre {
			return cpu // La primera CPU que esta libre
		}
	}
	return nil // No hay CPU libre
}

func (k *Kernel) ChequearSiHayQueDesalojar() {
	primer_elemento := k.procesoPorEstado[EstadoReady][0]
	estimacion := primer_elemento.SJF.Estimado_actual
	var maxEstimacionRestante float64 = -1
	var cpu_desalojo *CPU
	var pcb *PCB

	for _, cpu := range k.cpusLibres {

		pcb = k.BuscarPorPid(EstadoExecute, cpu.Pid)
		if pcb == nil {
			fmt.Println("no esta el proceso en la lista de execute en la que deberia estar")
			continue
		}

		tiempo_ejecutando := duracionEnEstado(pcb)
		estimacion_falta_ejecutar := pcb.SJF.Estimado_actual - float64(tiempo_ejecutando)

		if estimacion_falta_ejecutar > estimacion && estimacion_falta_ejecutar > maxEstimacionRestante {
			maxEstimacionRestante = estimacion_falta_ejecutar
			cpu_desalojo = cpu
		}
	}

	if cpu_desalojo != nil {
		fmt.Printf("Desalojando proceso %d de CPU %d (estimacion restante: %.2f) para ejecutar proceso %d (estimacion: %.2f)",
			cpu_desalojo.Pid, cpu_desalojo.ID, maxEstimacionRestante, primer_elemento.Pid, primer_elemento.SJF.Estimado_actual)

		pc_a_actualizar := EnviarInterrupt(cpu_desalojo)

		k.CambiosEnElPlantel(cpu_desalojo, pc_a_actualizar)

		return
	}

}

// -----------Informa el Club Atletico Velez Sarsfield------------
func (k *Kernel) CambiosEnElPlantel(cpu *CPU, pc_a_actualizar int) {
	// Debutante
	// CALIENTA KAROL
	proceso_suplente := k.procesoPorEstado[EstadoReady][0]

	proceso_titular := k.BuscarPorPid(EstadoExecute, cpu.Pid)

	// Actualizamos pc en el pcb del proceso que estaba ejecutando
	proceso_titular.Pc = pc_a_actualizar

	// Ahora si desalojamos al pcb correspondiente
	tiempo_en_cancha := duracionEnEstado(proceso_titular)
	k.actualizarEstimacionSJF(proceso_titular, tiempo_en_cancha)
	k.MoverDeEstadoPorPid(EstadoExecute, EstadoReady, proceso_titular.Pid)

	// Actualizar la cpu con el proceso nuevo
	cpu.Pc = proceso_suplente.Pc
	cpu.Pid = proceso_suplente.Pid

	// Enviar nuevo proceso a cpu
	handleDispatch(cpu)
	// ENTRA AQUINO (Mi primo, que si aprobo el tp)
	k.MoverDeEstadoPorPid(EstadoReady, EstadoExecute, proceso_suplente.Pid)

	fmt.Printf("CAMBIO: Sale %d (est. %.2f), entra %d (est. %.2f)\n", // Leer con voz de gangoso
		proceso_titular.Pid, proceso_titular.SJF.Estimado_actual,
		proceso_suplente.Pid, proceso_suplente.SJF.Estimado_actual)
}

func EnviarInterrupt(cpu *CPU) int { // yo te hablo por la puerta interrupt y me desocupo
	fullURL := fmt.Sprintf("%s/interrupt", cpu.Url)
	resp, err := utils.EnviarSolicitudHTTPString("POST", fullURL, "OK")
	if err != nil {
		return -1
	}

	pc, _ := strconv.Atoi(resp)
	return pc
}

func (k *Kernel) BuscarPorPid(estado_actual int, pid_a_buscar int) *PCB {
	// Buscar el puntero al PCB en el estado actual
	procesos := k.procesoPorEstado[estado_actual]
	var pcb *PCB
	for _, proceso := range procesos {
		if proceso.Pid == pid_a_buscar {
			pcb = proceso
			return pcb
		}
	}
	return nil
}

func (k *Kernel) ReplanificarProceso() bool {
	// Para FIFO ya esta preparada la lista

	if k.ConfigKernel.Algoritmo_Plani == "SJF" || k.ConfigKernel.Algoritmo_Plani == "SRT" {
		lista_ready := k.procesoPorEstado[EstadoReady]
		pcb_nuevo_pid := lista_ready[len(lista_ready)-1].Pid

		sort.Sort(PorSJF(k.procesoPorEstado[EstadoReady])) //SJF distinto de nil

		if k.ConfigKernel.Algoritmo_Plani == "SRT" && pcb_nuevo_pid == lista_ready[0].Pid { //  10 15 18 31 32 500
			return true
		}
	}
	return false
}

func (k *Kernel) IntentarEnviarProcesoAExecute() {

	if len(k.procesoPorEstado[EstadoReady]) != 0 {
		fmt.Println("No hay procesos en READY")
	}

	// intentamos asignarle cpu
	cpu_seleccionada := k.ObtenerCPULibre()

	chequear_desalojo := k.ReplanificarProceso()

	//Tomamos el primer PCB tras el sort si era SJF y sino FIFO
	indice := 0
	pcb := k.procesoPorEstado[EstadoReady][indice]

	if cpu_seleccionada == nil { //no hay cpu libre
		if chequear_desalojo {
			k.ChequearSiHayQueDesalojar()
			return
		}
	}

	cpu_seleccionada.Esta_libre = false
	cpu_seleccionada.Pid = pcb.Pid
	cpu_seleccionada.Pc = pcb.Pc

	if err := handleDispatch(cpu_seleccionada); err != nil {
		fmt.Printf("Error al despachar proceso a la cpu: %v\n", err)
		cpu_seleccionada.Esta_libre = true //Revertir si falla
	}

	k.MoverDeEstadoPorPid(EstadoReady, EstadoExecute, pcb.Pid)

	fmt.Printf("Proceso %d a Execute en CPU %d\n", pcb.Pid, cpu_seleccionada.ID)
}

func handleDispatch(cpu_seleccionada *CPU) error {

	fullURL := fmt.Sprintf("%s/dispatch", cpu_seleccionada.Url)

	datos := fmt.Sprintf("%d %d", cpu_seleccionada.Pid, cpu_seleccionada.Pc)
	_, err := utils.EnviarSolicitudHTTPString("POST", fullURL, datos)
	if err != nil {
		return err
	}

	return nil
}

func (k *Kernel) solicitudEliminarProceso(pid int) (string, error) {

	url := fmt.Sprintf("http://%s:%d/memoria/%d", k.ConfigKernel.Ip_memoria, k.ConfigKernel.Puerto_Memoria, pid)
	respuestaMemo, err := utils.EnviarSolicitudHTTPString("GET", url, nil)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return "", err
	}

	//Deberia responder "OK"
	return respuestaMemo, err

}

func (k *Kernel) MemoHayEspacio(pid int, tamanio int, archivoPseudo string) (bool, error) {

	solicitud_memoria := fmt.Sprintf("%d %d %s", pid, tamanio, archivoPseudo)
	url := fmt.Sprintf("http://%s:%d/memoria/hay_lugar", k.ConfigKernel.Ip_memoria, k.ConfigKernel.Puerto_Memoria)
	respuesta, err := utils.EnviarSolicitudHTTPString("POST", url, solicitud_memoria)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return false, err
	}

	if respuesta == "Si kernel, hay espacio" {
		fmt.Println("PRUEBA - efectivamente, habia espacio")
		return true, nil
	}
	return false, nil

}

func (k *Kernel) RecibirSyscallCPU(w http.ResponseWriter, r *http.Request) {
	var respuesta string

	if err := json.NewDecoder(r.Body).Decode(&respuesta); err != nil {
		fmt.Println("error creando la solicitud:", err)
		return
	}

	syscall := strings.Split(respuesta, " ")
	fmt.Println("PRUEBA, syscall haber como llegas: ", syscall)
	k.GestionarSyscalls(syscall)
}

func (k *Kernel) GestionarSyscalls(syscall []string) {

	id_cpu, err := strconv.Atoi(syscall[IdCPU])

	if err != nil || id_cpu >= len(k.cpusLibres) { // id_cpu != k.cpusLibres[IdCPU].ID
		log.Printf("ID de CPU invalido: %v", syscall[IdCPU])
		return
	}

	pc, err := strconv.Atoi(syscall[PC])

	if err != nil {
		log.Printf("PC invalido: %v", syscall[PC])
		return
	}

	cpu_ejecutando := k.cpusLibres[id_cpu]
	cod_op := syscall[CodOp]

	switch cod_op {
	case "IO":
		// 2 20 IO AURICULARES 9000

		nombre := syscall[3]
		tiempo, _ := strconv.Atoi(syscall[4])
		cpu_ejecutando.Pc = pc
		k.ManejarIO(nombre, cpu_ejecutando, tiempo)
		//manejarIO
		//validar que exista la io
		//enviar mensaje a io

	case "INIT_PROC":
		// 2 20 INIT_PROC proceso1 256
		nombre_arch := syscall[3]
		tamanio, _ := strconv.Atoi(syscall[4])
		k.GestionarINIT_PROC(nombre_arch, tamanio, pc, cpu_ejecutando)

	case "DUMP_MEMORY":
		// 2 30 DUMP_MEMORY

		mensaje_DUMP_MEMORY := fmt.Sprintf("DUMP_MEMORY %d", cpu_ejecutando.Pid)
		utils.EnviarSolicitudHTTPString("POST", cpu_ejecutando.Url, mensaje_DUMP_MEMORY)

	case "EXIT":
		// 2 30 EXIT
		//finalizarProc
		k.GestionarEXIT(cpu_ejecutando)

	}

	cpu_ejecutando.Esta_libre = true
	k.IntentarEnviarProcesoAExecute()
}

func (k *Kernel) GestionarINIT_PROC(nombre_arch string, tamanio int, pc int, cpu_ejecutando *CPU) {
	fmt.Println("PRUEBA - Se entro a gestionarINIT_PROC")
	new_pcb := k.IniciarProceso(tamanio, nombre_arch)
	k.AgregarAEstado(EstadoNew, new_pcb)
	//log.Info("## (%d) Se crea el proceso - Estado: NEW", pcb.Pid)

	unElemento, err := k.ListaNewSoloYo()
	if err != nil {
		return
	}
	if !unElemento {
		k.PlaniLargoPlazo()
	}
	// cpu_ejecutando.Pc = pc //Actualizar pc para cpu
	// handleDispatch(cpu_ejecutando)
}

func (k *Kernel) GestionarEXIT(cpu_ejecutando *CPU) {
	respuesta, err := k.solicitudEliminarProceso(cpu_ejecutando.Pid)
	if err != nil {
		fmt.Println("Error", err)
	}

	if respuesta == "OK" {
		k.MoverDeEstadoPorPid(EstadoExecute, EstadoExit, cpu_ejecutando.Pid)
		k.QuitarDeEstado(cpu_ejecutando.Pid, EstadoExit)
		//k.EliminarProceso(cpu_ejecutando.Pid)
		cpu_ejecutando.Esta_libre = true
		//k.IntentarEnviarProcesoAReady()
	}
}

func (k *Kernel) buscarIOLibre(nombre string) *IO {

	ioMutex.RLock()
	defer ioMutex.RUnlock()

	if iosDispo, ok := k.ios[nombre]; ok {
		for _, instancia := range iosDispo.io {
			if instancia.Esta_libre {
				return instancia
			}
		}
	}

	return nil

}

func RegistrarCPUaLibre(cpu_a_liberar *CPU) {
	cpu_a_liberar.Esta_libre = true
	cpu_a_liberar.Pid = -1
}

func (k *Kernel) ManejarIO(nombre_io string, cpu_ejecutando *CPU, duracion int) {
	defer RegistrarCPUaLibre(cpu_ejecutando)
	io, existeIO := k.ios[nombre_io]

	if !existeIO {
		k.GestionarEXIT(cpu_ejecutando)
		return
	}

	pcb := k.BuscarPorPid(EstadoExecute, cpu_ejecutando.Pid)
	pcb.Pc = cpu_ejecutando.Pc
	k.MoverDeEstadoPorPid(EstadoExecute, EstadoBlock, cpu_ejecutando.Pid)
	go k.temporizadorSuspension(pcb.Pid)

	IO_Seleccionada := k.buscarIOLibre(nombre_io)

	if IO_Seleccionada == nil { //no hay io libre
		io.procEsperandoPorIO = append(io.procEsperandoPorIO, cpu_ejecutando.Pid)
		return
	}
	// si hay io libre
	IO_Seleccionada.Pid = cpu_ejecutando.Pid
	//enviar a io
	IO_Seleccionada.Esta_libre = false

	enviarProcesoAIO(IO_Seleccionada, duracion)

}

func (k *Kernel) temporizadorSuspension(pid int) {
	suspension := time.Duration(k.ConfigKernel.Tiempo_Suspension)
	time.Sleep(suspension)

	pcb := k.BuscarPorPid(EstadoBlock, pid)
	if pcb != nil {
		k.MoverDeEstadoPorPid(EstadoBlock, EstadoBlockSuspended, pid)
	}
}

func enviarProcesoAIO(io_seleccionada *IO, duracion int) {

	fullURL := fmt.Sprintf("%sio/hace_algo", io_seleccionada.Url)
	datos := fmt.Sprintf("%d %d", io_seleccionada.Pid, duracion)
	utils.EnviarSolicitudHTTPString("POST", fullURL, datos)
}

func (k *Kernel) RecibirRespuestaIO(w http.ResponseWriter, r *http.Request) {
	var respuesta string
	if err := json.NewDecoder(r.Body).Decode(&respuesta); err != nil {
		fmt.Println("Error recibiendo la solicitud:", err)
		return
	}

	data := strings.Split(respuesta, " ")

	if len(data) < 3 {
		log.Printf("Respuesta IO mal formada: %s", respuesta)
		return
	}

	cod_op := data[0]    // cod_op
	nombre_io := data[1] // nombre_io
	pid_io, _ := strconv.Atoi(data[2])

	switch cod_op {
	case "FIN_IO":
		k.MoverDeEstadoPorPid(EstadoBlock, EstadoReady, pid_io)
		k.MoverDeEstadoPorPid(EstadoBlockSuspended, EstadoReadySuspended, pid_io)

	case "DESCONEXION":
		k.BorrarIO(nombre_io, pid_io)
	}

}

func (k *Kernel) BorrarIO(nombre_io string, pid int) {

	iosMismoNombre := k.ios[nombre_io]
	for i, valor := range iosMismoNombre.io {
		if valor.Pid == pid {
			iosMismoNombre.io = append(iosMismoNombre.io[:i], iosMismoNombre.io[i+1:]...)
			return
		}
	}
	fmt.Printf("No se encontró una instancia con pid %d en %s para desconectar\n", pid, nombre_io)
}

func (k *Kernel) registrarNuevaCPU(w http.ResponseWriter, r *http.Request) { // Handshake

	fmt.Println("Al menos, entre a registrar nueva cpu")
	var cpu_string string
	if err := json.NewDecoder(r.Body).Decode(&cpu_string); err != nil {
		fmt.Println("Error recibiendo la solicitud:", err)
		return
	}

	aux := strings.Split(cpu_string, " ") //ID IP PUERTO

	if len(aux) != 3 {
		http.Error(w, "Formato invalido. Esperando: 'ID IP PUERTO'", http.StatusBadRequest)
		return
	}

	http.Error(w, "Formato invalido. Esperando: 'ID IP PUERTO'", http.StatusBadRequest)

	nueva_ID_CPU, _ := strconv.Atoi(aux[0])
	ip := aux[1]
	puerto := aux[2]

	url := fmt.Sprintf("http://%s:%s/cpu", ip, puerto)

	nueva_cpu := CPU{
		ID:         nueva_ID_CPU,
		Url:        url,
		Pid:        -1,
		Pc:         0,
		Esta_libre: true}

	// Agrego y sincronizo el nuevo CPU
	mutex_cpus_libres.Lock()
	defer mutex_cpus_libres.Unlock()

	k.cpusLibres[nueva_cpu.ID] = &nueva_cpu

	fmt.Printf("Se conectó una nueva CPU con ID %d en %s\n", nueva_cpu.ID, url)

	//RESPONDER OK
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

}

func (k *Kernel) registrarNuevaIO(w http.ResponseWriter, r *http.Request) { // Handshake
	var io_string string
	if err := json.NewDecoder(r.Body).Decode(&io_string); err != nil {
		fmt.Println("Error recibiendo la solicitud:", err)
		return
	}

	aux := strings.Split(io_string, " ") // NOMBRE IP PUERTO

	if len(aux) != 3 {
		http.Error(w, "Formato inválido", http.StatusBadRequest)
		return
	}

	nombre_io := aux[0] //Para mas claridad :p
	url := fmt.Sprintf("http://%s:%s/io/registrar_io", aux[1], aux[2])

	mutex_ios.Lock()

	defer mutex_ios.Unlock()

	nuevaIO := &IO{
		Url:        url,
		Pid:        -1,
		Esta_libre: true,
	}
	// Si no existe una io con ese nombre, lo agrego nuevito

	if _, ok := k.ios[nombre_io]; !ok {
		// Agrego y sincronizo el nuevo dispositivo io

		k.ios[nombre_io] = &IOS{
			io:                 []*IO{nuevaIO},
			procEsperandoPorIO: []int{},
		}
	} else {
		// Sino, actualizo los valores en esa posicion
		k.ios[nombre_io].io = append(k.ios[nombre_io].io, nuevaIO)
	}

}
