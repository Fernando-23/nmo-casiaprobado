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

	"slices"

	utils "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

// array de arrays que contenga a todas las colas

func IniciarConfiguracion[T any](ruta string, estructuraDeConfig *T) error {

	fmt.Println("Cargando configuracion desde", ruta)
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

func esperarEnter(signalEnter chan struct{}) {

	fmt.Println("Ingrese ENTER para empezar con la planificacion")

	reader := bufio.NewReader(os.Stdin)
	_, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Println("Error leyendo del teclado:", err)
	}

	signalEnter <- struct{}{} //Envia una señal para avisar al hilo principal que el usuario presiono Enter

}

func (k *Kernel) InicializarMapaDeEstados() {
	k.ProcesoPorEstado = make(map[int][]*PCB)

	// Inicializamos todos los estados del map
	for i := 0; i < cantEstados; i++ {
		k.ProcesoPorEstado[i] = []*PCB{}
	}
}

func (k *Kernel) HandshakeMemoria() error {
	url := fmt.Sprintf("http://%s:%d/memoria/handshake", k.Configuracion.Ip_memoria, k.Configuracion.Puerto_Memoria)
	respuesta, err := utils.EnviarSolicitudHTTPString("GET", url, nil)
	if err != nil {
		return fmt.Errorf("memoria no responde: %w", err)
	}

	if respuesta != "OK" {
		return fmt.Errorf("respuesta inesperada de memoria: %s", respuesta)
	}

	return nil
}

//	FUNCIONES QUE CREAN  ELEMENTOS DEL KERNEL

func (k *Kernel) IniciarProceso(tamanio int, arch_pseudo string) *PCB {
	pcb := k.CrearPCB(tamanio, arch_pseudo)
	pcb.HoraIngresoAEstado = time.Now()

	cambiarMetricasEstado(pcb, EstadoNew)
	//k.AgregarAEstado(EstadoNew, pcb) //meter en la cola new no hay planificacion para meter en la cola new
	return pcb
}

func (k *Kernel) CrearPCB(tamanio int, arch_pseudo string) *PCB {
	mutex_SiguientePid.Lock()
	pidMio := k.SiguientePID
	k.SiguientePID++
	mutex_SiguientePid.Unlock()

	pcb := &PCB{
		Pid:         pidMio,
		Tamanio:     tamanio,
		Arch_pseudo: arch_pseudo,
		Pc:          0,
	}

	if k.Configuracion.Algoritmo_Plani == "SJF" {
		k.CrearSJF(pcb)
	} else {
		pcb.SJF = nil
	}

	return pcb
}

func (k *Kernel) CrearSJF(pcb *PCB) {
	sjf := &SJF{
		Estimado_anterior: k.Configuracion.Estimacion_Inicial,
		Estimado_actual:   k.Configuracion.Estimacion_Inicial,
		Real_anterior:     0, //no ejecuto valor igual a 0
	}
	pcb.SJF = sjf
}

func (k *Kernel) registrarNuevaCPU(mensajeCPU string) bool {

	aux := strings.Split(mensajeCPU, " ") //ID IP PUERTO

	if len(aux) != 3 {
		fmt.Println("Formato invalido. Esperando: 'ID IP PUERTO'")
		return false
	}

	nueva_ID_CPU, err := strconv.Atoi(aux[0])

	if err != nil {
		fmt.Println("ID de CPU inválido")
		return false
	}

	ip := aux[1]
	puerto := aux[2]
	url := fmt.Sprintf("http://%s:%s/cpu", ip, puerto)

	mutex_CPUsConectadas.Lock()
	defer mutex_CPUsConectadas.Unlock()

	if _, existe := k.CPUsConectadas[nueva_ID_CPU]; existe {
		fmt.Println("Ya existe una CPU registrada con ese ID")
		return false
	}

	k.CPUsConectadas[nueva_ID_CPU] = crearCPU(nueva_ID_CPU, url)

	fmt.Printf("Se conecto una nueva CPU con ID %d en %s\n", nueva_ID_CPU, url)

	return true
}

func crearCPU(id int, url string) *CPU {
	nueva_cpu := &CPU{
		ID:         id,
		Url:        url,
		Pid:        -1,
		Pc:         0,
		Esta_libre: true,
	}
	return nueva_cpu

}

//	FUNCIONES QUE ACTUALIZAN  ELEMENTOS DEL KERNEL

func cambiarMetricasEstado(pPcb *PCB, posEstado int) {
	pPcb.Me[posEstado]++ //ver si puede quedar mas lindo

}

func cambiarMetricasTiempo(pPcb *PCB, posEstado int) {
	pPcb.Mt[posEstado] += duracionEnEstado(pPcb)
}

func (k *Kernel) actualizarEstimacionSJF(pcb *PCB, tiempoEnExecute time.Duration) {
	if pcb == nil || pcb.SJF == nil {
		return
	}
	real_anterior := float64(tiempoEnExecute.Milliseconds())
	alpha := k.Configuracion.Alfa
	sjf := pcb.SJF
	aux := sjf.Estimado_actual
	sjf.Estimado_actual = (alpha * real_anterior) + ((1 - alpha) * sjf.Estimado_anterior)
	sjf.Estimado_anterior = aux
}

// FUNCIONES PARA BUSQUEDAS

func (k *Kernel) BuscarPorPidSeguro(estado int, pid int) *PCB {
	mutex_ProcesoPorEstado[estado].Lock()
	defer mutex_ProcesoPorEstado[estado].Unlock()

	return k.BuscarPorPidSinLock(estado, pid)
}

func (k *Kernel) BuscarPorPidSinLock(estado int, pid int) *PCB {

	// Buscar el puntero al PCB en el estado actual
	procesos := k.ProcesoPorEstado[estado]
	var pcb *PCB
	for _, proceso := range procesos {
		if proceso.Pid == pid {
			pcb = proceso
			return pcb
		}
	}
	return nil
}

// FUNCIONES DE UTILIDAD GENERAL

func duracionEnEstado(pPcb *PCB) time.Duration {
	tiempoActual := time.Now()
	return tiempoActual.Sub(pPcb.HoraIngresoAEstado)
}

func (k *Kernel) TieneProcesos(estado int) bool {

	return len(k.ProcesoPorEstado[estado]) > 0
}

func (k *Kernel) ObtenerCPULibre() *CPU {

	mutex_CPUsConectadas.Lock()
	defer mutex_CPUsConectadas.Unlock()

	for id, cpu := range k.CPUsConectadas {
		if cpu.Esta_libre {
			//la sacamos para que nadie mas la tome
			delete(k.CPUsConectadas, id)
			cpu.Esta_libre = false
			return cpu // La primera CPU que esta libre
		}
	}
	return nil // No hay CPU libre
}

//FUNCIONES PARA AGREGAR, SACAR y MOVER ELEMENTOS DE ESTRUCTURAS DEL KERNEL

func (k *Kernel) AgregarAEstado(estado int, pcb *PCB, hacerSincro bool) {
	if hacerSincro {
		mutex_ProcesoPorEstado[estado].Lock()
		defer mutex_ProcesoPorEstado[estado].Unlock()
	}

	k.ProcesoPorEstado[estado] = append(k.ProcesoPorEstado[estado], pcb)
}

func (k *Kernel) QuitarYObtenerPCB(estado int, pid int, hacerSincro bool) *PCB {
	if hacerSincro {
		mutex_ProcesoPorEstado[estado].Lock()
		defer mutex_ProcesoPorEstado[estado].Unlock()

	}

	procesos := k.ProcesoPorEstado[estado]
	for i, pcb := range procesos {
		if pcb.Pid == pid {
			// Quitar el proceso de la lista del estado
			k.ProcesoPorEstado[estado] = slices.Delete(procesos, i, i+1)
			return pcb
		}
	}
	return nil //no se encontro
}

func (k *Kernel) QuitarPrimerElemento(estado int) *PCB {

	pcbs := k.ProcesoPorEstado[estado]

	if len(pcbs) == 0 {
		return nil
	}

	primer_elemento := pcbs[0]
	k.ProcesoPorEstado[estado] = pcbs[1:]

	return primer_elemento

}

func (k *Kernel) MoverDeEstadoPorPid(estadoActual, estadoNuevo int, pid int, hacerSincro bool) {
	// Buscar el puntero al PCB en el estado actual
	pcb := k.QuitarYObtenerPCB(estadoActual, pid, hacerSincro) //aca sincroniza

	if pcb == nil {
		fmt.Printf("Proceso %d no encontrado en el estado %d\n", pid, estadoActual)
		return
	}

	estadoAnterior := estadoActual

	// Actualizar metricas no requiere sincro porque nadie tiene acceso ya a este pcb no pertenece a ningun estado en este instante
	//sincronizacion inplicita

	cambiarMetricasTiempo(pcb, estadoAnterior)

	pcb.HoraIngresoAEstado = time.Now()

	cambiarMetricasEstado(pcb, estadoNuevo)

	utils.LoggerConFormato("## (%d) Pasa del estado %s al estado %s", pid, estados_proceso[estadoActual], estados_proceso[estadoNuevo])

	// Cambiar el estado del proceso
	pcb.estado = estadoNuevo

	// Agregar el puntero al PCB al nuevo estado
	k.AgregarAEstado(estadoNuevo, pcb, hacerSincro) //aca sincroniza

}

func (k *Kernel) MoverDeEstado(estadoActual, estadoNuevo int) {

	pcb := k.QuitarPrimerElemento(estadoActual) //ya sincroniza internamente

	if pcb == nil {
		fmt.Printf("No hay procesos en el estado %d\n", estadoActual)
		return
	}
	// Actualizar métricas
	cambiarMetricasTiempo(pcb, estadoActual)

	pcb.HoraIngresoAEstado = time.Now()

	cambiarMetricasEstado(pcb, estadoNuevo)

	utils.LoggerConFormato("## (%d) Pasa del estado %s al estado %s", pcb.Pid, estados_proceso[estadoActual], estados_proceso[estadoNuevo])

	// Cambiar el estado del proceso
	pcb.estado = estadoNuevo

	// Agregar el puntero al PCB al nuevo estado
	k.AgregarAEstado(estadoNuevo, pcb, true) //ya sincroniza internamente

}

// PLANIFICACION

func (k *Kernel) UnicoEnNewYNadaEnSuspReady() (bool, error) {
	mutex_ProcesoPorEstado[EstadoNew].Lock()
	mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()

	lista_new := k.ProcesoPorEstado[EstadoNew]
	lista_susp_ready := k.ProcesoPorEstado[EstadoReadySuspended]

	//Caso en en que hay exactamente un proceso en NEW y ninguno en SUSPENDED-READY
	if len(lista_susp_ready) == 0 && len(lista_new) == 1 {
		fmt.Println("Soy el primer elemento y no hay procesos en SUSP READY", lista_new[0].Pid)
		primer_elemento := k.QuitarPrimerElemento(EstadoNew)

		//hay que desbloquear porque vamos a hacer una peticion http y no da que siga reteniendo el recurso
		//y ademas como si fuera poco ya no necesita acceder a las listas hasta el mover deestado que tiene su debida sincronizacion
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
		mutex_ProcesoPorEstado[EstadoNew].Unlock()

		hay_espacio, err := k.MemoHayEspacio(primer_elemento.Pid, primer_elemento.Tamanio, primer_elemento.Arch_pseudo)

		if err != nil {
			log.Printf("Error codificando mensaje: %s", err.Error())
			return true, err
		}
		if hay_espacio {
			// Cambiar de estado del proceso de NEW a READY
			k.AgregarAEstado(EstadoNew, primer_elemento, true) //aca sincroniza
		}
		return true, nil
	}
	return false, nil
}

func (k *Kernel) IntentarEnviarProcesoAReady() (bool, error) {
	mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()
	mutex_ProcesoPorEstado[EstadoNew].Lock()

	if k.TieneProcesos(EstadoReadySuspended) {
		return k.PlanificarLargoPorLista(EstadoReadySuspended)
	}
	return k.PlanificarLargoPorLista(EstadoNew)
}

func (k *Kernel) PlanificarLargoPorLista(codLista int) (bool, error) {

	//si el algoritmo es PCMP, ordenamos antes de tomar el primero
	if k.Configuracion.Ready_ingress_algorithm == "PCMP" {
		sort.Sort(PorTamanio(k.ProcesoPorEstado[codLista]))
	}

	pcb := k.QuitarPrimerElemento(codLista)

	mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
	mutex_ProcesoPorEstado[EstadoNew].Unlock()

	if pcb == nil {
		return false, nil
	}

	hay_espacio, err := k.MemoHayEspacio(pcb.Pid, pcb.Tamanio, pcb.Arch_pseudo)

	if err != nil {
		return false, err
	}

	if hay_espacio {
		k.AgregarAEstado(EstadoReady, pcb, true)
		return true, nil
	}
	k.AgregarAEstado(codLista, pcb, true)
	return false, nil
}

func (k *Kernel) PlaniCortoPlazo() bool {

	// Para FIFO ya esta preparada la lista

	if k.Configuracion.Algoritmo_Plani == "SJF" || k.Configuracion.Algoritmo_Plani == "SRT" {
		lista_ready := k.ProcesoPorEstado[EstadoReady]
		pcb_nuevo_pid := lista_ready[len(lista_ready)-1].Pid

		sort.Sort(PorSJF(k.ProcesoPorEstado[EstadoReady])) //SJF distinto de nil

		if k.Configuracion.Algoritmo_Plani == "SRT" && pcb_nuevo_pid == lista_ready[0].Pid { //  10 15 18 31 32 500
			return true
		}
	}
	return false
}

func (k *Kernel) IntentarEnviarProcesoAExecute() {
	if !k.TieneProcesos(EstadoReady) {
		fmt.Println("No hay procesos en READY")
		return
	}
	// intentamos asignarle cpu
	cpu_seleccionada := k.ObtenerCPULibre()

	hay_que_chequear_desalojo := k.PlaniCortoPlazo()

	//Tomamos el primer PCB tras la planificacion
	indice := 0
	pcb := k.ProcesoPorEstado[EstadoReady][indice]

	if cpu_seleccionada == nil { //no hay cpu libre
		if hay_que_chequear_desalojo {
			k.ChequearDesalojoPorSRT()
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

	k.MoverDeEstadoPorPid(EstadoReady, EstadoExecute, pcb.Pid, false)

	fmt.Printf("Proceso %d a Execute en CPU %d\n", pcb.Pid, cpu_seleccionada.ID)
}

func (k *Kernel) temporizadorSuspension(pid int) {
	suspension := time.Duration(k.Configuracion.Tiempo_Suspension)
	time.Sleep(suspension)

	pcb := k.BuscarPorPidSeguro(EstadoBlock, pid)
	if pcb != nil {
		k.MoverDeEstadoPorPid(EstadoBlock, EstadoBlockSuspended, pid, true)
	}
}

func (k *Kernel) ChequearDesalojoPorSRT() {
	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	mutex_ProcesoPorEstado[EstadoReady].Lock()
	mutex_CPUsConectadas.Lock()

	defer mutex_CPUsConectadas.Unlock()
	defer mutex_ProcesoPorEstado[EstadoReady].Unlock()
	defer mutex_ProcesoPorEstado[EstadoExecute].Unlock()

	listaReady := k.ProcesoPorEstado[EstadoReady]

	if len(listaReady) == 0 {
		return //no hay procesos en READY, no tiene sentido desalojar
	}

	procesoCandidato := listaReady[0]
	estimacionReady := procesoCandidato.SJF.Estimado_actual

	var estimacionMaxRestante float64 = -1
	var cpuElegida *CPU
	var procesoEjecutando *PCB

	for _, cpu := range k.CPUsConectadas {

		procesoEjecutando = k.BuscarPorPidSinLock(EstadoExecute, cpu.Pid)
		if procesoEjecutando == nil {
			fmt.Println("ERROR: el proceso no esta en la lista execute, incosistencia interna")
			return
		}

		tiempoEjecutando := duracionEnEstado(procesoEjecutando)
		estimacionRestante := procesoEjecutando.SJF.Estimado_actual - float64(tiempoEjecutando)

		if estimacionRestante > estimacionReady && estimacionRestante > estimacionMaxRestante {
			estimacionMaxRestante = estimacionRestante
			cpuElegida = cpu
		}
	}

	if cpuElegida != nil {
		fmt.Printf("Desalojando proceso %d de CPU %d (estimacion restante: %.2f) para ejecutar proceso %d (estimacion: %.2f)",
			cpuElegida.Pid,
			cpuElegida.ID,
			estimacionMaxRestante,
			procesoCandidato.Pid,
			estimacionReady)

		procesoDesalojado := EnviarInterrupt(cpuElegida)

		k.CambiosEnElPlantel(cpuElegida, procesoDesalojado)
		return
	}

}

// -----------Informa el Club Atletico Velez Sarsfield------------
func (k *Kernel) CambiosEnElPlantel(cpu *CPU, pc_a_actualizar int) {
	// Debutante
	// CALIENTA KAROL

	lista_ready := k.ProcesoPorEstado[EstadoReady]

	if len(lista_ready) == 0 {
		fmt.Printf("Se intento desalojar por SRT pero no hay procesos en ready")
		return
	}

	proceso_suplente := k.ProcesoPorEstado[EstadoReady][0]

	proceso_titular := k.BuscarPorPidSinLock(EstadoExecute, cpu.Pid)

	if proceso_titular == nil {
		fmt.Printf("Se intentó desalojar por SRT pero no se encontro el proceso a desalojar en execute")
		return
	}

	// Actualizamos pc en el pcb del proceso que estaba ejecutando
	proceso_titular.Pc = pc_a_actualizar

	// Ahora si desalojamos al pcb correspondiente
	tiempo_en_cancha := duracionEnEstado(proceso_titular)
	k.actualizarEstimacionSJF(proceso_titular, tiempo_en_cancha)

	utils.LoggerConFormato("## (%d) - Desalojado por algoritmo SJF/SRT", cpu.Pid)

	k.MoverDeEstadoPorPid(EstadoExecute, EstadoReady, proceso_titular.Pid, false)

	// Actualizar la cpu con el proceso nuevo
	cpu.Pc = proceso_suplente.Pc
	cpu.Pid = proceso_suplente.Pid

	// Enviar nuevo proceso a cpu
	handleDispatch(cpu)
	// ENTRA AQUINO (Mi primo, que si aprobo el tp)
	k.MoverDeEstadoPorPid(EstadoReady, EstadoExecute, proceso_suplente.Pid, false)

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

func handleDispatch(cpu_seleccionada *CPU) error {

	fullURL := fmt.Sprintf("%s/dispatch", cpu_seleccionada.Url)

	datos := fmt.Sprintf("%d %d", cpu_seleccionada.Pid, cpu_seleccionada.Pc)
	_, err := utils.EnviarSolicitudHTTPString("POST", fullURL, datos)
	if err != nil {
		return err
	}

	return nil
}

func (k *Kernel) MemoHayEspacio(pid int, tamanio int, archivoPseudo string) (bool, error) {

	solicitud_memoria := fmt.Sprintf("%d %d %s", pid, tamanio, archivoPseudo)
	url := fmt.Sprintf("http://%s:%d/memoria/hay_lugar", k.Configuracion.Ip_memoria, k.Configuracion.Puerto_Memoria)
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

func RegistrarCPUaLibre(cpu_a_liberar *CPU) {
	cpu_a_liberar.Esta_libre = true
	cpu_a_liberar.Pid = -1
}

func (k *Kernel) llegaNuevaCPU(w http.ResponseWriter, r *http.Request) { // Handshake

	var mensajeCPU string
	if err := json.NewDecoder(r.Body).Decode(&mensajeCPU); err != nil {
		fmt.Println("Error recibiendo la solicitud:", err)
		http.Error(w, "Error en el formato de la solicitud", http.StatusBadRequest)
		return
	}

	fmt.Printf("CPU recibida con mensaje: %s\n", mensajeCPU)

	if !k.registrarNuevaCPU(mensajeCPU) {
		http.Error(w, "No se pudo registar la CPU", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
