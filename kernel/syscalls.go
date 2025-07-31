package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	utils "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (k *Kernel) ActualizarPCEnExec(pid, pc int) {

	mutex_ProcesoPorEstado[EstadoExecute].Lock()

	procesoEjecutando := k.BuscarPorPidSinLock(EstadoExecute, pid)

	if procesoEjecutando == nil {
		slog.Error("Error -(ActualizarPCEnExec) - No se encontró el proceso en ejecucion para esa CPU",
			"pid", pid,
		)
		mutex_ProcesoPorEstado[EstadoExecute].Unlock()
		return
	}

	procesoEjecutando.Pc = pc

	slog.Debug("Debug - (ActualizarPCEnExec) - PC actualizado",
		"pid", pid,
		"pc", pc,
	)
	mutex_ProcesoPorEstado[EstadoExecute].Unlock()
}

func (k *Kernel) llegaSyscallCPU(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (llegoSyscallCPU) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llegó syscall", "mensaje", mensaje)

	debeContinuar, err := k.GestionarSyscalls(mensaje)
	if err != nil {
		slog.Error("Error al gestionar syscall", "detalle", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//k.ImprimirPCBsDeEstado(EstadoReady)

	if debeContinuar {
		w.Write([]byte("SEGUI"))
	} else {

		//slog.Debug("intentando enviar proceso a execute", "mensaje", mensaje)

		w.Write([]byte("REPLANIFICAR"))
	}

}

func (k *Kernel) GestionarSyscalls(respuesta string) (bool, error) {

	syscall := strings.Split(respuesta, " ")

	if len(syscall) < 3 {
		return false, fmt.Errorf("error - syscall incompleta: %v", respuesta)
	}

	idCPU, err := strconv.Atoi(syscall[0])

	if err != nil {
		return false, fmt.Errorf("error - ID de CPU inválido: %s", syscall[0])
	}

	pid, err := strconv.Atoi(syscall[1])

	if err != nil {
		return false, fmt.Errorf("error - PID de CPU inválido: %s", syscall[1])
	}

	pc, err := strconv.Atoi(syscall[2])

	if err != nil {
		return false, fmt.Errorf("error - PC inválido: %s", syscall[2])
	}

	cod_op := syscall[3]

	utils.LoggerConFormato("## (%d) - Solicitó syscall: %s", pid, cod_op)

	switch cod_op {
	case "IO":
		// IDCPU PID PC IO DISCO 9000
		if len(syscall) < 6 {
			return false, fmt.Errorf("error - syscall IO mal formada")
		}

		nombre := syscall[4]
		tiempo, err := strconv.Atoi(syscall[5])
		if err != nil {
			return false, fmt.Errorf("error - syscall IO - tiempo inválido: %s", syscall[5])
		}

		k.ActualizarPCEnExec(pid, pc)

		k.GestionarIO(nombre, pid, tiempo, idCPU)

		return false, nil // la CPU debe replanificar

	case "INIT_PROC":
		// IDCPU PID PC INIT_PROC proceso1 256
		if len(syscall) < 6 {
			return false, fmt.Errorf("error - syscall INIT_PROC mal formada")
		}

		nombre_arch := syscall[4]
		tamanio, err := strconv.Atoi(syscall[5])

		if err != nil {
			return false, fmt.Errorf("error - syscall INIT_PROC - tamaño inválido: %s", syscall[5])
		}

		k.GestionarINIT_PROC(nombre_arch, tamanio)

		return true, nil // la CPU debe seguir con el proceso

	case "DUMP_MEMORY":
		// IDCPU PID PC DUMP_MEMORY
		k.ActualizarPCEnExec(pid, pc)
		k.GestionarDUMP_MEMORY(pid, idCPU)

		return false, nil // la CPU debe replanificar

	case "EXIT":
		// IDCPU PID PC EXIT
		k.GestionarEXIT(pid, idCPU)
		//k.liberarCPU(idCPU)

		return false, nil // la CPU debe replanificar

	default:
		return false, fmt.Errorf("error - syscall no reconocida %s", cod_op)
	}

}

func (k *Kernel) GestionarIO(nombreIO string, pid, duracion, idCPU int) {

	//mutex IOs
	mutex_DispositivosIO.Lock()
	defer mutex_DispositivosIO.Unlock()

	iosMismoNombre, existeIO := k.DispositivosIO[nombreIO]

	if !existeIO {
		slog.Error("Error - (GestionarIO) - No existe el dispositivo",
			"nombre_io", nombreIO)
		k.GestionarEXIT(pid, idCPU)
		return
	}

	mutex_ProcesoPorEstado[EstadoBlock].Lock()
	mutex_ProcesoPorEstado[EstadoExecute].Lock()

	pcb := k.BuscarPorPidSinLock(EstadoExecute, pid)
	if pcb == nil {
		slog.Error("Error - (GestionarIO) - No se encontró el PCB del proceso en EXECUTE",
			"pid", pid)
		mutex_ProcesoPorEstado[EstadoExecute].Unlock()
		mutex_ProcesoPorEstado[EstadoBlock].Unlock()
		return
	}

	k.MoverDeEstadoPorPid(EstadoExecute, EstadoBlock, pid, false)
	k.actualizarEstimacionSJF(pcb, duracionEnEstado(pcb))

	mutex_ProcesoPorEstado[EstadoExecute].Unlock()
	mutex_ProcesoPorEstado[EstadoBlock].Unlock()

	go k.temporizadorSuspension(pid)

	IO_seleccionada := k.buscarIOLibre(nombreIO)

	if IO_seleccionada == nil { //no hay io libre
		nuevo_proc_esperando := &ProcesoEsperandoIO{
			Pid:      pid,
			TiempoIO: duracion,
		}
		iosMismoNombre.ColaEspera = append(iosMismoNombre.ColaEspera, nuevo_proc_esperando)
		utils.LoggerConFormato("## (%d) - Encolado en espera de IO: %s", pid, nombreIO)
		return
	}
	// si hay io libre
	IO_seleccionada.PidOcupante = pid
	IO_seleccionada.Libre = false

	utils.LoggerConFormato("## (%d) - Bloqueado por IO: %s", pid, nombreIO)
	enviarProcesoAIO(IO_seleccionada, duracion)

}

func (k *Kernel) GestionarINIT_PROC(nombre_arch string, tamanio int) {

	nuevo_pcb := k.IniciarProceso(tamanio, nombre_arch)
	pid := nuevo_pcb.Pid
	k.AgregarAEstado(EstadoNew, nuevo_pcb, true)
	utils.LoggerConFormato(" (%d) Se crea el proceso - Estado: NEW", pid)

	unElemento, _ := k.UnicoEnNewYNadaEnSuspReady() //intenta con esta funcion (le da un cachetazo, lo tantea haber si puede entrar)

	if !unElemento { //sino, intenta con esta funcion kjahsdkjashd
		k.IntentarEnviarProcesoAReady(EstadoNew, pid)
	}

	//k.IntentarEnviarProcesosAReady()
}

func (k *Kernel) GestionarDUMP_MEMORY(pid int, idCpu int) {

	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	pcb := k.BuscarPorPidSinLock(EstadoExecute, pid)
	if !k.MoverDeEstadoPorPid(EstadoExecute, EstadoBlock, pid, true) {
		slog.Error("GestionarDUMP_MEMORY: no se pudo mover a BLOCK",
			"pid", pid)
		return
	}
	k.actualizarEstimacionSJF(pcb, duracionEnEstado(pcb))
	mutex_ProcesoPorEstado[EstadoExecute].Unlock()
	go func(pid int) {

		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic en GestionarDUMP_MEMORY goroutine", "pid", pid, "panic", r)
			}
		}()

		if !EnviarMemoryDump(pid) {
			k.GestionarEXIT(pid, idCpu)
			return
		}

		if !k.MoverDeEstadoPorPid(EstadoBlock, EstadoReady, pid, true) {
			slog.Error("Error - (GestionarDUMP_MEMORY) - No se encontó el proceso",
				"pid", pid)
			return
		}

		utils.LoggerConFormato("## (%d) - DumpMemory finalizado correctamente", pid)
		slog.Debug("Debug - (GestionarDUMP_MEMORY) - A esta altura, se supone que envie a READY al proceso que hizo el Dump")
		puedo_ejecutar, pcb_candidato := k.SoyPrimeroEnREADYyNadaEnSuspREADY(pid)

		if puedo_ejecutar {
			mutex_ProcesoPorEstado[EstadoReady].Lock()
			k.IntentarEnviarProcesoAExecutePorPCB(pcb_candidato)
			mutex_ProcesoPorEstado[EstadoReady].Unlock()
		}

	}(pid)

}

func (k *Kernel) GestionarEXIT(pid int, idCPU int) {
	//saco de execute el proceso que esta ejecutando y lo obtengo
	mutex_ProcesoPorEstado[EstadoExecute].Lock()
	pcb := k.QuitarYObtenerPCB(EstadoExecute, pid, false)
	mutex_ProcesoPorEstado[EstadoExecute].Unlock()
	if pcb == nil {
		slog.Error("Error - (GestionarEXIT) - No se encontró el pid asociado a la cpu en Execute",
			"pid", pid,
			"cpu", idCPU,
		)

		return
	}

	mutex_expulsadosPorRoja.Lock()
	k.ExpulsadosPorRoja = append(k.ExpulsadosPorRoja, pid)
	mutex_expulsadosPorRoja.Unlock()

	go k.EliminarProceso(pcb, true)
}
