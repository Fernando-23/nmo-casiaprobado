package main

import (
	"fmt"
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
