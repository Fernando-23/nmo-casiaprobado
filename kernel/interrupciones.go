package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (k *Kernel) llegaFinInterrupcion(w http.ResponseWriter, r *http.Request) {
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
