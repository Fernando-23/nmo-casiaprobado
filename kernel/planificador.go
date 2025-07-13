package main

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

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
			slog.Error("Error", "error", err)
			return true, false
		}

		if entro {
			return true, true
		}
		return true, false
	}

	slog.Debug("No es unico proceso en new o hay procesos en susp ready",
		"pid", procCandidatoAReady.Pid,
	)

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
		k.IntentarEnviarProcesoAExecute()
		return
	}
	slog.Debug("Debug - (IntentarEnviarProcesoAReady) - No paso a READY ",
		"pid", pid, "estado_origen", estadoOrigen)
}

func (k *Kernel) IntentarEnviarProcesosAReady() {
	mutex_peticionHayEspacioMemoria.Lock()
	defer mutex_peticionHayEspacioMemoria.Unlock()

	estados := []int{EstadoReadySuspended, EstadoNew}

	for _, estado := range estados {
		mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()
		mutex_ProcesoPorEstado[EstadoNew].Lock()

		k.PlanificarLargoPorLista(estado)

		for k.TieneProcesos(estado) {
			proc := k.PrimerElementoSinSacar(estado)
			pid := proc.Pid
			tamanio := proc.Tamanio
			arch_pseudo := proc.Arch_pseudo

			// Desbloquear mutexes para evitar deadlocks durante llamada externa
			mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
			mutex_ProcesoPorEstado[EstadoNew].Unlock()

			exito, err := k.gestionarAccesoAReady(pid, tamanio, arch_pseudo, estado)
			if err != nil {
				slog.Error("Error en gestionarAccesoAReady", "error", err)
				return
			}

			if !exito {
				// Rebloquear mutexes antes de salir para mantener consistencia
				mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()
				mutex_ProcesoPorEstado[EstadoNew].Lock()
				break
			}

			k.IntentarEnviarProcesoAExecute()

			// Rebloquear mutexes para siguiente iteración
			mutex_ProcesoPorEstado[EstadoReadySuspended].Lock()
			mutex_ProcesoPorEstado[EstadoNew].Lock()
		}

		mutex_ProcesoPorEstado[EstadoReadySuspended].Unlock()
		mutex_ProcesoPorEstado[EstadoNew].Unlock()

		// Si el estado fue EstadoReadySuspended y ya no hay procesos, hace EstadoNew
		if estado == EstadoReadySuspended && !k.TieneProcesos(EstadoReadySuspended) {
			continue
		}
		break
	}
}

func (k *Kernel) PlanificarLargoPorLista(codLista int) {

	//si el algoritmo es PCMP, ordenamos antes de tomar el primero
	if k.Configuracion.Ready_ingress_algorithm == "PCMP" {
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

func (k *Kernel) IntentarEnviarProcesoAExecute() {
	mutex_ProcesoPorEstado[EstadoReady].Lock()
	if !k.TieneProcesos(EstadoReady) {
		slog.Debug("Debug - (IntentarEnviarProcesoAExecute) - No hay procesos en READY")
		mutex_ProcesoPorEstado[EstadoReady].Unlock()
		return
	}

	hay_que_chequear_desalojo := k.PlaniCortoPlazo()

	//Tomamos el primer PCB "NO" reservado tras la planificacion
	var pcb *PCB
	listaReady := k.ProcesoPorEstado[EstadoReady]
	for i := range listaReady {
		candidato := k.ElementoNSinSacar(EstadoReady, i)
		if !EstaReservado(candidato) {
			pcb = candidato
			break
		}
	}
	if pcb == nil {
		slog.Debug("Debug - (IntentarEnviarProcesoAExecute) No hay procesos disponibles en READY todos reservados")
		mutex_ProcesoPorEstado[EstadoReady].Unlock()
		return
	}

	MarcarProcesoReservado(pcb, "ESPERANDO CPU")

	pid := pcb.Pid
	pc := pcb.Pc
	mutex_ProcesoPorEstado[EstadoReady].Unlock()

	// intentamos asignarle cpu
	mutex_CPUsConectadas.Lock()
	defer mutex_CPUsConectadas.Unlock()
	cpu_seleccionada := k.ObtenerCPULibre()

	if cpu_seleccionada == nil { //no hay cpu libre
		if hay_que_chequear_desalojo {
			slog.Debug("Debug - (IntentarEnviarProcesoAExecute) - No hay CPU libre, intentando desalojo por SRT", "pid", pid)
			if !k.IntentarDesalojoSRT(pid) {
				slog.Debug("Debug - (IntentarEnviarProcesoAExecute) - No hay que desalojar", "pid", pid)
				MarcarProcesoReservado(pcb, "NO")
			}

		} else {
			slog.Debug("Debug - (IntentarEnviarProcesoAExecute) - No hay CPU libre y no se requiere desalojo", "pid", pid)
			MarcarProcesoReservado(pcb, "NO")
		}

		return
	}
	//voy a liberar la cpu, la reservo
	reservarCPU(cpu_seleccionada, pid)

	idCPU := cpu_seleccionada.ID
	url := cpu_seleccionada.Url

	handleDispatch(pid, pc, url)

	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	defer mutex_ProcesoPorEstado[EstadoExecute].Unlock()

	mutex_ProcesoPorEstado[EstadoReady].Lock()
	defer mutex_ProcesoPorEstado[EstadoReady].Unlock()

	//Checkeo por las dudas que sigue en la misma posicion de memoria
	procVerificadoAExecute := k.QuitarYObtenerPCB(EstadoReady, pid, false)

	if procVerificadoAExecute == nil {
		slog.Warn("Cuidadito - (IntentarEnviarProcesoAExecute) - El proceso no esta en la lista Ready", "pid", pid)

		return
	}
	actualizarCPU(cpu_seleccionada, pid, pc, false)

	k.AgregarAEstado(EstadoExecute, procVerificadoAExecute, false)

	MarcarProcesoReservado(procVerificadoAExecute, "NO")

	slog.Debug("Debug - (IntentarEnviarProcesoAExecute)- Proceso enviado a EXECUTE", "pid", pid, "cpu_id", idCPU)

}

func (k *Kernel) PlaniCortoPlazo() bool {

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
