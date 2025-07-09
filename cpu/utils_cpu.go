package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (cpu *CPU) Fetch() string {
	// peticion := fmt.Sprintf("%d %d", *pid_ejecutando, *pc_ejecutando)
	// fullUrl := fmt.Sprintf("%s/fetch", url_memo)
	instruccion, _ := utils.FormatearUrlYEnviar(cpu.Url_memoria, "/fetch", true, "%d %d",
		cpu.Proc_ejecutando.Pid,
		cpu.Proc_ejecutando.Pc,
	)
	return instruccion
}

func (cpu *CPU) Decode(instruccion string) (string, []string) {
	l_instruccion := strings.Split(instruccion, " ")
	cod_op := l_instruccion[0]
	operacion := l_instruccion[1:]

	return cod_op, operacion
}

func (cpu *CPU) Execute(cod_op string, operacion []string, instruccion_completa string) {

	utils.LoggerConFormato("## PID: %d - Ejecutando: %s", cpu.Proc_ejecutando.Pid, instruccion_completa)
	switch cod_op {

	case "NOOP":
		//consume el tiempo de ciclo de instruccion

	case "WRITE":
		dir_logica, err := strconv.Atoi(operacion[0])

		if err != nil {
			slog.Error("Error - (Execute) - Pansando a int la direnccion logica")
			return
		}
		datos := operacion[1]

		respuesta, dir_fisica := cpu.RequestWRITE(dir_logica, datos)

		utils.LoggerConFormato("PID: %d - Acción: ESCRITURA - Dirección Física: [ %d |  %d  ] - Valor: %s",
			cpu.Proc_ejecutando.Pid,
			dir_fisica.frame,
			dir_fisica.offset,
			respuesta,
		)

	case "READ":
		dir_logica, err1 := strconv.Atoi(operacion[0])
		tamanio, err2 := strconv.Atoi(operacion[1])

		if err1 != nil || err2 != nil {
			slog.Error("Error - (Execute) - Pansando a int la direnccion logica o tamanio")
			return
		}

		//Gestionar mejor el error :p
		valor_leido, dir_fisica := cpu.RequestREAD(dir_logica, tamanio)
		//si el valor leido es un aviso de direccionamiento invalido
		//habilitar un hay_interrupcion

		utils.LoggerConFormato("PID: %d - Acción: LEER - Dirección Física: [ %d |  %d  ] - Valor: %s",
			cpu.Proc_ejecutando.Pid,
			dir_fisica.frame,
			dir_fisica.offset,
			valor_leido,
		)

	case "GOTO":

		nuevo_pc, err := strconv.Atoi(operacion[0])

		if err != nil {
			slog.Error("Error - (Execute) - Pansando a int PC")
			return
		}

		cpu.Proc_ejecutando.Pc = nuevo_pc

	// Syscalls
	case "IO":
		// ID_CPU PC IO TECLADO 20000

		mensaje_io := fmt.Sprintf("%s %d IO %s %s", cpu.Id, cpu.Proc_ejecutando.Pc, operacion[0], operacion[1])
		cpu.EnviarSyscall("IO", mensaje_io)

		HabilitarInterrupt(true)

	case "INIT_PROC":
		// ID_CPU PC INIT_PROC proceso1 256
		mensaje_init_proc := fmt.Sprintf("%s %d INIT_PROC %s %s", cpu.Id, cpu.Proc_ejecutando.Pc, operacion[0], operacion[1])
		cpu.EnviarSyscall("INIT_PROC", mensaje_init_proc)

		HabilitarInterrupt(false)

	case "DUMP_MEMORY":
		// ID_CPU PC DUMP_MEMORY
		mensaje_dump := fmt.Sprintf("%s %d DUMP_MEMORY", cpu.Id, cpu.Proc_ejecutando.Pc)
		cpu.EnviarSyscall("DUMP_MEMORY", mensaje_dump)
		HabilitarInterrupt(true)

	case "EXIT":
		// ID_CPU PC DUMP_MEMORY
		mensaje_exit := fmt.Sprintf("%s %d EXIT", cpu.Id, cpu.Proc_ejecutando.Pc)
		cpu.EnviarSyscall("EXIT", mensaje_exit)

		HabilitarInterrupt(true)

	default:
		slog.Error("Error - (Execute) - ingrese una instruccion valida")
	}

	// Incrementar PC
	if cod_op != "GOTO" {
		cpu.Proc_ejecutando.Pc++
	}

}

func (cpu *CPU) RecibirInterrupt(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (RecibirInterrupt) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}

	mensajeKernel := string(body_Bytes)

	slog.Debug("Llego interrrupción desde kernel", "mensaje", mensajeKernel)

	if mensajeKernel == "OK" {
		HabilitarInterrupt(true)
		utils.FormatearUrlYEnviar(cpu.Url_kernel, "/interrumpido", false, "%s %d %d", cpu.Id, cpu.Proc_ejecutando.Pid, cpu.Proc_ejecutando.Pc)
		return
	}
}

func (cpu *CPU) ChequarTLBActiva() {
	if cpu.Config_CPU.Cant_entradas_TLB > 0 {
		tlb_activa = true
	}
}

func (cpu *CPU) ChequearCachePagsActiva() {
	if cpu.Config_CPU.Cant_entradas_cache > 0 {
		cache_pags_activa = true
	}
}

func (cpu *CPU) LiberarTLB() {
	slog.Debug("Debug - (LiberarTLB) - Liberando entradas TLB...")
	for i := 0; i < cpu.Config_CPU.Cant_entradas_TLB; i++ {
		cpu.Tlb[i].pagina = -1
		cpu.Tlb[i].frame = -1
	}
}

func (cpu *CPU) InicializarTLB() {
	slog.Debug("Debug - (InicializarTLB) - Inicializando entradas TLB...")
	for i := 0; i < cpu.Config_CPU.Cant_entradas_TLB; i++ {
		cpu.Tlb[i] = &EntradaTLB{
			pagina: -1,
			frame:  -1,
		}
	}
}

func (cpu *CPU) IniciarCachePags() {

	slog.Debug("Debug - (IniciarCachePags) - Inicializando entradas CachePags...")
	for i := 0; i < cpu.Config_CPU.Cant_entradas_cache; i++ {
		cpu.Cache_pags[i] = &EntradaCachePag{
			pagina:    -1,
			contenido: "",
		}
	}
}

func (cpu *CPU) LiberarCachePags() {
	for i := 0; i < cpu.Config_CPU.Cant_entradas_cache; i++ {
		if cpu.Cache_pags[i] != nil && cpu.Cache_pags[i].bit_modificado == 1 {
			cpu.ActualizarPagCompleta(cpu.Cache_pags[i])
		}
		cpu.Cache_pags[i].pagina = -1
		cpu.Cache_pags[i].contenido = ""
	}

}

func (cpu *CPU) LiberarCaches() {

	if tlb_activa {
		cpu.LiberarTLB()
	}

	if cache_pags_activa {
		cpu.LiberarCachePags()
	}
}

func HabilitarInterrupt(valor_vg bool) {
	mutex_hay_interrupcion.Lock()
	hay_interrupcion = valor_vg
	mutex_hay_interrupcion.Unlock()
}

// var hola int = 0

func crearCPU(id string, path_config string) *CPU {

	p_config := new(ConfigCPU)

	if err := utils.IniciarConfiguracion(path_config, p_config); err != nil {
		panic(fmt.Sprintf("Error cargando config CPU: %v", err))
	}

	url_kernel := fmt.Sprintf("http://%s:%d/kernel", p_config.Ip_Kernel, p_config.Puerto_Kernel)
	url_memo := fmt.Sprintf("http://%s:%d/memoria", p_config.Ip_Memoria, p_config.Puerto_Memoria)

	cant_entradas_tlb := p_config.Cant_entradas_TLB
	cant_entradas_cache := p_config.Cant_entradas_cache

	aux_proc := &Proceso{Pid: 0, Pc: 0}
	tlb := make([]*EntradaTLB, cant_entradas_tlb)
	cache := make([]*EntradaCachePag, cant_entradas_cache)

	cpu := &CPU{
		Id:              id,
		Proc_ejecutando: aux_proc,
		Config_CPU:      p_config,
		Url_memoria:     url_memo,
		Url_kernel:      url_kernel,
		Tlb:             tlb,
		Cache_pags:      cache,
	}
	//Configurar Logger
	nombre_log := "cpu" + id
	if err := utils.ConfigurarLogger(nombre_log, cpu.Config_CPU.Log_level); err != nil {
		panic(fmt.Sprintf("Error creando log para CPU %s: %v", id, err))

	}

	return cpu
}
