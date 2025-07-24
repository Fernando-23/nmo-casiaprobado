package main

import (
	"log/slog"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

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
		Reservado:   "NO",
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

func (k *Kernel) CalcularEstSJF_sinModifPCB(estimado_anterior, real_anterior float64) float64 {
	alpha := k.Configuracion.Alfa
	nuevo_estimado_actual := (alpha * real_anterior) + ((1 - alpha) * estimado_anterior)

	return nuevo_estimado_actual
}

func (k *Kernel) EliminarProceso(procesoAEliminar *PCB, liberaMemoria bool) {

	if err := k.solicitudEliminarProceso(procesoAEliminar.Pid); err != nil {
		slog.Error("Error - (EliminarProceso) - Solicitud a memoria fallida",
			"pid", procesoAEliminar.Pid,
			"error", err,
		)
		return
	}

	utils.LoggerConFormato("## (%d) - Finaliza el proceso", procesoAEliminar.Pid)

	utils.LoggerConFormato("## (%d) - Métricas de estado: ", procesoAEliminar.Pid)

	for estado := range cantEstados {
		utils.LoggerConFormato(
			"%s (%d) (%s),",
			estados_proceso[estado],
			procesoAEliminar.Me[estado],
			procesoAEliminar.Mt[estado].String(),
		)

	}

	if liberaMemoria {
		slog.Debug("Debug - (EliminarProceso) - Libere memoria, voy a proceder a hacer mis cositas")
		k.IntentarEnviarProcesosAReady()
		//k.IntentarEnviarProcesoAExecute()
	}
}

func ReservarSRT(pcb *PCB, reservado string) {
	pcb.Reservado = reservado
}

func EstaReservado(pcb *PCB) bool {
	return pcb.Reservado != "NO"
}

func (k *Kernel) esProcesoMasChico(pid int, estadoOrigen int) bool {
	procQuiereDetonar := k.BuscarPorPidSinLock(estadoOrigen, pid)
	procMasChico := k.PrimerElementoSinSacar(estadoOrigen)

	//Si es mas chico
	if procQuiereDetonar.Tamanio < procMasChico.Tamanio {
		return true
	}
	return false
}
