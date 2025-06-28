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

	if k.Configuracion.Algoritmo_Plani == "SJF" || k.Configuracion.Algoritmo_Plani == "SRT" {
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

func actualizarMetricasEstado(pPcb *PCB, posEstado int) {
	pPcb.Me[posEstado]++ //ver si puede quedar mas lindo

}

func actualizarMetricasTiempo(pPcb *PCB, posEstado int) {
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
	return time.Since(pPcb.HoraIngresoAEstado)
}

func (k *Kernel) TieneProcesos(estado int) bool {

	return len(k.ProcesoPorEstado[estado]) > 0
}

func (k *Kernel) ObtenerCPULibre() *CPU {

	for _, cpu := range k.CPUsConectadas {
		if cpu.Esta_libre {
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

	actualizarMetricasEstado(pcb, estado)
	pcb.HoraIngresoAEstado = time.Now()

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

			// Actualizamos la metrica de tiempo de dicho estado
			actualizarMetricasTiempo(pcb, estado)

			// Sacar el proceso de la lista del estado
			k.ProcesoPorEstado[estado] = slices.Delete(procesos, i, i+1)
			return pcb
		}
	}
	return nil //no se encontro
}

func (k *Kernel) PrimerElementoSinSacar(estado int) *PCB {

	pcbs := k.ProcesoPorEstado[estado]

	if len(pcbs) == 0 {
		return nil
	}

	return pcbs[0]

}

func (k *Kernel) ElementoNSinSacar(estado int, n int) *PCB {

	pcbs := k.ProcesoPorEstado[estado]

	if len(pcbs) == 0 {
		return nil
	}

	return pcbs[n]

}

func (k *Kernel) MoverDeEstadoPorPid(estadoActual, estadoNuevo int, pid int, hacerSincro bool) bool {
	// Buscar el puntero al PCB en el estado actual
	pcb := k.QuitarYObtenerPCB(estadoActual, pid, hacerSincro) //aca sincroniza

	if pcb == nil {
		utils.LoggerConFormato("## ERROR (MoverDeEstadoPorPid) Proceso %d no encontrado en el estado %d\n", pid, estadoActual)
		return false
	}

	// Agregar el puntero al PCB al nuevo estado
	k.AgregarAEstado(estadoNuevo, pcb, hacerSincro) //aca sincroniza

	utils.LoggerConFormato("## (%d) Pasa del estado %s al estado %s", pid, estados_proceso[estadoActual], estados_proceso[estadoNuevo])
	return true
}

// PLANIFICACION

func (k *Kernel) UnicoEnNewYNadaEnSuspReady() (bool, bool) { //el primero es si es unico y el segundo si pudo pasar a ready
	mutex_peticionHayEspacioMemoria.Lock()
	defer mutex_peticionHayEspacioMemoria.Unlock()

	mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()
	mutex_ProcesoPorEstado[EstadoNew].Lock()

	lista_new := k.ProcesoPorEstado[EstadoNew]
	lista_susp_ready := k.ProcesoPorEstado[EstadoReadySuspended]

	//Caso en en que hay exactamente un proceso en NEW y ninguno en SUSPENDED-READY
	if len(lista_susp_ready) == 0 && len(lista_new) == 1 {
		fmt.Println("Soy el primer elemento y no hay procesos en SUSP READY", lista_new[0].Pid)
		procCandidatoAReady := k.PrimerElementoSinSacar(EstadoNew)

		// me clono los datos por las dudas no vaya ser que el puntero apunte a otro lado
		pid := procCandidatoAReady.Pid
		tamanio := procCandidatoAReady.Tamanio
		arch_pseudo := procCandidatoAReady.Arch_pseudo

		//Liberamos recursos por peticion http pero reservo el proceso
		MarcarProcesoReservado(procCandidatoAReady, true)
		mutex_ProcesoPorEstado[EstadoNew].Unlock()
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()

		if k.gestionarAccesoAReady(pid, tamanio, arch_pseudo, EstadoNew) {
			return true, true
		}
		return true, false
	}
	return false, false
}

func MarcarProcesoReservado(pcb *PCB, reservado bool) {
	pcb.Reservado = reservado
}
func EstaReservado(pcb *PCB) bool {
	return pcb.Reservado
}

func (k *Kernel) IntentarEnviarProcesoAReady(estadoOrigen int, pidQuiereEntrar int) bool {
	mutex_peticionHayEspacioMemoria.Lock()
	defer mutex_peticionHayEspacioMemoria.Unlock()

	mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()
	mutex_ProcesoPorEstado[EstadoNew].Lock()

	// Si intento mover NEW, pero hay procesos en READY_SUSPENDED, no hago nada
	if estadoOrigen == EstadoNew && k.TieneProcesos(EstadoReadySuspended) {
		mutex_ProcesoPorEstado[EstadoNew].Unlock()
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
		return false
	}

	if !k.TieneProcesos(estadoOrigen) {
		mutex_ProcesoPorEstado[EstadoNew].Unlock()
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
		return false
	}

	if !k.hayQuePlanificarAccesoAReady(estadoOrigen, pidQuiereEntrar) {
		mutex_ProcesoPorEstado[EstadoNew].Unlock()
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
		return false
	}

	if k.Configuracion.Ready_ingress_algorithm == "PCMP" {
		sort.Sort(PorTamanio(k.ProcesoPorEstado[estadoOrigen]))
	}

	procCandidatoAReady := k.PrimerElementoSinSacar(estadoOrigen)

	// Verifico que el primer proceso candidato sea el que quiere entrar
	if procCandidatoAReady.Pid != pidQuiereEntrar {
		mutex_ProcesoPorEstado[EstadoNew].Unlock()
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
		return false
	}

	// me clono los datos por las dudas no vaya ser que el puntero apunte a otro lado
	pid := procCandidatoAReady.Pid
	tamanio := procCandidatoAReady.Tamanio
	arch_pseudo := procCandidatoAReady.Arch_pseudo

	//Resevo el proceso
	MarcarProcesoReservado(procCandidatoAReady, true)

	//Libero los mutex antes de hacer la peticion HTTP (que puede tardar)
	mutex_ProcesoPorEstado[EstadoNew].Unlock()
	mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()

	// consulto memoria
	return k.gestionarAccesoAReady(pid, tamanio, arch_pseudo, estadoOrigen)
}

func (k *Kernel) hayQuePlanificarAccesoAReady(estadoOrigen int, pid int) bool {
	algoritmoPlani := k.Configuracion.Ready_ingress_algorithm
	lProcEstado := k.ProcesoPorEstado[estadoOrigen]

	if len(lProcEstado) == 1 {
		return true
	}

	if algoritmoPlani == "PCMP" && k.esProcesoMasChico(pid, estadoOrigen) {
		return true
	}

	return false
}

func (k *Kernel) esProcesoMasChico(pid int, estadoOrigen int) bool {
	procQuiereDestronar := k.BuscarPorPidSinLock(estadoOrigen, pid)
	procMasChico := k.PrimerElementoSinSacar(estadoOrigen)

	//Si es mas chico
	if procQuiereDestronar.Tamanio < procMasChico.Tamanio {
		return true
	}
	return false
}

func (k *Kernel) IntentarEnviarProcesosAReady() {
	mutex_peticionHayEspacioMemoria.Lock()
	defer mutex_peticionHayEspacioMemoria.Unlock()

	mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()
	mutex_ProcesoPorEstado[EstadoNew].Lock()

	k.PlanificarLargoPorLista(EstadoReadySuspended)

	for k.TieneProcesos(EstadoReadySuspended) {

		procCandidatoAReady := k.PrimerElementoSinSacar(EstadoReadySuspended)
		pid := procCandidatoAReady.Pid
		tamanio := procCandidatoAReady.Tamanio
		arch_pseudo := procCandidatoAReady.Arch_pseudo

		MarcarProcesoReservado(procCandidatoAReady, true)

		mutex_ProcesoPorEstado[EstadoNew].Unlock()
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()

		exito := k.gestionarAccesoAReady(pid, tamanio, arch_pseudo, EstadoReadySuspended)

		// Volvemos a tomar los mutex para la siguiente iteración o para pasar a NEW
		mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()
		mutex_ProcesoPorEstado[EstadoNew].Lock()

		if !exito {
			// No pudo pasar, salimos del ciclo para evitar bucle infinito
			break
		}
	}

	//si no quedan procesos en READY_SUSPENDED, intentamos con new
	if !k.TieneProcesos(EstadoReadySuspended) {
		k.PlanificarLargoPorLista(EstadoNew)

		for k.TieneProcesos(EstadoNew) {

			procCandidatoAReady := k.PrimerElementoSinSacar(EstadoNew)
			pid := procCandidatoAReady.Pid
			tamanio := procCandidatoAReady.Tamanio
			arch_pseudo := procCandidatoAReady.Arch_pseudo

			MarcarProcesoReservado(procCandidatoAReady, true)

			mutex_ProcesoPorEstado[EstadoNew].Unlock()
			mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()

			exito := k.gestionarAccesoAReady(pid, tamanio, arch_pseudo, EstadoNew)

			mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()
			mutex_ProcesoPorEstado[EstadoNew].Lock()

			if !exito {
				// No pudo pasar, salimos del ciclo para evitar bucle infinito
				break
			}
		}
	}
	mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
	mutex_ProcesoPorEstado[EstadoNew].Unlock()
}

func (k *Kernel) PlanificarLargoPorLista(codLista int) {

	//si el algoritmo es PCMP, ordenamos antes de tomar el primero
	if k.Configuracion.Ready_ingress_algorithm == "PCMP" {
		sort.Sort(PorTamanio(k.ProcesoPorEstado[codLista]))
	}
}

func (k *Kernel) gestionarAccesoAReady(pid int, tamanio int, arch_pseudo string, estadoOrigen int) bool {

	//Consultamos memoria ...
	hay_espacio, err := k.MemoHayEspacio(pid, tamanio, arch_pseudo)

	if err != nil {
		mutex_ProcesoPorEstado[estadoOrigen].Lock()
		defer mutex_ProcesoPorEstado[estadoOrigen].Unlock()
		utils.LoggerConFormato("MemoHayEspacio retorno el error:  %e", err)
		utils.LoggerConFormato("Intento deslockear el proceso %d", pid)
		pcb := k.BuscarPorPidSinLock(estadoOrigen, pid)
		if pcb == nil {
			utils.LoggerConFormato("No pude deslockear al proceso %d ya que no estaba en la lista %s", pid, estados_proceso[estadoOrigen])
			return false

		}
		MarcarProcesoReservado(pcb, false)
		return false
	}
	if hay_espacio {
		//Pedimos mutex para intentar pasar al procCandidato de NEW a READY
		mutex_ProcesoPorEstado[estadoOrigen].Lock()
		mutex_ProcesoPorEstado[EstadoReady].Lock()
		defer mutex_ProcesoPorEstado[EstadoReady].Unlock()
		defer mutex_ProcesoPorEstado[estadoOrigen].Unlock()

		//Checkeo por las dudas que sigue en la misma posicion de memoria
		procVerificadoAReady := k.QuitarYObtenerPCB(estadoOrigen, pid, false)

		if procVerificadoAReady == nil {
			utils.LoggerConFormato("## ERROR (gestionarAccesoAReady), el proceso no esta en la lista %s", estados_proceso[estadoOrigen])
			return false
		}

		MarcarProcesoReservado(procVerificadoAReady, false)
		k.AgregarAEstado(EstadoReady, procVerificadoAReady, false)

		utils.LoggerConFormato("(%d) Pasa del estado <%s> al estado <%s>", pid, estados_proceso[estadoOrigen], estados_proceso[EstadoReady])

		return true
	}
	return false
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
	mutex_ProcesoPorEstado[EstadoReady].Lock()
	if !k.TieneProcesos(EstadoReady) {
		fmt.Println("No hay procesos en READY")
		mutex_ProcesoPorEstado[EstadoReady].Unlock()
		return
	}
	hay_que_chequear_desalojo := k.PlaniCortoPlazo()

	//Tomamos el primer PCB "NO" reservado tras la planificacion
	var pcb *PCB
	listaReady := k.ProcesoPorEstado[EstadoReady]
	for i := 0; i < len(listaReady); i++ {
		candidato := k.ElementoNSinSacar(EstadoReady, i)
		if !EstaReservado(candidato) {
			pcb = candidato
			break
		}
	}
	if pcb == nil {
		utils.LoggerConFormato("No hay procesos disponibles en READY que no estén reservados")
		mutex_ProcesoPorEstado[EstadoReady].Unlock()
		return
	}

	MarcarProcesoReservado(pcb, true)

	pid := pcb.Pid
	pc := pcb.Pc

	mutex_ProcesoPorEstado[EstadoReady].Unlock()

	// intentamos asignarle cpu
	mutex_CPUsConectadas.Lock()

	cpu_seleccionada := k.ObtenerCPULibre()

	if cpu_seleccionada == nil { //no hay cpu libre
		if hay_que_chequear_desalojo {
			mutex_CPUsConectadas.Unlock()
			k.IntentarDesalojoSRT(pid)

		} else {
			mutex_CPUsConectadas.Unlock()
		}
		return
	}

	actualizarCPU(cpu_seleccionada, pid, pc, false)

	idCPU := cpu_seleccionada.ID

	url := cpu_seleccionada.Url

	mutex_CPUsConectadas.Unlock()

	handleDispatch(pid, pc, url)

	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	mutex_ProcesoPorEstado[EstadoReady].Lock()
	defer mutex_ProcesoPorEstado[EstadoReady].Unlock()
	defer mutex_ProcesoPorEstado[EstadoExecute].Unlock()

	//Checkeo por las dudas que sigue en la misma posicion de memoria
	procVerificadoAExecute := k.QuitarYObtenerPCB(EstadoReady, pid, false)

	if procVerificadoAExecute == nil {
		utils.LoggerConFormato("## ERROR (IntentarEnviarProcvesoAExecute), el proceso no esta en la lista %s", estados_proceso[EstadoReady])
		return
	}
	MarcarProcesoReservado(procVerificadoAExecute, false)
	k.AgregarAEstado(EstadoExecute, procVerificadoAExecute, false)

	fmt.Printf("Proceso %d a Execute en CPU %d\n", pid, idCPU)
}

func (k *Kernel) BuscarCPUPorID(id int) *CPU {
	cpu, existe := k.CPUsConectadas[id]
	if !existe {
		return nil
	}
	return cpu
}

func actualizarCPU(cpu *CPU, pid int, pc int, liberar bool) {
	cpu.Esta_libre = liberar
	cpu.Pid = pid
	cpu.Pc = pc
}

func (k *Kernel) temporizadorSuspension(pid int) {
	suspension := time.Duration(k.Configuracion.Tiempo_Suspension) * time.Millisecond
	utils.LoggerConFormato("## (%d) - Temporizador de suspensión iniciado por %v", pid, suspension)

	time.Sleep(suspension)

	mutex_ProcesoPorEstado[EstadoBlock].Lock()
	defer mutex_ProcesoPorEstado[EstadoBlock].Unlock()

	pcb := k.BuscarPorPidSinLock(EstadoBlock, pid)
	if pcb != nil {

		utils.LoggerConFormato("## (%d) - Tiempo de suspensión cumplido, moviendo a SUSPENDED_BLOCKED", pid)

		mutex_ProcesoPorEstado[EstadoBlockSuspended].Lock()
		defer mutex_ProcesoPorEstado[EstadoBlockSuspended].Unlock()

		k.MoverDeEstadoPorPid(EstadoBlock, EstadoBlockSuspended, pid, false)

		return

	}
	utils.LoggerConFormato("## (%d) - Proceso ya no está en BLOCKED, no se suspende", pid)

}

func (k *Kernel) IntentarDesalojoSRT(pidQuiereDesalojar int) {
	mutex_CPUsConectadas.Lock()
	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	mutex_ProcesoPorEstado[EstadoReady].Lock()

	defer mutex_ProcesoPorEstado[EstadoReady].Unlock()
	defer mutex_ProcesoPorEstado[EstadoExecute].Unlock()
	defer mutex_CPUsConectadas.Unlock()

	listaReady := k.ProcesoPorEstado[EstadoReady]

	if len(listaReady) == 0 {
		return //no hay procesos en READY, no tiene sentido desalojar
	}

	procesoCandidato := k.BuscarPorPidSinLock(EstadoReady, pidQuiereDesalojar)

	if procesoCandidato == nil {
		utils.LoggerConFormato("ERROR: proceso %d ya no está en READY", pidQuiereDesalojar)
		return
	}
	estimacionReady := procesoCandidato.SJF.Estimado_actual

	var estimacionMaxRestante float64 = -1
	var cpuElegida *CPU

	for _, cpu := range k.CPUsConectadas {

		procesoEjecutando := k.BuscarPorPidSinLock(EstadoExecute, cpu.Pid)
		if procesoEjecutando == nil {
			fmt.Println("ERROR: el proceso no esta en la lista execute, incosistencia interna")
			return
		}

		tiempoEjecutando := duracionEnEstado(procesoEjecutando)
		estimacionRestante := procesoEjecutando.SJF.Estimado_actual - float64(tiempoEjecutando.Milliseconds())

		if estimacionRestante > estimacionReady && estimacionRestante > estimacionMaxRestante {
			estimacionMaxRestante = estimacionRestante
			cpuElegida = cpu
		}
	}

	if cpuElegida != nil {
		utils.LoggerConFormato("Peticion de desalojo del proceso %d de la CPU %d (estimacion restante: %.2f) para ejecutar el proceso %d (estimacion: %.2f)",
			cpuElegida.Pid,
			cpuElegida.ID,
			estimacionMaxRestante,
			procesoCandidato.Pid,
			estimacionReady)

		mutex_desalojos.Lock()
		k.ProcesosEsperandoDesalojo[cpuElegida.ID] = procesoCandidato.Pid
		mutex_desalojos.Unlock()

		EnviarInterrupt(cpuElegida)
		return
	}

}

// -----------Informa el Club Atletico Velez Sarsfield------------
func (k *Kernel) CambiosEnElPlantel(cpuPosicion *CPU, procesoTitular *PCB, procesoSuplente *PCB) bool {
	// Debutante
	// CALIENTA KAROL

	// Actualizamos pc en el pcb del proceso que estaba ejecutando
	procesoTitular.Pc = cpuPosicion.Pc

	// Ahora si desalojamos al pcb correspondiente
	tiempo_en_cancha := duracionEnEstado(procesoTitular)
	k.actualizarEstimacionSJF(procesoTitular, tiempo_en_cancha)

	utils.LoggerConFormato("## (%d) - Desalojado por algoritmo SJF/SRT", cpuPosicion.Pid)

	if !k.MoverDeEstadoPorPid(EstadoExecute, EstadoReady, procesoTitular.Pid, false) {
		utils.LoggerConFormato("## ERROR (CambiosEnElPlantel) el procesoEjecutando no esta en la lista EXECUTE")
		return false
	}

	// Actualizar la cpu con el proceso nuevo
	cpuPosicion.Pc = procesoSuplente.Pc
	cpuPosicion.Pid = procesoSuplente.Pid

	// Enviar nuevo proceso a cpu
	handleDispatch(cpuPosicion.Pid, cpuPosicion.Pc, cpuPosicion.Url)
	// ENTRA AQUINO (Mi primo, que si aprobo el tp)

	procVerificadoAExecute := k.QuitarYObtenerPCB(EstadoReady, procesoSuplente.Pid, false)

	if procVerificadoAExecute == nil {
		utils.LoggerConFormato("## ERROR (CambiosEnElPlantel) el procesoQuiereEjecutar no esta en la lista READY")
		return false
	}

	MarcarProcesoReservado(procVerificadoAExecute, false)
	k.AgregarAEstado(EstadoExecute, procVerificadoAExecute, false)

	utils.LoggerConFormato("(%d) Pasa del estado <%s> al estado <%s>",
		procesoSuplente.Pid,
		estados_proceso[EstadoReady],
		estados_proceso[EstadoExecute])

	fmt.Printf("CAMBIO: Sale %d (est. %.2f), entra %d (est. %.2f)\n", // Leer con voz de gangoso
		procesoTitular.Pid, procesoTitular.SJF.Estimado_actual,
		procesoSuplente.Pid, procesoSuplente.SJF.Estimado_actual)
	return true
}

func EnviarInterrupt(cpu *CPU) { // yo te hablo por la puerta interrupt y me desocupo
	fullURL := fmt.Sprintf("%s/interrupt", cpu.Url)
	utils.EnviarStringSinEsperar("POST", fullURL, "")
}

func handleDispatch(pid int, pc int, url string) {

	fullURL := fmt.Sprintf("%s/dispatch", url)

	datos := fmt.Sprintf("%d %d", pid, pc)
	utils.EnviarStringSinEsperar("POST", fullURL, datos)
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

	utils.LoggerConFormato("(llegaNuevaCPU) con mensaje: %s\n", mensajeCPU)

	if !k.registrarNuevaCPU(mensajeCPU) {
		http.Error(w, "No se pudo registar la CPU", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
func (k *Kernel) llegoFinInterrupcion(w http.ResponseWriter, r *http.Request) {
	var mensajeCPU string
	if err := json.NewDecoder(r.Body).Decode(&mensajeCPU); err != nil {
		fmt.Println("Error recibiendo la solicitud:", err)
		http.Error(w, "Error en el formato de la solicitud", http.StatusBadRequest)
		return
	}

	utils.LoggerConFormato("(llegoFinInterrupcion) con mensaje: %s\n", mensajeCPU)

	idCPU, pidDesalojado, pcActualizado, err := decodificarMensajeFinInterrupcion(mensajeCPU)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Ya validado, respondemos OK
	w.WriteHeader(http.StatusOK)

	go k.antenderFinInterrupcion(idCPU, pidDesalojado, pcActualizado)

	utils.LoggerConFormato("Fin de antenderFinInterrupcion para CPU %d", idCPU)

}
func decodificarMensajeFinInterrupcion(mensaje string) (idCPU, pid, pc int, err error) {
	aux := strings.Split(mensaje, " ")
	if len(aux) != 3 {
		return 0, 0, 0, fmt.Errorf("esperando formato 'ID PID PC'")
	}
	idCPU, err1 := strconv.Atoi(aux[0])
	pid, err2 := strconv.Atoi(aux[1])
	pc, err3 := strconv.Atoi(aux[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, fmt.Errorf("valores inválidos: %v %v %v", err1, err2, err3)
	}
	return idCPU, pid, pc, nil
}

func (k *Kernel) antenderFinInterrupcion(idCPU, pidDesalojado, pcActualizado int) {
	mutex_desalojos.Lock()
	defer mutex_desalojos.Unlock()
	mutex_CPUsConectadas.Lock()
	defer mutex_CPUsConectadas.Unlock()
	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	defer mutex_ProcesoPorEstado[EstadoExecute].Unlock()
	mutex_ProcesoPorEstado[EstadoReady].Lock()
	defer mutex_ProcesoPorEstado[EstadoReady].Unlock()

	cpu := k.BuscarCPUPorID(idCPU)

	if cpu == nil {
		utils.LoggerConFormato("ERROR (antenderFinInterrupcion) NO se encontró la CPU con ID %d\n",
			idCPU)
		return
	}
	cpu.Pc = pcActualizado

	procesoEjecutando := k.BuscarPorPidSinLock(EstadoExecute, pidDesalojado)

	if procesoEjecutando == nil {
		utils.LoggerConFormato("ERROR (antenderFinInterrupcion) NO se encontró el proceso %d en Execute\n",
			pidDesalojado)
		return
	}
	pidQuiereEjecutar, hayEsperando := k.ProcesosEsperandoDesalojo[idCPU]

	if !hayEsperando {
		utils.LoggerConFormato("ERROR (antenderFinInterrupcion) No hay proceso esperando desalojo para CPU %d\n", idCPU)
		return
	}
	delete(k.ProcesosEsperandoDesalojo, idCPU)

	procesoQuiereEjecutar := k.BuscarPorPidSinLock(EstadoReady, pidQuiereEjecutar)

	if procesoQuiereEjecutar == nil {
		utils.LoggerConFormato("ERROR (antenderFinInterrupcion) NO se encontró el proceso %d en Ready\n",
			pidQuiereEjecutar)
		return
	}

	if !k.CambiosEnElPlantel(cpu, procesoEjecutando, procesoQuiereEjecutar) {
		utils.LoggerConFormato("ERROR (antenderFinInterrupcion) NO se pudo realizar los cambiosEnElPLantel\n")
		return
	}
}
