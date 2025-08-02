package main

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

// Ya esta previamente tomado el mutex CPUsConectadas
// ya esta previamente tomado el mutex de EXECUTE

// consiste en buscar CPU candidata
func (k *Kernel) ChequearDesalojo(proceso_suplente *PCB) *CPU {

	//Auxs
	var nuevo_estimado_actual_en_exec float64
	var pcb_aux *PCB

	//cpu candidata a DESALOJAR
	var cpu_candidata *CPU = nil
	//pcb candidato a DESALOJAR
	var pcb_a_desalojar *PCB = nil

	for _, cpu := range k.CPUsConectadas {
		//si la cpu ya fue candidata y esta en proceso de desalojo, la desestimo
		slog.Debug("Debug - (ChequearDesalojo) ", "id_cpu", cpu.ID, "estaSiendoDesalojada", cpu.EstaSiendoDesalojada)
		if cpu.EstaSiendoDesalojada {
			continue
		}

		pcb_aux = k.BuscarPorPidSinLock(EstadoExecute, cpu.Pid)

		// calculamos
		nuevo_estimado_actual_en_exec = float64(pcb_aux.SJF.Real_anterior) - float64(duracionEnEstado(pcb_aux)) //a chequear

		slog.Debug("Debug - (ChequearDesalojo) ",
			"estimacion actual proceso suplente", proceso_suplente.SJF.Estimado_actual, "nuevo estimado actual del titular", nuevo_estimado_actual_en_exec)

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

func (k *Kernel) RealizarDesalojo(cpu_a_detonar *CPU, pid_a_entrar int) bool {
	k.EnviarInterrupt(cpu_a_detonar)

	//este lo saco porque lo voy a mover apenas llegue el fin de la interrupcion
	proceso_a_entrar := k.QuitarYObtenerPCB(EstadoReady, pid_a_entrar, false)
	if proceso_a_entrar == nil {
		slog.Error("Error - (RealizarDesalojo) - Por alguna razon no encontre al proceso titular en READY")
		return false
	}
	//al que va a ser desalojado NO PUEDO MOVERLO
	//no se si va a moverse por alguna syscall bloqueante
	proceso_desalojado := k.BuscarPorPidSinLock(EstadoExecute, cpu_a_detonar.Pid)
	if proceso_desalojado == nil {
		slog.Error("Error - (RealizarDesalojo) - Por alguna razon no encontre al proceso a desalojar en EXECUTE")
		return false
	}

	//me guardo esos PCBs para cuando llegue el fin de interrupcion y ahi gestiono todo
	//me lavo las manos en pocas palabras
	nuevo_intermedio := &InstanciaEsperandoDesalojo{
		id_cpu:             cpu_a_detonar.ID,
		proceso_titular:    proceso_a_entrar,
		proceso_desalojado: proceso_desalojado,
	}

	//MUY IMPORTANTE, asi nadie mas chequea con esta cpu hasta que se complete el desalojo
	cpu_a_detonar.EstaSiendoDesalojada = true

	mutex_esperandoDesalojo.Lock()
	//agrego esta instancia momentanea a mi lista de desalojados
	k.EsperandoDesalojo = append(k.EsperandoDesalojo, nuevo_intermedio)
	mutex_esperandoDesalojo.Unlock()

	slog.Debug("Debug - (RealizarDesalojo) - Envie interrupcion y me guarde los pcbs en un intermedio",
		"cpu_marcada", cpu_a_detonar.ID, "proceso_titular", pid_a_entrar, "proceso_a_salir", proceso_desalojado.Pid)
	return true

}

func (k *Kernel) EnviarInterrupt(cpu_a_detonar *CPU) {
	utils.FormatearUrlYEnviar(cpu_a_detonar.Url, "/interrupt", true, "cortala pipo")
}

// LA FUNCION QUE SIGUE EL LEGADO DE CAMBIOS EN EL PLANTEL
// ---------------- Informa el Club Atletico Velez Sarfield ----------------     leer con voz de gangoso
func (k *Kernel) LlegaFinInterrupt(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (LlegaFinInterrupt) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	mensaje := string(body_Bytes)
	aux := strings.Split(mensaje, " ")

	id_cpu, _ := strconv.Atoi(aux[0])
	nuevo_pc_proc_desalojado, _ := strconv.Atoi(aux[1])
	tipo_instruccion := aux[2]

	//recorro para encontrar la instancia de interrupcion
	mutex_esperandoDesalojo.Lock()
	for i := 0; i < len(k.EsperandoDesalojo); i++ {

		if k.EsperandoDesalojo[i].id_cpu == id_cpu {

			//actualizo proceso desalojado SI ES NECESARIO
			if tipo_instruccion == "REQUIERO_DESALOJO" {
				// ULTIMO PARTIDO DEL GENIO DEL FUTBOL ARGENTINO
				// SALE LIAM

				slog.Debug("Debug - (LlegaFinInterrupt) - El proceso desalojado no es una syscall bloqueante, deberia seguir en EXECUTE. Voy a moverlo a READY")
				if !k.GestionarSalidaProcesoPorInterrupt(k.EsperandoDesalojo[i].proceso_desalojado, nuevo_pc_proc_desalojado) {
					slog.Error("Error - (LlegaFinInterrupt) - No encontre al proceso desalojado en EXECUTE, CUANDO DEBERIA DE ESTAR AHHHHH")
					mutex_esperandoDesalojo.Unlock()
					return
				}

				slog.Debug("Debug - (LlegaFinInterrupt) - Se movio correctamente el proc. desalojado a READY")
				// https://www.youtube.com/watch?v=BiRCZLeIvvQ
			} else {
				slog.Debug("Debug - (LlegaFinInterrupt) - La ultima instruccion del proc. desalojado fue una syscall bloqueante, ya esta donde debe estar...")
			}

			mutex_CPUsConectadas.Lock()

			cpu_a_despachar := k.CPUsConectadas[id_cpu]

			//DEBUTANTE
			//ENTRA AQUINO, SENIORAS Y SENIORES
			k.AgregarAEstado(EstadoExecute, k.EsperandoDesalojo[i].proceso_titular, true)

			pid_desalojado_para_log := k.EsperandoDesalojo[i].proceso_desalojado.Pid

			handleDispatch(k.EsperandoDesalojo[i].proceso_titular.Pid, k.EsperandoDesalojo[i].proceso_titular.Pc, cpu_a_despachar.Url)
			actualizarCPU(cpu_a_despachar, k.EsperandoDesalojo[i].proceso_titular.Pid, k.EsperandoDesalojo[i].proceso_titular.Pc, false)
			cpu_a_despachar.EstaSiendoDesalojada = false
			utils.LoggerConFormato("## (%d) Pasa del estado READY al estado EXECUTE", k.EsperandoDesalojo[i].proceso_titular.Pid)
			//quito la instancia de esperandoDesalojo
			k.EsperandoDesalojo = append(k.EsperandoDesalojo[:i], k.EsperandoDesalojo[i+1:]...)
			utils.LoggerConFormato("## (%d) - Desalojado por algoritmo SJF/SRT", pid_desalojado_para_log)
			mutex_CPUsConectadas.Unlock()
			mutex_esperandoDesalojo.Unlock()

			return
		}

	}
	mutex_esperandoDesalojo.Unlock()
	slog.Error("Error - (LlegaFinInterrupt) - Por alguna razon, cuando me llego el fin de interrupt no encontre id de la CPU en EsperandoDesalojo")
}

func (k *Kernel) GestionarSalidaProcesoPorInterrupt(proceso_desalojado_a_act *PCB, nuevo_pc int) bool {
	//actualizo el biri bira del PCB saliente
	proceso_desalojado_a_act.Pc = nuevo_pc
	tiempo_en_cancha := duracionEnEstado(proceso_desalojado_a_act)
	k.actualizarEstimacionSJF(proceso_desalojado_a_act, tiempo_en_cancha)

	if !k.MoverDeEstadoPorPid(EstadoExecute, EstadoReady, proceso_desalojado_a_act.Pid, true) {
		return false
	}

	return true

}

/*















lastimosamente, todo lo bueno tiene lo malo...
la mejor funcion (con las MEJORES ANALOGIAS) que esta materia en sus años ha parido
quedo obsoleta...
Como el idolo de Agus una vez dijo...

"Eres un funcion increíble, diste lo mejor de ti y por eso te admiro.
Pasaste por varias nombres, fuiste tan complejo que todos nosotros te odiamos.
Espero que renazcas como una buena funcion, te estaré esperando para formatear kernel otra vez.
Yo también recursare operativos, recursare mucho para volverme más fuerte.
¡Adiós, CambiosEnElPlantel!"


------ QEPD -------
CambiosEnElPlantel
MudancasNoElenco
--- 2025 - 2025 ---

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


func (k *Kernel) buscarEnExpulsados(pid_expulsado_a_buscar int) bool {

	for i := 0; i < len(k.ExpulsadosPorRoja); i++ {
		if k.ExpulsadosPorRoja[i] == pid_expulsado_a_buscar {
			k.ExpulsadosPorRoja = append(k.ExpulsadosPorRoja[:i], k.ExpulsadosPorRoja[i+1:]...)
			return true
		}
	}

	return false
}



*/
