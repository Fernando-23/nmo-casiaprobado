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

// Mueve a READY consultando a memoria si tiene espacio
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

func (k *Kernel) OrdenarYConfirmarSiHayQueChequearDesalojo() bool {
	lista_ready := k.ProcesoPorEstado[EstadoReady]
	pcb_nuevo_pid := lista_ready[len(lista_ready)-1].Pid

	sort.Sort(PorSJF(lista_ready))

	if pcb_nuevo_pid == lista_ready[0].Pid {
		slog.Debug("Debug - (OrdenarYConfirmarSiHayQueChequearDesalojo) - Hay que desalojar", "pid", pcb_nuevo_pid)
		return true
	}

	return false
}

func (k *Kernel) SoloOrdenarPorAlgoritmoREADY() {
	// Para FIFO ya esta preparada la lista
	if k.Configuracion.Algoritmo_Plani == "SJF" || k.Configuracion.Algoritmo_Plani == "SRT" {
		sort.Sort(PorSJF(k.ProcesoPorEstado[EstadoReady]))
	}

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
		slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorPID) - No encontre una cpu libre, se va a entrar al switch case")
		switch algoritmo_corto_plazo {
		case "SRT":
			//esto no esta chequeado, chequear bien como usamos cpu_desalojo
			cpu_desalojo := k.ChequearDesalojo(proc_a_dispatch)
			if cpu_desalojo != nil {
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
	slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorPID) - Encontre una cpu libre", "id_cpu", cpu_seleccionada.ID)
	//caso lindo, hay cpu libre
	pid_dispatch := proc_a_dispatch.Pid
	pc_dispatch := proc_a_dispatch.Pc

	id_cpu := cpu_seleccionada.ID
	url_cpu := cpu_seleccionada.Url

	//mando a laburar al proceso a la cpu libre
	handleDispatch(pid_dispatch, pc_dispatch, url_cpu)
	slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorPID) - Envie a dispatch",
		"pid_enviado", pid_dispatch, "id_cpu", id_cpu)
	// verifico si esta en el mismo espacio de memoria, lo saco
	proceso_enviado_a_exec := k.QuitarYObtenerPCB(EstadoReady, pid_dispatch, false)

	if proceso_enviado_a_exec == nil {
		slog.Warn("Cuidadito - (IntentarEnviarProcesoAExecute) - El proceso no esta en la lista READY", "pid", pid_dispatch)
		return
	}

	actualizarCPU(cpu_seleccionada, pid_dispatch, pc_dispatch, false)

	//movemos a la lista correspondiente
	k.AgregarAEstado(EstadoExecute, proceso_enviado_a_exec, false)

	slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorPID)- Proceso enviado a EXECUTE", "pid", pid_dispatch, "cpu_id", id_cpu)
}

// NO me asegura que este la CPU este libre, no tiene sentido pero soluciona tema deadlock
func (k *Kernel) IntentarEnviarProcesoAExecutePorCPU(cpu_a_dispatch *CPU) {
	mutex_ProcesoPorEstado[EstadoReady].Lock()
	defer mutex_ProcesoPorEstado[EstadoReady].Unlock()

	slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorCPU) - Entre IntentarEnviarProcesoAExecutePorCPU con la CPU",
		"id_cpu", cpu_a_dispatch.ID)

	// chequeamos si no hay nadie en READY asi no laburamos al dope
	if !k.TieneProcesos(EstadoReady) {
		slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorCPU) - No hay procesos en READY")
		return
	}

	// sacamos al primer elemento de la lisat ready
	k.SoloOrdenarPorAlgoritmoREADY()
	primer_elemento_READY := k.ProcesoPorEstado[EstadoReady][0]
	// le hago un pop de ready
	proc := k.QuitarYObtenerPCB(EstadoReady, primer_elemento_READY.Pid, false)

	if proc == nil {
		slog.Warn("Cuidadito - (IntentarEnviarProcesoAExecutePorCPU) - No esta en READY", "pid", primer_elemento_READY.Pid)
		return
	}

	// mandamos el proceso a los de la CPU
	handleDispatch(proc.Pid, proc.Pc, cpu_a_dispatch.Url)

	actualizarCPU(cpu_a_dispatch, proc.Pid, proc.Pc, false)

	// lo mandamos a execute
	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	k.AgregarAEstado(EstadoExecute, proc, false)
	mutex_ProcesoPorEstado[EstadoExecute].Unlock()

	slog.Debug("Debug - (IntentarEnviarProcesoAExecutePorCPU)- Proceso enviado a EXECUTE", "pid", proc.Pid, "cpu_id", cpu_a_dispatch.ID)
}

func (k *Kernel) GestionDeExecute(id int) {

	cod_op := "SOY_UNA_CPU"

	if id == -2 && k.Configuracion.Algoritmo_Plani == "SRT" {
		cod_op = "NO_SOY_UNA_CPU_Y_QUIERO_INTENTAR_DESALOJO"
	} else if id == -2 {
		cod_op = "NO_SOY_UNA_CPU_Y_ALGUIEN_LLEGO_A_READY"
	}

	switch cod_op {
	//CASO 1, EL MAS LINDO, UN SIMPLE DISPATCH
	case "SOY_UNA_CPU":
		mutex_CPUsConectadas.Lock()
		cpu := k.CPUsConectadas[id]
		cpu.Esta_libre = false
		mutex_CPUsConectadas.Unlock()

		mutex_CPUsConectadasPorId[cpu.ID].Lock()
		k.IntentarEnviarProcesoAExecutePorCPU(cpu)
		mutex_CPUsConectadasPorId[cpu.ID].Unlock()

	//CASO 2, ALGUIEN LLEGO A READY Y TENGO CPUs LIBRES
	case "NO_SOY_UNA_CPU_Y_ALGUIEN_LLEGO_A_READY":
		mutex_CPUsConectadas.Lock()
		potencial_cpu_libre := k.ObtenerCPULibre()
		mutex_CPUsConectadas.Unlock()

		if potencial_cpu_libre == nil {
			slog.Debug("Debug - (GestionDeExecute) - Tengo procesos en READY, no hay alg de desalojo, pero no tengo CPUs disponibles")
			return
		}

		mutex_CPUsConectadasPorId[potencial_cpu_libre.ID].Lock()
		k.IntentarEnviarProcesoAExecutePorCPU(potencial_cpu_libre)
		mutex_CPUsConectadasPorId[potencial_cpu_libre.ID].Unlock()

	//CASO 3, EL CONTROVERSIAL. ALGUIEN LLEGO A READY Y ENCIMA HAY ALGORITMO SRT
	case "NO_SOY_UNA_CPU_Y_QUIERO_INTENTAR_DESALOJO":
		mutex_CPUsConectadas.Lock()
		potencial_cpu_libre := k.ObtenerCPULibre()
		mutex_CPUsConectadas.Unlock()

		//CASO 3.1 - FEO - ALGORITMO SRT Y NO HAY CPU LIBRE
		if potencial_cpu_libre == nil {

			slog.Debug("Debug - (GestionDeExecute) - Tengo procesos en READY y tengo alg READY SRT, voy a chequear desalojo")
			mutex_ProcesoPorEstado[EstadoReady].Lock()
			defer mutex_ProcesoPorEstado[EstadoReady].Unlock()
			k.OrdenarPorAlgoritmoREADY()
			slog.Debug("Debug - (GestionDeExecute) - Ordene la lista READY para potencial desalojo")
			primer_elemento_READY := k.ProcesoPorEstado[EstadoReady][0]

			mutex_CPUsConectadas.Lock()
			mutex_ProcesoPorEstado[EstadoExecute].Lock()
			cpu_potencial_a_desalojar := k.ChequearDesalojo(primer_elemento_READY)
			mutex_ProcesoPorEstado[EstadoExecute].Unlock()

			if cpu_potencial_a_desalojar != nil {
				slog.Debug("Debug - (GestionDeExecute) - Hay que desalojar, cumple condicion de menor estimacion",
					"pid a entrar", primer_elemento_READY.Pid, "id de cpu a detonar", cpu_potencial_a_desalojar.ID, "pid a detonar", cpu_potencial_a_desalojar.Pid)

				//me "reservo" la cpu antes de liberar la global de cpu
				mutex_CPUsConectadasPorId[cpu_potencial_a_desalojar.ID].Lock()
				cpu_potencial_a_desalojar.Esta_libre = false
				//ahora si, dejo libre la global para otros
				mutex_CPUsConectadas.Unlock()

				mutex_ProcesoPorEstado[EstadoExecute].Lock()
				k.RealizarDesalojo(cpu_potencial_a_desalojar, primer_elemento_READY.Pid)
				mutex_ProcesoPorEstado[EstadoExecute].Unlock()

				mutex_CPUsConectadasPorId[cpu_potencial_a_desalojar.ID].Unlock()
				return
			}

			slog.Debug("Debug - (GestionDeExecute) - No hay que desalojar, no cumple condicion de menor estimacion",
				"pid candidato que fallo", primer_elemento_READY.Pid)
			mutex_CPUsConectadas.Unlock()
			return
		}

		//CASO 3.2 - LINDO - ES ALGORITMO SRT PERO HAY CPU LIBRE
		mutex_CPUsConectadasPorId[potencial_cpu_libre.ID].Lock()
		k.IntentarEnviarProcesoAExecutePorCPU(potencial_cpu_libre)
		mutex_CPUsConectadasPorId[potencial_cpu_libre.ID].Unlock()
	}

}
