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
	var tipo_instruccion string
	switch cod_op {

	case "NOOP":
		//consume el tiempo de ciclo de instruccion
		tipo_instruccion = "REQUIERO_DESALOJO"
	case "WRITE":
		dir_logica, err := strconv.Atoi(operacion[0])

		if err != nil {
			slog.Error("Error - (Execute) - Pansando a int la direnccion logica")
			return
		}
		datos := operacion[1]

		_, dir_fisica := cpu.RequestWRITE(dir_logica, datos)

		utils.LoggerConFormato("PID: %d - Acción: ESCRITURA - Dirección Física: [ %d |  %d  ] - Valor: %s",
			cpu.Proc_ejecutando.Pid,
			dir_fisica.frame,
			dir_fisica.offset,
			datos,
		)
		tipo_instruccion = "REQUIERO_DESALOJO"

	case "READ":
		dir_logica, err1 := strconv.Atoi(operacion[0])
		tamanio, err2 := strconv.Atoi(operacion[1])

		if err1 != nil || err2 != nil {
			slog.Error("Error - (Execute) - Pasando a int la direnccion logica o tamanio")
			return
		}

		valor_leido, dir_fisica := cpu.RequestREAD(dir_logica, tamanio)

		utils.LoggerConFormato("PID: %d - Acción: LEER - Dirección Física: [ %d |  %d  ] - Valor: %s",
			cpu.Proc_ejecutando.Pid,
			dir_fisica.frame,
			dir_fisica.offset,
			valor_leido,
		)
		tipo_instruccion = "REQUIERO_DESALOJO"
	case "GOTO":

		nuevo_pc, err := strconv.Atoi(operacion[0])

		if err != nil {
			slog.Error("Error - (Execute) - Pansando a int PC")
			return
		}

		cpu.Proc_ejecutando.Pc = nuevo_pc
		tipo_instruccion = "REQUIERO_DESALOJO"
	// Syscalls
	case "IO":
		// ID_CPU PC IO TECLADO 20000

		pc_a_actualizar := cpu.Proc_ejecutando.Pc + 1 //le mando a kernel la siguiente instruccion
		mensaje_io := fmt.Sprintf("%s %d IO %s %s", cpu.Id, pc_a_actualizar, operacion[0], operacion[1])
		cpu.EnviarSyscall("IO", mensaje_io)

		CambiarValorActualizarContexto(true)

		CambiarValorTengoQueActualizarEnKernel(true)
		tipo_instruccion = "NO_REQUIERO_DESALOJO"

	case "INIT_PROC":
		// ID_CPU PC INIT_PROC proceso1 256
		pc_a_actualizar := cpu.Proc_ejecutando.Pc + 1
		mensaje_init_proc := fmt.Sprintf("%d %d INIT_PROC %s %s", cpu.Proc_ejecutando.Pid, pc_a_actualizar, operacion[0], operacion[1])
		cpu.EnviarSyscall("INIT_PROC", mensaje_init_proc)

		//deberia por default estar los 2 en false, peeeero, para asegurarnos, que los setee igual
		CambiarValorActualizarContexto(false)

		CambiarValorTengoQueActualizarEnKernel(false)
		tipo_instruccion = "REQUIERO_DESALOJO"
	case "DUMP_MEMORY":
		// ID_CPU PC DUMP_MEMORY
		pc_a_actualizar := cpu.Proc_ejecutando.Pc + 1
		mensaje_dump := fmt.Sprintf("%s %d DUMP_MEMORY", cpu.Id, pc_a_actualizar)
		cpu.EnviarSyscall("DUMP_MEMORY", mensaje_dump)

		CambiarValorActualizarContexto(true)

		CambiarValorTengoQueActualizarEnKernel(true)
		tipo_instruccion = "NO_REQUIERO_DESALOJO"
	case "EXIT":
		// ID_CPU PC DUMP_MEMORY
		pc_a_actualizar := cpu.Proc_ejecutando.Pc + 1
		mensaje_exit := fmt.Sprintf("%s %d EXIT", cpu.Id, pc_a_actualizar)
		cpu.EnviarSyscall("EXIT", mensaje_exit)

		CambiarValorActualizarContexto(true)

		CambiarValorTengoQueActualizarEnKernel(true)
		tipo_instruccion = "NO_REQUIERO_DESALOJO"

	default:
		slog.Error("Error - (Execute) - ingrese una instruccion valida")
	}

	// Incrementar PC
	if cod_op != "GOTO" {
		cpu.Proc_ejecutando.Pc++
	}

	cpu.CheckInterrupt(cpu.Proc_ejecutando.Pc, tipo_instruccion)
}

func (cpu *CPU) RecibirInterrupt(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (RecibirInterrupt) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensajeKernel := string(body_Bytes)

	slog.Debug("Llego interrupción desde kernel", "mensaje", mensajeKernel)
	mutex_tenemosInterrupt.Lock()
	tenemos_interrupt = true
	mutex_tenemosInterrupt.Unlock()

	nuevo_pc := <-ch_respuesta_interrupt
	mensaje_a_enviar := fmt.Sprintf("%d %d", nuevo_pc, cpu.Proc_ejecutando.Pid)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(mensaje_a_enviar))

}

func (cpu *CPU) CheckInterrupt(pc_a_actualizar int, tipo_instruccion string) {
	mutex_tenemosInterrupt.Lock()
	if tenemos_interrupt {
		ch_respuesta_interrupt <- pc_a_actualizar
		ch_tipo_instruccion <- tipo_instruccion
		mutex_tenemosInterrupt.Unlock()

		CambiarValorActualizarContexto(true)

		CambiarValorTengoQueActualizarEnKernel(false)

		return
	}
	mutex_tenemosInterrupt.Unlock()
}

func CambiarValorActualizarContexto(nuevo_valor bool) {
	mutex_hayQueActualizarContexto.Lock()
	hay_que_actualizar_contexto = nuevo_valor
	mutex_hayQueActualizarContexto.Unlock()
}

func CambiarValorTengoQueActualizarEnKernel(nuevo_valor bool) {
	mutex_tengoQueActualizarEnKernel.Lock()
	tengo_que_actualizar_en_kernel = true
	mutex_tengoQueActualizarEnKernel.Unlock()
}

func (cpu *CPU) ChequearSiTengoQueActualizarEnKernel() {
	mutex_tengoQueActualizarEnKernel.Lock()
	if tengo_que_actualizar_en_kernel {
		mutex_tengoQueActualizarEnKernel.Unlock()
		cpu.ActualizarContexto()
		slog.Debug("Debug - (ChequearSiTengoQueActualizarEnKernel) - No hubo interrupcion, mando senial para liberarme")
		return
	}
	mutex_tengoQueActualizarEnKernel.Unlock()

	slog.Debug("Debug - (ChequearSiTengoQueActualizarEnKernel) - Probablemente hubo una interrupcion")
}

func (cpu *CPU) ActualizarContexto() {
	utils.FormatearUrlYEnviar(cpu.Url_kernel, "/actualizar_contexto", false, "%s %d", cpu.Id, cpu.Proc_ejecutando.Pc)
}

//###################################################
//#####
//###################################################

func (cpu *CPU) ChequarTLBActiva() {
	tlb_activa = false
	if cpu.Config_CPU.Cant_entradas_TLB > 0 {
		tlb_activa = true
	}
}

func (cpu *CPU) ChequearCachePagsActiva() {
	cache_pags_activa = false
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
		cpu.Cache_pags[i].bit_modificado = 0
		cpu.Cache_pags[i].bit_uso = 0
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

// func (cpu *CPU) MostrarEstadoTLB(){
// 	slog.Debug("Debug - (MostrarEstadoTLB) - Estado de la TLB:")
// }
