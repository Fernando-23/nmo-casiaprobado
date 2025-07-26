package main

import (
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

	if !k.MudancasNoElenco(cpu, procesoQuiereEjecutar.Pid, pcActualizado) {
		slog.Error("Error - (antenderFinInterrupcion) - NO se pudo realizar los cambiosEnElPLantel")
		return
	}

	actualizarCPU(cpu, cpu.ADesalojarPor, pcActualizado, false)
}

//-------------------------------------------EL BASURERO DE KERNEL-------------------------------------------

// // -----------Informa el Club Atletico Velez Sarsfield------------
// func (k *Kernel) CambiosEnElPlantel(cpuPosicion *CPU, procesoTitular *PCB, procesoSuplente *PCB) bool {
// 	// Debutante
// 	// CALIENTA KAROL

// 	// Actualizamos pc en el pcb del proceso que estaba ejecutando
// 	procesoTitular.Pc = cpuPosicion.Pc

// 	// Ahora si desalojamos al pcb correspondiente
// 	tiempo_en_cancha := duracionEnEstado(procesoTitular)
// 	k.actualizarEstimacionSJF(procesoTitular, tiempo_en_cancha)

// 	utils.LoggerConFormato("## (%d) - Desalojado por algoritmo SJF/SRT", cpuPosicion.Pid)

// 	if !k.MoverDeEstadoPorPid(EstadoExecute, EstadoReady, procesoTitular.Pid, false) {
// 		slog.Error("Error - (CambiosEnElPlantel) el procesoEjecutando no esta en la lista EXECUTE")
// 		return false
// 	}

// 	// Actualizar la cpu con el proceso nuevo
// 	cpuPosicion.Pc = procesoSuplente.Pc
// 	cpuPosicion.Pid = procesoSuplente.Pid

// 	// Enviar nuevo proceso a cpu
// 	handleDispatch(cpuPosicion.Pid, cpuPosicion.Pc, cpuPosicion.Url)
// 	// ENTRA AQUINO (Mi primo, que si aprobo el tp)

// 	procVerificadoAExecute := k.QuitarYObtenerPCB(EstadoReady, procesoSuplente.Pid, false)

// 	if procVerificadoAExecute == nil {
// 		slog.Error("Error - (CambiosEnElPlantel) el procesoQuiereEjecutar no esta en la lista READY")
// 		return false
// 	}

// 	ReservarSRT(procVerificadoAExecute, "NO")

// 	k.AgregarAEstado(EstadoExecute, procVerificadoAExecute, false)

// 	utils.LoggerConFormato("(%d) Pasa del estado <%s> al estado <%s>",
// 		procesoSuplente.Pid,
// 		estados_proceso[EstadoReady],
// 		estados_proceso[EstadoExecute],
// 	)

// 	fmt.Printf("CAMBIO: Sale %d (est. %.2f), entra %d (est. %.2f)\n", // Leer con voz de gangoso
// 		procesoTitular.Pid, procesoTitular.SJF.Estimado_actual,
// 		procesoSuplente.Pid, procesoSuplente.SJF.Estimado_actual,
// 	)
// 	return true
// }

// func EnviarInterrupt(cpu *CPU) { // yo te hablo por la puerta interrupt y me desocupo
// 	fullURL := fmt.Sprintf("%s/interrupt", cpu.Url)
// 	utils.EnviarStringSinEsperar("POST", fullURL, "")
// }

// func (k *Kernel) IntentarDesalojoSRT(pidQuiereDesalojar int) bool {

// 	slog.Debug("(IntentarDesalojoSRT) - Llegue hasta esta funcion", "pid", pidQuiereDesalojar)

// 	mutex_ProcesoPorEstado[EstadoExecute].Lock()
// 	defer mutex_ProcesoPorEstado[EstadoExecute].Unlock()

// 	mutex_ProcesoPorEstado[EstadoReady].Lock()
// 	defer mutex_ProcesoPorEstado[EstadoReady].Unlock()

// 	if !k.TieneProcesos(EstadoReady) {
// 		slog.Error("Error - (IntentarDEsalojoSRT) - no hay proceso en READY",
// 			"pid_quiere_desalojar", pidQuiereDesalojar,
// 		)
// 		return false //no hay procesos en READY, no tiene sentido desalojar
// 	}

// 	procesoCandidato := k.BuscarPorPidSinLock(EstadoReady, pidQuiereDesalojar)

// 	if procesoCandidato == nil {
// 		slog.Error("Error - (IntentarDesalojoSRT) - Proceso ya no está en READY", "pid", pidQuiereDesalojar)
// 		return false
// 	}
// 	estimacionReady := procesoCandidato.SJF.Estimado_actual

// 	var estimacionMaxRestante float64 = -1
// 	var cpuElegida *CPU

// 	for _, cpu := range k.CPUsConectadas {
// 		if cpu.ADesalojarPor != -1 { //si esta a la espera de ser desalojada
// 			continue
// 		}
// 		procesoEjecutando := k.BuscarPorPidSinLock(EstadoExecute, cpu.Pid)

// 		if procesoEjecutando == nil {
// 			slog.Error("Error - (IntentarDesalojoSRT) - El proceso no esta en la lista execute, incosistencia interna")
// 			return false
// 		}

// 		tiempoEjecutando := duracionEnEstado(procesoEjecutando)
// 		estimacionRestante := procesoEjecutando.SJF.Estimado_actual - float64(tiempoEjecutando.Milliseconds())

// 		if estimacionRestante < 0 { //por si ejecuto mas de lo que espero
// 			estimacionRestante = 0
// 		}

// 		if estimacionRestante > estimacionReady && estimacionRestante > estimacionMaxRestante { //
// 			estimacionMaxRestante = estimacionRestante
// 			cpuElegida = cpu
// 		}
// 	}

// 	if cpuElegida != nil {
// 		slog.Debug("Petición de desalojo por planificación SRT",
// 			"pid_desalojado", cpuElegida.Pid,
// 			"cpu_id", cpuElegida.ID,
// 			"estimacion_restante_desalojado", estimacionMaxRestante,
// 			"pid_candidato", procesoCandidato.Pid,
// 			"estimacion_restante_candidato", estimacionReady,
// 		)

// 		reservarCPU(cpuElegida, pidQuiereDesalojar)
// 		EnviarInterrupt(cpuElegida)

// 		return true
// 	}
// 	return false

// }

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

// Santiago Calizaya
