package main

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

// aka chequea si hay que intentar enviar un proceso a READY
func (k *Kernel) hayQuePlanificarAccesoAReady(estadoOrigen int, pid int) bool {
	algoritmoPlani := k.Configuracion.Ready_ingress_algorithm
	lProcEstado := k.ProcesoPorEstado[estadoOrigen]

	if len(lProcEstado) > 1 {
		return true
	}

	if algoritmoPlani == "PCMP" && k.esProcesoMasChico(pid, estadoOrigen) {
		return true
	}

	return false
}

// podria tener el pid
// algo importante es que no hay que sacar un proceso de una lista
// a menos que la vyaas a meter a otra
func (k *Kernel) UnicoEnNewYNadaEnSuspReady() (bool, bool) { //el primero es si es unico y el segundo si pudo pasar a ready
	mutex_peticionHayEspacioMemoria.Lock()
	defer mutex_peticionHayEspacioMemoria.Unlock()

	mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()
	mutex_ProcesoPorEstado[EstadoNew].Lock()

	lista_new := k.ProcesoPorEstado[EstadoNew]
	lista_susp_ready := k.ProcesoPorEstado[EstadoReadySuspended]

	//Caso en en que hay exactamente un proceso en NEW y ninguno en SUSPENDED-READY
	procCandidatoAReady := k.PrimerElementoSinSacar(EstadoNew)
	if procCandidatoAReady == nil {
		slog.Error("Error - (UnicoEnNewYNadaEnSuspReady) - no hay procesos en new")
		return false, false
	}

	if len(lista_susp_ready) == 0 && len(lista_new) == 1 {

		slog.Debug("Único proceso de new y no hay procesos en SUSP READY",
			"pid", procCandidatoAReady.Pid,
		)

		// me clono los datos por las dudas no vaya ser que el puntero apunte a otro lado
		pid := procCandidatoAReady.Pid
		tamanio := procCandidatoAReady.Tamanio
		arch_pseudo := procCandidatoAReady.Arch_pseudo

		//Liberamos recursos por peticion http
		mutex_ProcesoPorEstado[EstadoNew].Unlock()
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()

		entro, err := k.gestionarAccesoAReady(pid, tamanio, arch_pseudo, EstadoNew)

		if err != nil {
			slog.Error("Error - dentro de..(UnicoEnNewYNadaEnSuspReady)", "error", err)
			return true, false
		}

		if entro {
			slog.Warn("Cuidadito - (ListaNewSoloYo) - Mande de NEW a READY",
				"pid", pid)
			k.IntentarEnviarProcesoAExecutePorCPU(nil)

			return true, true
		}
		return true, false
	}

	slog.Debug("No es unico proceso en new o hay procesos en susp ready",
		"pid", procCandidatoAReady.Pid,
	)

	mutex_ProcesoPorEstado[EstadoNew].Unlock()
	mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
	return false, false
}

func (k *Kernel) IntentarEnviarProcesoAReady(estadoOrigen int, pidQuiereEntrar int) {
	slog.Debug("uh mal ahi huayo, habia procesos en new o el susp ready", "pid", pidQuiereEntrar)

	mutex_peticionHayEspacioMemoria.Lock()
	defer mutex_peticionHayEspacioMemoria.Unlock()

	mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()
	mutex_ProcesoPorEstado[EstadoNew].Lock()

	// Si intento mover NEW, pero hay procesos en READY_SUSPENDED, no hago nada
	if estadoOrigen == EstadoNew && k.TieneProcesos(EstadoReadySuspended) {
		mutex_ProcesoPorEstado[EstadoNew].Unlock()
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
		slog.Debug("Debug - (IntentarEnviarProcesoAReady) - proceso en NEW y hay procesos en SUSP-READY", "pid", pidQuiereEntrar)
		return
	}

	if !k.TieneProcesos(estadoOrigen) {
		mutex_ProcesoPorEstado[EstadoNew].Unlock()
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
		slog.Debug("Debug - (IntentarEnviarProcesoAReady) - No hay procesos en estado",
			"estado", estados_proceso[estadoOrigen],
			"pid", pidQuiereEntrar,
		)
		return
	}

	if !k.hayQuePlanificarAccesoAReady(estadoOrigen, pidQuiereEntrar) {

		slog.Debug("Debug - (IntentarEnviarProcesoAReady) - No te toca fernandito",
			"estado", estados_proceso[estadoOrigen],
			"pid", pidQuiereEntrar,
		)

		mutex_ProcesoPorEstado[EstadoNew].Unlock()
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
		return
	}

	if k.Configuracion.Ready_ingress_algorithm == "PCMP" {
		sort.Sort(PorTamanio(k.ProcesoPorEstado[estadoOrigen]))
	}

	procCandidatoAReady := k.PrimerElementoSinSacar(estadoOrigen)

	// Verifico que el primer proceso candidato sea el que quiere entrar
	if procCandidatoAReady.Pid != pidQuiereEntrar {
		slog.Debug("Error - (IntentarEnviarProcesoAReady) - pid distinto del que quiere entrar",
			"estado", estadoOrigen,
			"pid", pidQuiereEntrar,
		)
		mutex_ProcesoPorEstado[EstadoNew].Unlock()
		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
		return
	}

	// me clono los datos por las dudas no vaya ser que el puntero apunte a otro lado
	pid := procCandidatoAReady.Pid
	tamanio := procCandidatoAReady.Tamanio
	arch_pseudo := procCandidatoAReady.Arch_pseudo

	//Libero los mutex antes de hacer la peticion HTTP (que puede tardar)
	mutex_ProcesoPorEstado[EstadoNew].Unlock()
	mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()

	// consulto memoria
	entro, err := k.gestionarAccesoAReady(pid, tamanio, arch_pseudo, estadoOrigen)
	if err != nil {
		slog.Error("Error", "error", err)
		return
	}

	if entro {
		slog.Warn("Cuidadito - (IntentarEnviarProcesoAReady) - Mande de NEW a READY",
			"pid", pid)
		k.IntentarEnviarProcesoAExecutePorCPU(nil)
		return
	}
	slog.Debug("Debug - (IntentarEnviarProcesoAReady) - No paso a READY ",
		"pid", pid, "estado_origen", estadoOrigen)
}

// CHUSMEAR LUNES 14
func (k *Kernel) IntentarEnviarProcesosAReady() {
	mutex_peticionHayEspacioMemoria.Lock()
	defer mutex_peticionHayEspacioMemoria.Unlock()
	slog.Debug("Debug - (IntentarEnviarProcesosAReady) - Como minimo, entre a esta funcion")
	estados := []int{EstadoReadySuspended, EstadoNew}

	//k.PlanificarLargoPorLista(EstadoReadySuspended)
	//k.PlanificarLargoPorLista(EstadoNew)

	for _, estado := range estados {
		mutex_ProcesoPorEstado[estado].Lock()

		k.PlanificarLargoPorLista(estado)

		for k.TieneProcesos(estado) {
			proc := k.PrimerElementoSinSacar(estado)
			pid := proc.Pid
			tamanio := proc.Tamanio
			arch_pseudo := proc.Arch_pseudo

			// Desbloquear mutexes para evitar deadlocks durante llamada externa
			mutex_ProcesoPorEstado[estado].Unlock()

			exito, err := k.gestionarAccesoAReady(pid, tamanio, arch_pseudo, estado)
			if err != nil {
				slog.Error("Error - (IntentarEnviarProcesosAReady) - Error en gestionarAccesoAReady", "error", err)
				return
			}

			slog.Debug("Debug - (IntentarEnviarProcesosAReady) - Realice gestionarAccesoAReady")

			if !exito {
				// Rebloquear mutexes antes de salir para mantener consistencia
				slog.Debug("Debug - (IntentarEnviarProcesosAReady) - No hubo exito al pasar a READY")
				return
			}

			slog.Warn("Cuidadito - (IntentarEnviarProcesosAReady) - Mande de NEW a READY",
				"pid", pid)

			//k.IntentarEnviarProcesoAExecute()

			// Rebloquear mutexes para siguiente iteración
			mutex_ProcesoPorEstado[estado].Lock()
		}
		mutex_ProcesoPorEstado[estado].Unlock()
		//break
	}
}

func (k *Kernel) PlanificarLargoPorLista(codLista int) {

	//si el algoritmo es PMCP, ordenamos antes de tomar el primero
	if k.Configuracion.Ready_ingress_algorithm == "PMCP" {
		sort.Sort(PorTamanio(k.ProcesoPorEstado[codLista]))
	}
}

func (k *Kernel) gestionarAccesoAReady(pid int, tamanio int, arch_pseudo string, estadoOrigen int) (bool, error) {

	//Consultamos memoria ...

	var hayEspacio bool
	var err error

	switch estadoOrigen {
	case EstadoReadySuspended:
		hayEspacio, err = EnviarDesuspension(pid)
	case EstadoNew:
		hayEspacio, err = k.MemoHayEspacio(pid, tamanio, arch_pseudo)
	}

	if err != nil {
		return false, fmt.Errorf("error: %w - Pid: %d - Estado: %s - (gestionarAccesoAReady) - Peticion espacio en memoria",
			err, pid, estados_proceso[estadoOrigen])
	}

	mutex_ProcesoPorEstado[estadoOrigen].Lock()
	defer mutex_ProcesoPorEstado[estadoOrigen].Unlock()

	pcb := k.BuscarPorPidSinLock(estadoOrigen, pid)
	if pcb == nil {
		return false, fmt.Errorf("error - Pid: %d - (gestionarAccesoAReady) - Pcb no encontrado - Estado %s", pid, estados_proceso[estadoOrigen])
	}

	if hayEspacio {

		slog.Debug("Debug - (gestionarAccesoAReady) - Intentando mover proceso a READY",
			"pid", pid,
			"estado_origen", estados_proceso[estadoOrigen],
		)
		mutex_ProcesoPorEstado[EstadoReady].Lock()
		defer mutex_ProcesoPorEstado[EstadoReady].Unlock()

		//Checkeo por las dudas que sigue en la misma posicion de memoria
		procVerificadoAReady := k.QuitarYObtenerPCB(estadoOrigen, pid, false)

		if procVerificadoAReady == nil {
			return false, fmt.Errorf("error - Pid: %d - (gestionarAccesoAReady) - El proceso no está en la lista %s", pid, estados_proceso[estadoOrigen])
		}

		k.AgregarAEstado(EstadoReady, procVerificadoAReady, false)

		utils.LoggerConFormato("(%d) Pasa del estado <%s> al estado <%s>", pid, estados_proceso[estadoOrigen], estados_proceso[EstadoReady])

		return true, nil
	}

	slog.Debug("Debug - (gestionarAccesoAReady) - No hay espacio en memoria para mover el proceso a READY",
		"pid", pid,
		"estado_origen", estados_proceso[estadoOrigen],
	)

	return false, nil
}

func (k *Kernel) OrdenarPorAlgoritmoREADY() bool {

	// Para FIFO ya esta preparada la lista

	if k.Configuracion.Algoritmo_Plani == "SJF" || k.Configuracion.Algoritmo_Plani == "SRT" {
		lista_ready := k.ProcesoPorEstado[EstadoReady]
		pcb_nuevo_pid := lista_ready[len(lista_ready)-1].Pid

		sort.Sort(PorSJF(lista_ready)) //SJF distinto de nil

		if k.Configuracion.Algoritmo_Plani == "SRT" && pcb_nuevo_pid == lista_ready[0].Pid { //  10 15 18 31 32 500
			slog.Debug("(PlaniCortoPlazo) - Hay que desalojar", "pid", pcb_nuevo_pid)
			return true
		}
	}
	return false
}

func (k *Kernel) temporizadorSuspension(pid int) {
	suspension := time.Duration(k.Configuracion.Tiempo_Suspension) * time.Millisecond
	utils.LoggerConFormato("## (%d) - Temporizador de suspensión iniciado por %v", pid, suspension)

	time.Sleep(suspension)

	mutex_ProcesoPorEstado[EstadoBlock].Lock()
	defer mutex_ProcesoPorEstado[EstadoBlock].Unlock()

	pcb := k.BuscarPorPidSinLock(EstadoBlock, pid)
	if pcb != nil {

		slog.Debug("Debug - (temporizadorSuspension) - Tiempo de suspensión cumplido, moviendo a SUSPENDED_BLOCKED",
			"pid", pid,
		)

		mutex_ProcesoPorEstado[EstadoBlockSuspended].Lock()
		defer mutex_ProcesoPorEstado[EstadoBlockSuspended].Unlock()

		if !k.MoverDeEstadoPorPid(EstadoBlock, EstadoBlockSuspended, pid, false) {
			slog.Debug("Debug - (temporizadorSuspension) - Proceso ya no está en BLOCKED, no se suspende",
				"pid", pid,
			)
			return
		}

		EnviarSuspension(pid)
		return
	}
}

// NO me asegura que este la CPU este libre, no tiene sentido pero soluciona tema deadlock
func (k *Kernel) IntentarEnviarProcesoAExecutePorCPU(cpu_a_dispatch *CPU) {
	mutex_ProcesoPorEstado[EstadoReady].Lock()
	defer mutex_ProcesoPorEstado[EstadoReady].Unlock()
	if cpu_a_dispatch == nil {
		slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorCPU - Entre sin una CPU")
	}

	slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorCPU) - Entre IntentarEnviarProcesoAExecutePorCPU con la CPU",
		"id_cpu", cpu_a_dispatch.ID)

	// chequeamos si no hay nadie en READY asi no laburamos al dope
	if !k.TieneProcesos(EstadoReady) {
		slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorCPU) - No hay procesos en READY")
		return
	}

	// sacamos al primer elemento de la lisat ready
	pcb := k.PrimerElementoSinSacar(EstadoReady)

	pid := pcb.Pid
	pc := pcb.Pc

	mutex_CPUsConectadas.Lock()
	defer mutex_CPUsConectadas.Unlock()
	if !cpu_a_dispatch.Esta_libre {
		slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorCPU - En ese pequenio intervalo, me detonaron la cpu")
		return
	}

	// mandamos el proceso a los de la CPU (estos no se salvan, un dolor de cabeza)
	handleDispatch(pid, pc, cpu_a_dispatch.Url)

	actualizarCPU(cpu_a_dispatch, pid, pc, false)

	// le hago un pop de ready
	proc := k.QuitarYObtenerPCB(EstadoReady, pid, false)

	if proc == nil {
		slog.Warn("Cuidadito - (IntentarEnviarProcesoAExecutePorCPU) - No esta en READY", "pid", pid)
		return
	}
	// lo mandamos a execute
	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	k.AgregarAEstado(EstadoExecute, proc, false)
	mutex_ProcesoPorEstado[EstadoExecute].Unlock()

	slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorCPU)- Proceso enviado a EXECUTE", "pid", pid, "cpu_id", cpu_a_dispatch.ID)
}

// esta previamente tomado el mutex de READY
// tuviste que haber ordenado, hecho un guiso, etc etc (todo lo relacionado a READY previamente basicamente)
// esta mas enfocada para SRT
func (k *Kernel) IntentarEnviarProcesoAExecutePorPID(proc_a_dispatch *PCB) {
	algoritmo_corto_plazo := k.Configuracion.Algoritmo_Plani
	mutex_CPUsConectadas.Lock()
	defer mutex_CPUsConectadas.Unlock()
	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	defer mutex_ProcesoPorEstado[EstadoExecute].Unlock()

	cpu_seleccionada := k.ObtenerCPULibre()

	if cpu_seleccionada == nil {
		switch algoritmo_corto_plazo {
		case "SRT":
			hay_que_desalojar, cpu_desalojo := k.ChequearDesalojo(proc_a_dispatch)
			if !hay_que_desalojar {
				slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorPID) - En ChequearDesalojo, se retorno false y no hay que desalojar")
				return
			}
			slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorPID) - En ChequearDesalojo, se retorno true y hay que desalojar")
			//como ya hicimos todos los chequeos, si o si desalojamos
			k.RealizarDesalojo(cpu_desalojo, proc_a_dispatch.Pid)
			return
		default:
			slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorPID) - No hay CPUs libres")
			return
		}
	}

	//caso lindo, hay cpu libre
	pid_dispatch := proc_a_dispatch.Pid
	pc_dispatch := proc_a_dispatch.Pc

	id_cpu := cpu_seleccionada.ID
	url_cpu := cpu_seleccionada.Url

	//mando a laburar al proceso a la cpu libre
	handleDispatch(pid_dispatch, pc_dispatch, url_cpu)

	// verifico si esta en el mismo espacio de memoria, lo saco
	proceso_enviado_a_exec := k.QuitarYObtenerPCB(EstadoReady, pid_dispatch, false)

	if proceso_enviado_a_exec == nil {
		slog.Warn("Cuidadito - (IntentarEnviarProcesoAExecute) - El proceso no esta en la lista READY", "pid", pid_dispatch)
		return
	}

	actualizarCPU(cpu_seleccionada, pid_dispatch, pc_dispatch, false)

	//movemos a la lista correspondiente
	k.AgregarAEstado(EstadoExecute, proceso_enviado_a_exec, false)

	slog.Debug("Debug - (IntentarEnviarProcesoAExecute)- Proceso enviado a EXECUTE", "pid", pid_dispatch, "cpu_id", id_cpu)
}

// // No hay pid, no hay cpu, bien general y no muy linda
// func (k *Kernel) IntentarEnviarProcesoAExecute() {
// 	mutex_ProcesoPorEstado[EstadoReady].Lock()

// 	if !k.TieneProcesos(EstadoReady) {
// 		slog.Debug("Debug - (IntentarEnviarProcesoAExecute) - No hay procesos en READY")
// 		mutex_ProcesoPorEstado[EstadoReady].Unlock()
// 		return
// 	}
// 	// QUE DEVUELVA EL 1ER ELEMENTO DE READY, ASI ORGANIZAMOS BIEN
// 	hay_que_chequear_desalojo := k.OrdenarPorAlgoritmoREADY()

// 	//tomamos al primer PCB no reservado para la planificacion
// 	var pcb *PCB
// 	listaReady := k.ProcesoPorEstado[EstadoReady]

// 	if k.Configuracion.Algoritmo_Plani == "SRT" {
// 		for _, candidato := range listaReady {
// 			if !EstaReservado(candidato) { // si esta en la lista negra, no hago nada
// 				pcb = candidato //si no esta es candidato
// 				break
// 			}
// 		}
// 	} else {
// 		// los reales lo hacemos asi, nada de funciones
// 		pcb = listaReady[0]
// 		//pcb = k.PrimerElementoSinSacar(EstadoReady)

// 	}

// 	if pcb == nil {
// 		slog.Debug("Debug - (IntentarEnviarProcesoAExecute) - No hay procesos disponibles en READY")
// 		mutex_ProcesoPorEstado[EstadoReady].Unlock()
// 		return
// 	}

// 	if k.Configuracion.Algoritmo_Plani == "SRT" {
// 		ReservarSRT(pcb, "ESPERANDO CPU") // lo agregamos a la lista negra
// 	}

// 	pid := pcb.Pid
// 	pc := pcb.Pc
// 	mutex_ProcesoPorEstado[EstadoReady].Unlock()

// 	// buscamos una cpu libre
// 	mutex_CPUsConectadas.Lock()
// 	cpu_seleccionada := k.ObtenerCPULibre()
// 	mutex_CPUsConectadas.Unlock()

// 	if cpu_seleccionada == nil { // si no hay CPU LIBRE
// 		if hay_que_chequear_desalojo {
// 			slog.Debug("Debug - (IntentarEnviarProcesoAExecute) - No hay CPU libre, intentando desalojo por SRT", "pid", pid)
// 			if !k.IntentarDesalojoSRT(pid) {
// 				ReservarSRT(pcb, "NO") //lo sacamos al vende humo de la lista negra por ahora
// 			}
// 		} else {
// 			slog.Debug("Debug - (IntentarEnviarProcesoAExecute) - No hay CPU libre y no se requiere desalojo", "pid", pid)
// 			ReservarSRT(pcb, "NO") //lo sacamos al vende humo de la lista negra por ahora
// 		}
// 		return
// 	}

// 	//voy a reservar la cpu
// 	if k.Configuracion.Algoritmo_Plani == "SRT" {
// 		reservarCPU(cpu_seleccionada, pid)
// 	}

// 	idCPU := cpu_seleccionada.ID
// 	url := cpu_seleccionada.Url

// 	mutex_handleDispatch.Lock()
// 	handleDispatch(pid, pc, url)
// 	mutex_handleDispatch.Unlock()

// 	// verifico si esta en el mismo espacio de memoria, lo saco
// 	mutex_ProcesoPorEstado[EstadoReady].Lock()
// 	procVerificadoAExecute := k.QuitarYObtenerPCB(EstadoReady, pid, false)
// 	mutex_ProcesoPorEstado[EstadoReady].Unlock()

// 	if procVerificadoAExecute == nil {
// 		slog.Warn("Cuidadito - (IntentarEnviarProcesoAExecute) - El proceso no esta en la lista READY", "pid", pid)
// 		return
// 	}

// 	mutex_CPUsConectadas.Lock()
// 	actualizarCPU(cpu_seleccionada, pid, pc, false)
// 	mutex_CPUsConectadas.Unlock()

// 	//aca lo mandamos a execute
// 	mutex_ProcesoPorEstado[EstadoExecute].Lock()
// 	k.AgregarAEstado(EstadoExecute, procVerificadoAExecute, false)
// 	mutex_ProcesoPorEstado[EstadoExecute].Unlock()

// 	ReservarSRT(procVerificadoAExecute, "NO") //ahora pequenio pcb es un ninio bueno, lo saco de la lista negra
// 	slog.Debug("Debug - (IntentarEnviarProcesoAExecute)- Proceso enviado a EXECUTE", "pid", pid, "cpu_id", idCPU)
// }
