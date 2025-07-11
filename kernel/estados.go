package main

import (
	"log/slog"
	"slices"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (k *Kernel) InicializarMapaDeEstados() {
	k.ProcesoPorEstado = make(map[int][]*PCB)

	// Inicializamos todos los estados del map
	for i := 0; i < cantEstados; i++ {
		k.ProcesoPorEstado[i] = []*PCB{}
	}
}

func (k *Kernel) ImprimirPCBsDeEstado(estado int) {
	mutex_ProcesoPorEstado[estado].Lock()
	defer mutex_ProcesoPorEstado[estado].Unlock()

	listaPCB, ok := k.ProcesoPorEstado[estado]

	if !ok || len(listaPCB) == 0 {
		slog.Debug("No hay procesos en estado", "estado", estados_proceso[estado])
		return
	}

	for _, pcb := range listaPCB {
		if pcb != nil {
			slog.Debug("PCB",
				"pid", pcb.Pid,
				"estado", estados_proceso[estado],
				"pc", pcb.Pc)
		}
	}
}

func (k *Kernel) AgregarAEstado(estado int, pcb *PCB, hacerSincro bool) {

	if hacerSincro {
		mutex_ProcesoPorEstado[estado].Lock()
		defer mutex_ProcesoPorEstado[estado].Unlock()
	}

	actualizarMetricasEstado(pcb, estado)
	pcb.HoraIngresoAEstado = time.Now()

	k.ProcesoPorEstado[estado] = append(k.ProcesoPorEstado[estado], pcb)
}

func (k *Kernel) QuitarYObtenerPCB(estado int, pid int, hacerSincro bool) *PCB {
	if hacerSincro {
		mutex_ProcesoPorEstado[estado].Lock()
		defer mutex_ProcesoPorEstado[estado].Unlock()

	}

	procesos := k.ProcesoPorEstado[estado]
	for i, pcb := range procesos {
		if pcb.Pid == pid {

			// Actualizamos la metrica de tiempo de dicho estado
			actualizarMetricasTiempo(pcb, estado)

			// Sacar el proceso de la lista del estado
			k.ProcesoPorEstado[estado] = slices.Delete(procesos, i, i+1)
			return pcb
		}
	}
	return nil //no se encontro
}

func (k *Kernel) MoverDeEstadoPorPid(estadoActual, estadoNuevo int, pid int, hacerSincro bool) bool {
	// Buscar el puntero al PCB en el estado actual
	pcb := k.QuitarYObtenerPCB(estadoActual, pid, hacerSincro) //aca sincroniza

	if pcb == nil {
		slog.Warn("Cuidadito -(MoverDeEstadoPorPid) - No se encontró  proceso para mover de estado",
			"pid", pid,
			"estado_actual", estados_proceso[estadoActual])
		return false
	}

	// Agregar el puntero al PCB al nuevo estado
	k.AgregarAEstado(estadoNuevo, pcb, hacerSincro) //aca sincroniza

	utils.LoggerConFormato("## (%d) Pasa del estado %s al estado %s", pid, estados_proceso[estadoActual], estados_proceso[estadoNuevo])
	return true
}

func (k *Kernel) TieneProcesos(estado int) bool {

	return len(k.ProcesoPorEstado[estado]) > 0
}

func (k *Kernel) PrimerElementoSinSacar(estado int) *PCB {

	pcbs := k.ProcesoPorEstado[estado]

	if len(pcbs) == 0 {
		return nil
	}

	return pcbs[0]

}

func (k *Kernel) ElementoNSinSacar(estado int, n int) *PCB {

	pcbs := k.ProcesoPorEstado[estado]

	if len(pcbs) == 0 {
		return nil
	}

	return pcbs[n]

}

func (k *Kernel) BuscarPorPidSeguro(estado int, pid int) *PCB {
	mutex_ProcesoPorEstado[estado].Lock()
	defer mutex_ProcesoPorEstado[estado].Unlock()

	return k.BuscarPorPidSinLock(estado, pid)
}

func (k *Kernel) BuscarPorPidSinLock(estado int, pid int) *PCB {

	// Buscar el puntero al PCB en el estado actual
	procesos := k.ProcesoPorEstado[estado]
	var pcb *PCB
	for _, proceso := range procesos {
		if proceso.Pid == pid {
			pcb = proceso
			return pcb
		}
	}
	return nil
}

func duracionEnEstado(pPcb *PCB) time.Duration {
	return time.Since(pPcb.HoraIngresoAEstado)
}

func actualizarMetricasEstado(pPcb *PCB, posEstado int) {
	pPcb.Me[posEstado]++ //ver si puede quedar mas lindo

}

func actualizarMetricasTiempo(pPcb *PCB, posEstado int) {
	pPcb.Mt[posEstado] += duracionEnEstado(pPcb)
}
