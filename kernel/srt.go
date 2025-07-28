package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

// Ya esta previamente tomado el mutex CPUsConectadas
// ya esta previamente tomado el mutex de EXECUTE
func (k *Kernel) ChequearDesalojo(proceso_suplente *PCB) *CPU {
	//1er check - Buscar CPU candidata
	//Auxs
	var nuevo_estimado_actual_en_exec float64
	var pcb_aux *PCB

	//cpu candidata a DESALOJAR
	var cpu_candidata *CPU = nil
	//pcb candidato a DESALOJAR
	var pcb_a_desalojar *PCB = nil

	for _, cpu := range k.CPUsConectadas {
		pcb_aux = k.BuscarPorPidSinLock(EstadoExecute, cpu.Pid)

		// calculamos
		nuevo_estimado_actual_en_exec = pcb_aux.SJF.Estimado_actual - float64(duracionEnEstado(pcb_aux)) //a chequear

		if proceso_suplente.SJF.Estimado_actual < nuevo_estimado_actual_en_exec {
			slog.Debug("Debug - (ChequearDesalojo) - Consegui un posible candidato",
				"pcb_candidato", pcb_aux.Pid, "cpu_candidata", cpu.ID)
			cpu_candidata = cpu
			pcb_a_desalojar = pcb_aux
		}
	}
	//-------Si no encontre
	if cpu_candidata == nil && pcb_a_desalojar == nil {
		slog.Debug("Debug - (ChequearDesalojo) - Ninguna cpu cumplica con condicion de estimacion, no hubo desalojo")
		return nil
	}

	//-------Si encontre
	return cpu_candidata
}

func (k *Kernel) RealizarDesalojo(cpu_a_detonar *CPU, pid_a_entrar int) {
	pc_actualizado, pid_aux_exit := k.EnviarInterrupt(cpu_a_detonar)

	// Actualizamos pc en la cpu que estaba ejecutando
	desaloje := k.MudancasNoElenco(cpu_a_detonar, pid_aux_exit, pid_a_entrar, pc_actualizado)
	if !desaloje {
		slog.Error("Error - (RealizarDesalojo) - Por alguna razon, no se pudo desalojar",
			"pid que tenia que desalojar", pid_a_entrar)
		return
	}

	slog.Debug("Debug - (RealizarDesalojo) - Pude desalojar correctamente, jupiiiii",
		"pid que entro", pid_a_entrar, "cpu detonada", cpu_a_detonar.ID)

}

// -----------------------------Relatorios do Clube Atletico Velez Sarsfield------------------------------
func (k *Kernel) MudancasNoElenco(cpu_ejecutando *CPU, pid_aux_exit, pid_suplente, pc_titular int) bool {

	proceso_suplente := k.BuscarPorPidSinLock(EstadoReady, pid_suplente)
	proceso_titular := k.BuscarPorPidSinLock(EstadoExecute, cpu_ejecutando.Pid)

	fue_expulsado := false

	if proceso_titular == nil {
		slog.Debug("Debug - (MudancasNoElenco) el procesoEjecutando no esta en la lista EXECUTE, capaaaaaz fue expulsado, procedo a buscarlo")

		mutex_expulsadosPorRoja.Lock()
		encontre := k.buscarEnExpulsados(pid_aux_exit)
		mutex_expulsadosPorRoja.Unlock()

		if !encontre {
			slog.Error("Error - (MudancasNoElenco) - El procesoEjecutando no esta ni en la lista EXECUTE ni fue expulsado")
			return false
		}

		fue_expulsado = true
	}

	// SALE KAROL (DO RIO DE JANEIRO), actualizamos datos del pcb titular
	if !fue_expulsado {
		tiempo_en_cancha := duracionEnEstado(proceso_titular)
		k.actualizarEstimacionSJF(proceso_titular, tiempo_en_cancha)
		proceso_titular.Pc = pc_titular
		k.MoverDeEstadoPorPid(EstadoExecute, EstadoReady, proceso_titular.Pid, false)
	}

	utils.LoggerConFormato("## (%d) - Desalojado por algoritmo SJF/SRT", pid_aux_exit)

	// Actualizamos la cpu con el proceso nuevo
	cpu_ejecutando.Pc = proceso_suplente.Pc
	cpu_ejecutando.Pid = proceso_suplente.Pid

	// Enviamos el nuevo proceso a cpu
	// Debutante
	// ENTRA AQUINO (Mi primo, que si aprobo el tp )
	handleDispatch(cpu_ejecutando.Pid, cpu_ejecutando.Pc, cpu_ejecutando.Url)

	proceso_enviado_a_exec := k.QuitarYObtenerPCB(EstadoReady, proceso_suplente.Pid, false)

	if proceso_enviado_a_exec == nil {
		slog.Error("Error - (MudancasNoElenco) - El procesoQuiereEjecutar no esta en la lista READY")
		return false
	}

	k.AgregarAEstado(EstadoExecute, proceso_enviado_a_exec, false)

	utils.LoggerConFormato("(%d) Pasa del estado <%s> al estado <%s>",
		proceso_suplente.Pid,
		estados_proceso[EstadoReady],
		estados_proceso[EstadoExecute],
	)

	if !fue_expulsado {
		fmt.Printf("CAMBIO: Sale %d (est. %.2f), entra %d (est. %.2f)\n", // Leer con voz de gangoso
			proceso_titular.Pid, proceso_titular.SJF.Estimado_actual,
			proceso_suplente.Pid, proceso_suplente.SJF.Estimado_actual,
		)
		return true
	}

	fmt.Printf("CAMBIO: Sale %d (como fue expulsado, no importa su estimacion, fue un EXIT), entra %d (est. %.2f)\n", // Leer con voz de gangoso
		pid_aux_exit,
		proceso_suplente.Pid, proceso_suplente.SJF.Estimado_actual,
	)

	return true
}

func (k *Kernel) EnviarInterrupt(cpu_a_detonar *CPU) (int, int) {
	respuesta, _ := utils.FormatearUrlYEnviar(cpu_a_detonar.Url, "/interrupt", true, "")
	aux := strings.Split(respuesta, " ")

	pc_actualizado, _ := strconv.Atoi(aux[0])
	pid_aux_exit, _ := strconv.Atoi(aux[1])

	return pc_actualizado, pid_aux_exit
}

func (k *Kernel) buscarEnExpulsados(pid_expulsado_a_buscar int) bool {

	for i := 0; i < len(k.ExpulsadosPorRoja); i++ {
		if k.ExpulsadosPorRoja[i] == pid_expulsado_a_buscar {
			k.ExpulsadosPorRoja = append(k.ExpulsadosPorRoja[:i], k.ExpulsadosPorRoja[i+1:]...)
			return true
		}
	}

	return false
}
