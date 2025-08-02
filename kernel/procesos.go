package main

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"

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

func (k *Kernel) actualizarEstimacionSJF(pcb *PCB, tiempoEnExecute int64) {

	if pcb == nil || pcb.SJF == nil {
		slog.Debug("Debug - (actualizarEstimacionSJF) - PCB o SJF nil en actualizarEstimacionSJF")
		return
	}

	pcb.SJF.Real_anterior = float64(tiempoEnExecute)
	alpha := k.Configuracion.Alfa
	sjf := pcb.SJF
	aux_estimacion_actual := sjf.Estimado_actual

	sjf.Estimado_actual = (alpha * sjf.Real_anterior) + ((1 - alpha) * aux_estimacion_actual)

	sjf.Estimado_anterior = aux_estimacion_actual

	slog.Debug("Debug - (actualizarEstimacionSJF) - Actualizando SJF",
		"pid", pcb.Pid,
		"real", pcb.SJF.Real_anterior,
		"estimado_anterior", sjf.Estimado_anterior,
		"nuevo_estimado", sjf.Estimado_actual,
	)
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
			"%s (%d) (%d)ms,",
			estados_proceso[estado],
			procesoAEliminar.Me[estado],
			procesoAEliminar.Mt[estado],
		)

	}

	if liberaMemoria {
		slog.Debug("Debug - (EliminarProceso) - Libere memoria, voy a entrar a IntentarEnviarProcesosAReady()")
		k.IntentarEnviarProcesosAReady()
	}
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

func (k *Kernel) RecibirAvisoLiberarCPU(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (RecibirAvisoLiberarCPU) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensajeCPU := string(body_Bytes)

	id_cpu, _ := strconv.Atoi(mensajeCPU) // IDCPU

	exito := k.RenovacionDeCPU(id_cpu)

	if !exito {
		slog.Error("Error - (RecibirAvisoLiberarCPU) - Error en RenovacionDeCPU", "id recibido", id_cpu)
		return
	}

	slog.Debug("Debug - (RecibirAvisoLiberarCPU) - Libere correctamente la CPU",
		"id cpu", id_cpu)

}
