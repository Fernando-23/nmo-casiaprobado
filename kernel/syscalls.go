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

func (k *Kernel) liberarCPU(idCPU int) {
	mutex_CPUsConectadas.Lock()
	defer mutex_CPUsConectadas.Unlock()

	cpu, ok := k.CPUsConectadas[idCPU]
	if !ok {
		slog.Error("No se encontró CPU al liberar", "idCPU", idCPU)
		return
	}
	actualizarCPU(cpu, -1, 0, true)
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

	if debeContinuar {
		w.Write([]byte("SEGUI"))
	} else {
		k.IntentarEnviarProcesoAExecute()
		w.Write([]byte("REPLANIFICAR"))
	}

}

func (k *Kernel) GestionarSyscalls(respuesta string) (bool, error) {

	syscall := strings.Split(respuesta, " ")

	if len(syscall) < 3 {
		return false, fmt.Errorf("error - syscall incompleta: %v", respuesta)
	}

	idCPU, err := strconv.Atoi(syscall[IdCPU])

	if err != nil {
		return false, fmt.Errorf("error - ID de CPU inválido: %s", syscall[IdCPU])
	}

	pc, err := strconv.Atoi(syscall[PC])

	if err != nil {
		return false, fmt.Errorf("error - PC inválido: %s", syscall[PC])
	}

	mutex_CPUsConectadas.Lock()
	cpu_ejecutando, existe := k.CPUsConectadas[idCPU]
	if !existe {
		mutex_CPUsConectadas.Unlock()
		return false, fmt.Errorf("error - CPU no registrada: %d", idCPU)
	}
	pid := cpu_ejecutando.Pid
	mutex_CPUsConectadas.Unlock()

	cod_op := syscall[CodOp]

	utils.LoggerConFormato("## (%d) - Solicitó syscall: %s", pid, cod_op)
	switch cod_op {
	case "IO":
		// 2 20 IO AURICULARES 9000
		if len(syscall) < 5 {
			return false, fmt.Errorf("error - syscall IO mal formada")
		}

		nombre := syscall[3]
		tiempo, err := strconv.Atoi(syscall[4])
		if err != nil {
			return false, fmt.Errorf("error - syscall IO - tiempo inválido: %s", syscall[4])
		}
		k.GestionarIO(nombre, pid, pc, tiempo, idCPU)
		k.liberarCPU(idCPU)
		return false, nil // la CPU debe replanificar

	case "INIT_PROC":
		// 2 20 INIT_PROC proceso1 256
		if len(syscall) < 5 {
			return false, fmt.Errorf("error - syscall INIT_PROC mal formada")
		}

		nombre_arch := syscall[3]
		tamanio, err := strconv.Atoi(syscall[4])

		if err != nil {
			return false, fmt.Errorf("error - syscall IO - tamaño inválido: %s", syscall[4])
		}

		k.GestionarINIT_PROC(nombre_arch, tamanio, pc)

		return true, nil // la CPU debe seguir con el proceso

	case "DUMP_MEMORY":
		// 2 30 DUMP_MEMORY
		k.GestionarDUMP_MEMORY(pid, idCPU)
		k.liberarCPU(idCPU)

		return false, nil // la CPU debe replanificar

	case "EXIT":
		// 2 30 EXIT
		k.GestionarEXIT(pid, idCPU)
		k.liberarCPU(idCPU)

		return false, nil // la CPU debe replanificar

	default:
		return false, fmt.Errorf("error - syscall no reconocida %s", cod_op)
	}

}

func (k *Kernel) GestionarIO(nombreIO string, pid, pc, duracion, idCPU int) {

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

	pcb.Pc = pc

	k.MoverDeEstadoPorPid(EstadoExecute, EstadoBlock, pid, false)

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

func (k *Kernel) GestionarINIT_PROC(nombre_arch string, tamanio int, pc int) {

	nuevo_pcb := k.IniciarProceso(tamanio, nombre_arch)
	pid := nuevo_pcb.Pid
	k.AgregarAEstado(EstadoNew, nuevo_pcb, true)
	utils.LoggerConFormato(" (%d) Se crea el proceso - Estado: NEW", pid)

	unElemento, _ := k.UnicoEnNewYNadaEnSuspReady()

	if !unElemento {
		if !k.IntentarEnviarProcesoAReady(EstadoNew, pid) {
			return
		}
	}

	k.IntentarEnviarProcesoAExecute()
	// cpu_ejecutando.Pc = pc //Actualizar pc para cpu
}

func (k *Kernel) GestionarDUMP_MEMORY(pid int, idCpu int) {

	if !k.MoverDeEstadoPorPid(EstadoExecute, EstadoBlock, pid, true) {
		slog.Error("GestionarDUMP_MEMORY: no se pudo mover a BLOCK",
			"pid", pid)
		return
	}

	go func(pid int) {

		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic en GestionarDUMP_MEMORY goroutine", "pid", pid, "panic", r)
			}
		}()

		if !EnviarMemoryDump(pid) {
			k.GestionarEXIT(pid, idCpu)
		}

		if !k.MoverDeEstadoPorPid(EstadoBlock, EstadoReady, pid, true) {
			slog.Error("Error - (GestionarDUMP_MEMORY) - No se encontó el proceso",
				"pid", pid)
			return
		}
		utils.LoggerConFormato("## (%d) - DumpMemory finalizado correctamente", pid)
	}(pid)

}

func (k *Kernel) GestionarEXIT(pid int, idCPU int) {
	//saco de execute el proceso que esta ejecutando y lo obtengo
	pcb := k.QuitarYObtenerPCB(EstadoExecute, pid, true)
	if pcb == nil {
		slog.Error("Error - (GestionarEXIT) - No se encontró el pid asociado a la cpu en Execute",
			"pid", pid,
			"cpu", idCPU,
		)
		return
	}
	//envio solicitud para eliminar proceso
	go k.EliminarProceso(pcb, true)
}
