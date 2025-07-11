package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (k *Kernel) llegaFinInterrupcion(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (llegoFinInterrupcion) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensajeCPU := string(body_Bytes)

	slog.Debug("Llego fin interrupcion", "mensaje", mensajeCPU)

	idCPU, pidDesalojado, pcActualizado, err := decodificarMensajeFinInterrupcion(mensajeCPU)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Ya validado, respondemos OK
	w.WriteHeader(http.StatusOK)

	go k.AtenderFinInterrupcion(idCPU, pidDesalojado, pcActualizado)

	utils.LoggerConFormato("Fin de AtenderFinInterrupcion para CPU %d", idCPU)

}

func (k *Kernel) AtenderFinInterrupcion(idCPU, pidDesalojado, pcActualizado int) {
	mutex_CPUsConectadas.Lock()
	defer mutex_CPUsConectadas.Unlock()
	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	defer mutex_ProcesoPorEstado[EstadoExecute].Unlock()
	mutex_ProcesoPorEstado[EstadoReady].Lock()
	defer mutex_ProcesoPorEstado[EstadoReady].Unlock()

	cpu := k.BuscarCPUPorID(idCPU)

	if cpu == nil {
		slog.Error("Error - (antenderFinInterrupcion) - NO se encontró la CPU",
			"id_cpu", idCPU,
		)
		return
	}

	if cpu.Pid != pidDesalojado {
		slog.Error("Error - (antenderFinInterrupcion) - NO coincide el pid recibido con el registrado",
			"pid_recibido", pidDesalojado,
			"pid_registrado", cpu.Pid,
		)
	}

	procesoEjecutando := k.BuscarPorPidSinLock(EstadoExecute, pidDesalojado)

	if procesoEjecutando == nil {
		slog.Error("Error - (antenderFinInterrupcion) - NO se encontró el proceso en Execute",
			"pid", cpu.ADesalojarPor,
		)
		return
	}
	procesoQuiereEjecutar := k.BuscarPorPidSinLock(EstadoReady, cpu.ADesalojarPor)

	if procesoQuiereEjecutar == nil {
		slog.Error("Error - (antenderFinInterrupcion) - NO se encontró el proceso a ejecutar",
			"pid", cpu.ADesalojarPor,
		)
		return
	}

	if !k.CambiosEnElPlantel(cpu, procesoEjecutando, procesoQuiereEjecutar) {
		slog.Error("Error - (antenderFinInterrupcion) - NO se pudo realizar los cambiosEnElPLantel")
		return
	}

	actualizarCPU(cpu, cpu.ADesalojarPor, pcActualizado, false)
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
		slog.Error("Error - (CambiosEnElPlantel) el procesoEjecutando no esta en la lista EXECUTE")
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
		slog.Error("Error - (CambiosEnElPlantel) el procesoQuiereEjecutar no esta en la lista READY")
		return false
	}

	MarcarProcesoReservado(procVerificadoAExecute, "NO")

	k.AgregarAEstado(EstadoExecute, procVerificadoAExecute, false)

	utils.LoggerConFormato("(%d) Pasa del estado <%s> al estado <%s>",
		procesoSuplente.Pid,
		estados_proceso[EstadoReady],
		estados_proceso[EstadoExecute],
	)

	fmt.Printf("CAMBIO: Sale %d (est. %.2f), entra %d (est. %.2f)\n", // Leer con voz de gangoso
		procesoTitular.Pid, procesoTitular.SJF.Estimado_actual,
		procesoSuplente.Pid, procesoSuplente.SJF.Estimado_actual,
	)
	return true
}

func EnviarInterrupt(cpu *CPU) { // yo te hablo por la puerta interrupt y me desocupo
	fullURL := fmt.Sprintf("%s/interrupt", cpu.Url)
	utils.EnviarStringSinEsperar("POST", fullURL, "")
}

func (k *Kernel) IntentarDesalojoSRT(pidQuiereDesalojar int) bool {

	slog.Debug("(IntentarDesalojoSRT) - Llegue hasta esta funcion", "pid", pidQuiereDesalojar)

	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	defer mutex_ProcesoPorEstado[EstadoExecute].Unlock()

	mutex_ProcesoPorEstado[EstadoReady].Lock()
	defer mutex_ProcesoPorEstado[EstadoReady].Unlock()

	if !k.TieneProcesos(EstadoReady) {
		slog.Error("Error - (IntentarDEsalojoSRT) - no hay proceso en READY",
			"pid_quiere_desalojar", pidQuiereDesalojar,
		)
		return false //no hay procesos en READY, no tiene sentido desalojar
	}

	procesoCandidato := k.BuscarPorPidSinLock(EstadoReady, pidQuiereDesalojar)

	if procesoCandidato == nil {
		slog.Error("Error - (IntentarDesalojoSRT) - Proceso ya no está en READY", "pid", pidQuiereDesalojar)
		return false
	}
	estimacionReady := procesoCandidato.SJF.Estimado_actual

	var estimacionMaxRestante float64 = -1
	var cpuElegida *CPU

	for _, cpu := range k.CPUsConectadas {
		if cpu.ADesalojarPor != -1 { //si esta a la espera de ser desalojada
			continue
		}
		procesoEjecutando := k.BuscarPorPidSinLock(EstadoExecute, cpu.Pid)

		if procesoEjecutando == nil {
			slog.Error("Error - (IntentarDesalojoSRT) - El proceso no esta en la lista execute, incosistencia interna")
			return false
		}

		tiempoEjecutando := duracionEnEstado(procesoEjecutando)
		estimacionRestante := procesoEjecutando.SJF.Estimado_actual - float64(tiempoEjecutando.Milliseconds())

		if estimacionRestante < 0 { //por si ejecuto mas de lo que espero
			estimacionRestante = 0
		}

		if estimacionRestante > estimacionReady && estimacionRestante > estimacionMaxRestante { //
			estimacionMaxRestante = estimacionRestante
			cpuElegida = cpu
		}
	}

	if cpuElegida != nil {
		slog.Debug("Petición de desalojo por planificación SRT",
			"pid_desalojado", cpuElegida.Pid,
			"cpu_id", cpuElegida.ID,
			"estimacion_restante_desalojado", estimacionMaxRestante,
			"pid_candidato", procesoCandidato.Pid,
			"estimacion_restante_candidato", estimacionReady,
		)

		reservarCPU(cpuElegida, pidQuiereDesalojar)
		EnviarInterrupt(cpuElegida)

		return true
	}
	return false

}
