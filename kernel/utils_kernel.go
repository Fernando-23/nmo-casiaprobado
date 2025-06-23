package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"slices"

	utils "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

// array de arrays que contenga a todas las colas

func IniciarConfiguracion[T any](ruta string, estructuraDeConfig *T) error {

	fmt.Println("Cargando configuracion desde", ruta)
	configFile, err := os.Open(ruta)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de configuracion: %w", err)
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	if err := jsonParser.Decode(estructuraDeConfig); err != nil {
		return fmt.Errorf("error al decodificar la configuracion %w", err)
	}
	return nil

}

func esperarEnter(signalEnter chan struct{}) {

	fmt.Println("Ingrese ENTER para empezar con la planificacion")

	reader := bufio.NewReader(os.Stdin)
	_, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Println("Error leyendo del teclado:", err)
	}

	signalEnter <- struct{}{} //Envia una señal para avisar al hilo principal que el usuario presiono Enter

}

func (k *Kernel) InicializarMapaDeEstados() {
	k.procesoPorEstado = make(map[int][]*PCB)

	// Inicializamos todos los estados del map
	for i := 0; i < cantEstados; i++ {
		k.procesoPorEstado[i] = []*PCB{}
	}
}

func (k *Kernel) HandshakeMemoria() error {
	url := fmt.Sprintf("http://%s:%d/memoria/handshake", k.ConfigKernel.Ip_memoria, k.ConfigKernel.Puerto_Memoria)
	respuesta, err := utils.EnviarSolicitudHTTPString("GET", url, nil)
	if err != nil {
		return fmt.Errorf("memoria no responde: %w", err)
	}

	if respuesta != "OK" {
		return fmt.Errorf("respuesta inesperada de memoria: %s", respuesta)
	}

	return nil
}

//en memo
/*
func handlecheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
*/
func (k *Kernel) CrearPCB(tamanio int, arch_pseudo string) *PCB {

	pcb := &PCB{
		Pid:         k.pidActual,
		Tamanio:     tamanio,
		Arch_pseudo: arch_pseudo,
	}

	if k.ConfigKernel.Algoritmo_Plani == "SJF" {
		k.CrearSJF(pcb)
	} else {
		pcb.SJF = nil
	}

	k.pidActual++
	//pcb.Pc = 0
	return pcb
}

func FIFO(l_estado *[]*PCB, pcb *PCB) { //FIFO
	*l_estado = append(*l_estado, pcb)
}

func cambiarMetricasEstado(pPcb *PCB, posEstado int) {
	pPcb.Me[posEstado]++ //ver si puede quedar mas lindo

}

func duracionEnEstado(pPcb *PCB) time.Duration {
	tiempoActual := time.Now()
	return tiempoActual.Sub(pPcb.HoraIngresoAEstado)
}

func cambiarMetricasTiempo(pPcb *PCB, posEstado int) {
	pPcb.Mt[posEstado] += duracionEnEstado(pPcb)
}

func (k *Kernel) AgregarAEstado(estado int, pcb *PCB) {
	mutex_procesoPorEstado[estado].Lock()
	defer mutex_procesoPorEstado[estado].Unlock()

	k.procesoPorEstado[estado] = append(k.procesoPorEstado[estado], pcb)
}

func (k *Kernel) QuitarYObtenerPCB(estado int, pid int) *PCB {
	mutex_procesoPorEstado[estado].Lock()
	defer mutex_procesoPorEstado[estado].Unlock()

	procesos := k.procesoPorEstado[estado]
	for i, pcb := range procesos {
		if pcb.Pid == pid {
			// Quitar el proceso de la lista del estado
			k.procesoPorEstado[estado] = slices.Delete(procesos, i, i+1)
			return pcb
		}
	}
	return nil //no se encontro
}

func (k *Kernel) QuitarPrimerElemento(estado int) *PCB {
	mutex_procesoPorEstado[estado].Lock()
	defer mutex_procesoPorEstado[estado].Unlock()

	pcbs := k.procesoPorEstado[estado]

	if len(pcbs) == 0 {
		return nil
	}

	primer_elemento := pcbs[0]
	k.procesoPorEstado[estado] = pcbs[1:]

	return primer_elemento

}

func (k *Kernel) MoverDeEstadoPorPid(estadoActual, estadoNuevo int, pid int) {
	// Buscar el puntero al PCB en el estado actual
	pcb := k.QuitarYObtenerPCB(estadoActual, pid) //aca sincroniza

	if pcb == nil {
		fmt.Printf("Proceso %d no encontrado en el estado %d\n", pid, estadoActual)
		return
	}

	estadoAnterior := estadoActual

	// Actualizar metricas no requiere sincro porque nadie tiene acceso ya a este pcb no pertenece a ningun estado en este instante
	//sincronizacion inplicita

	cambiarMetricasTiempo(pcb, estadoAnterior)

	pcb.HoraIngresoAEstado = time.Now()

	cambiarMetricasEstado(pcb, estadoNuevo)

	utils.LoggerConFormato("## (%d) Pasa del estado %s al estado %s", pid, estados_proceso[estadoActual], estados_proceso[estadoNuevo])

	// Cambiar el estado del proceso
	pcb.estado = estadoNuevo

	// Agregar el puntero al PCB al nuevo estado
	k.AgregarAEstado(estadoNuevo, pcb) //aca sincroniza

}

func (k *Kernel) MoverDeEstado(estadoActual, estadoNuevo int) {

	pcb := k.QuitarPrimerElemento(estadoActual) //ya sincroniza internamente

	if pcb == nil {
		fmt.Printf("No hay procesos en el estado %d\n", estadoActual)
		return
	}
	// Actualizar métricas
	cambiarMetricasTiempo(pcb, estadoActual)

	pcb.HoraIngresoAEstado = time.Now()

	cambiarMetricasEstado(pcb, estadoNuevo)

	utils.LoggerConFormato("## (%d) Pasa del estado %s al estado %s", pcb.Pid, estados_proceso[estadoActual], estados_proceso[estadoNuevo])

	// Cambiar el estado del proceso
	pcb.estado = estadoNuevo

	// Agregar el puntero al PCB al nuevo estado
	k.AgregarAEstado(estadoNuevo, pcb) //ya sincroniza internamente

}

func (k *Kernel) CrearSJF(pcb *PCB) {
	sjf := &SJF{
		Estimado_anterior: k.ConfigKernel.Estimacion_Inicial,
		Estimado_actual:   k.ConfigKernel.Estimacion_Inicial,
		Real_anterior:     0, //no ejecuto valor igual a 0
	}
	pcb.SJF = sjf
}

func (k *Kernel) actualizarEstimacionSJF(pcb *PCB, tiempoEnExecute time.Duration) {
	if pcb == nil || pcb.SJF == nil {
		return
	}
	real_anterior := float64(tiempoEnExecute.Milliseconds())
	alpha := k.ConfigKernel.Alfa
	sjf := pcb.SJF
	aux := sjf.Estimado_actual
	sjf.Estimado_actual = (alpha * real_anterior) + ((1 - alpha) * sjf.Estimado_anterior)
	sjf.Estimado_anterior = aux
}

func (k *Kernel) IniciarProceso(tamanio int, arch_pseudo string) *PCB {
	pcb := k.CrearPCB(tamanio, arch_pseudo)
	pcb.HoraIngresoAEstado = time.Now()

	cambiarMetricasEstado(pcb, EstadoNew)
	//k.AgregarAEstado(EstadoNew, pcb) //meter en la cola new no hay planificacion para meter en la cola new
	return pcb
}

func (k *Kernel) UnicoEnNewYNadaEnSuspReady() (bool, error) {
	mutex_procesoPorEstado[EstadoNew].Lock()
	mutex_procesoPorEstado[EstadoReadySuspended].Lock()

	lista_new := k.procesoPorEstado[EstadoNew]
	lista_susp_ready := k.procesoPorEstado[EstadoReadySuspended]

	//Caso en en que hay exactamente un proceso en NEW y ninguno en SUSPENDED-READY
	if len(lista_susp_ready) == 0 && len(lista_new) == 1 {
		fmt.Println("Soy el primer elemento y no hay procesos en SUSP READY", lista_new[0].Pid)
		primer_elemento := lista_new[0]

		//hay que desbloquear porque vamos a hacer una peticion http y no da que siga reteniendo el recurso
		//y ademas como si fuera poco ya no necesita acceder a las listas hasta el mover deestado que tiene su debida sincronizacion
		mutex_procesoPorEstado[EstadoReadySuspended].Unlock()
		mutex_procesoPorEstado[EstadoNew].Unlock()

		hay_espacio, err := k.MemoHayEspacio(primer_elemento.Pid, primer_elemento.Tamanio, primer_elemento.Arch_pseudo)

		if err != nil {
			log.Printf("Error codificando mensaje: %s", err.Error())
			return true, err
		}
		if hay_espacio {
			// Cambiar de estado del proceso de NEW a READY
			k.MoverDeEstado(EstadoNew, EstadoReady) //aca sincroniza
		}
		return true, nil
	}
	return false, nil
}

func (k *Kernel) PlanificarLargoPorLista(codLista int) (bool, error) {
	mutex_procesoPorEstado[codLista].Lock()

	//si el algoritmo es PCMP, ordenamos antes de tomar el primero
	if k.ConfigKernel.Ready_ingress_algorithm == "PCMP" {
		sort.Sort(PorTamanio(k.procesoPorEstado[codLista]))
	}

	mutex_procesoPorEstado[codLista].Unlock()

	pcb := k.QuitarPrimerElemento(codLista)

	if pcb == nil {
		return false, nil
	}

	hay_espacio, err := k.MemoHayEspacio(pcb.Pid, pcb.Tamanio, pcb.Arch_pseudo)

	if err != nil {
		return false, err
	}

	if hay_espacio {
		k.MoverDeEstado(codLista, EstadoReady)
		return true, nil
	}
	return false, nil
}

func (k *Kernel) TieneProcesos(estado int) bool {
	mutex_procesoPorEstado[estado].Lock()
	defer mutex_procesoPorEstado[estado].Unlock()
	return len(k.procesoPorEstado[estado]) > 0
}

func (k *Kernel) IntentarEnviarProcesoAReady() (bool, error) {

	if k.TieneProcesos(EstadoReadySuspended) {
		return k.PlanificarLargoPorLista(EstadoReadySuspended)
	}
	return k.PlanificarLargoPorLista(EstadoNew)
}

func (k *Kernel) ObtenerCPULibre() *CPU {

	mutex_cpus_libres.Lock()
	defer mutex_cpus_libres.Unlock()

	for _, cpu := range k.cpusLibres {
		if cpu.Esta_libre {
			return cpu // La primera CPU que esta libre
		}
	}
	return nil // No hay CPU libre
}

func (k *Kernel) ChequearSiHayQueDesalojar() {
	mutex_procesoPorEstado[EstadoExecute].Lock()
	defer mutex_procesoPorEstado[EstadoExecute].Unlock()

	mutex_cpus_libres.Lock()
	defer mutex_cpus_libres.Unlock()

	primer_elemento := k.procesoPorEstado[EstadoReady][0]
	estimacion := primer_elemento.SJF.Estimado_actual
	var maxEstimacionRestante float64 = -1
	var cpu_desalojo *CPU
	var pcb *PCB

	for _, cpu := range k.cpusLibres {

		pcb = k.BuscarPorPid(EstadoExecute, cpu.Pid)
		if pcb == nil {
			fmt.Println("no esta el proceso en la lista de execute en la que deberia estar")
			continue
		}

		tiempo_ejecutando := duracionEnEstado(pcb)
		estimacion_falta_ejecutar := pcb.SJF.Estimado_actual - float64(tiempo_ejecutando)

		if estimacion_falta_ejecutar > estimacion && estimacion_falta_ejecutar > maxEstimacionRestante {
			maxEstimacionRestante = estimacion_falta_ejecutar
			cpu_desalojo = cpu
		}
	}

	if cpu_desalojo != nil {
		fmt.Printf("Desalojando proceso %d de CPU %d (estimacion restante: %.2f) para ejecutar proceso %d (estimacion: %.2f)",
			cpu_desalojo.Pid, cpu_desalojo.ID, maxEstimacionRestante, primer_elemento.Pid, primer_elemento.SJF.Estimado_actual)

		pc_a_actualizar := EnviarInterrupt(cpu_desalojo)

		k.CambiosEnElPlantel(cpu_desalojo, pc_a_actualizar)

		return
	}

}

// -----------Informa el Club Atletico Velez Sarsfield------------
func (k *Kernel) CambiosEnElPlantel(cpu *CPU, pc_a_actualizar int) {
	// Debutante
	// CALIENTA KAROL
	utils.LoggerConFormato("## (%d) - Desalojado por algoritmo SJF/SRT", cpu.Pid)
	proceso_suplente := k.procesoPorEstado[EstadoReady][0]

	proceso_titular := k.BuscarPorPid(EstadoExecute, cpu.Pid)

	// Actualizamos pc en el pcb del proceso que estaba ejecutando
	proceso_titular.Pc = pc_a_actualizar

	// Ahora si desalojamos al pcb correspondiente
	tiempo_en_cancha := duracionEnEstado(proceso_titular)
	k.actualizarEstimacionSJF(proceso_titular, tiempo_en_cancha)
	k.MoverDeEstadoPorPid(EstadoExecute, EstadoReady, proceso_titular.Pid)

	// Actualizar la cpu con el proceso nuevo
	cpu.Pc = proceso_suplente.Pc
	cpu.Pid = proceso_suplente.Pid

	// Enviar nuevo proceso a cpu
	handleDispatch(cpu)
	// ENTRA AQUINO (Mi primo, que si aprobo el tp)
	k.MoverDeEstadoPorPid(EstadoReady, EstadoExecute, proceso_suplente.Pid)

	fmt.Printf("CAMBIO: Sale %d (est. %.2f), entra %d (est. %.2f)\n", // Leer con voz de gangoso
		proceso_titular.Pid, proceso_titular.SJF.Estimado_actual,
		proceso_suplente.Pid, proceso_suplente.SJF.Estimado_actual)
}

func EnviarInterrupt(cpu *CPU) int { // yo te hablo por la puerta interrupt y me desocupo
	fullURL := fmt.Sprintf("%s/interrupt", cpu.Url)
	resp, err := utils.EnviarSolicitudHTTPString("POST", fullURL, "OK")
	if err != nil {
		return -1
	}

	pc, _ := strconv.Atoi(resp)
	return pc
}

func (k *Kernel) BuscarPorPid(estado_actual int, pid_a_buscar int) *PCB {

	mutex_procesoPorEstado[estado_actual].Lock()
	defer mutex_procesoPorEstado[estado_actual].Unlock()

	// Buscar el puntero al PCB en el estado actual
	procesos := k.procesoPorEstado[estado_actual]
	var pcb *PCB
	for _, proceso := range procesos {
		if proceso.Pid == pid_a_buscar {
			pcb = proceso
			return pcb
		}
	}
	return nil
}

func (k *Kernel) PlaniCortoPlazo() bool {

	// Para FIFO ya esta preparada la lista

	if k.ConfigKernel.Algoritmo_Plani == "SJF" || k.ConfigKernel.Algoritmo_Plani == "SRT" {
		lista_ready := k.procesoPorEstado[EstadoReady]
		pcb_nuevo_pid := lista_ready[len(lista_ready)-1].Pid

		sort.Sort(PorSJF(k.procesoPorEstado[EstadoReady])) //SJF distinto de nil

		if k.ConfigKernel.Algoritmo_Plani == "SRT" && pcb_nuevo_pid == lista_ready[0].Pid { //  10 15 18 31 32 500
			return true
		}
	}
	return false
}

func (k *Kernel) IntentarEnviarProcesoAExecute() {
	if !k.TieneProcesos(EstadoReady) {
		fmt.Println("No hay procesos en READY")
		return
	}

	// intentamos asignarle cpu
	cpu_seleccionada := k.ObtenerCPULibre()

	hay_que_chequear_desalojo := k.PlaniCortoPlazo()

	//Tomamos el primer PCB tras la planificacion
	indice := 0
	pcb := k.procesoPorEstado[EstadoReady][indice]

	if cpu_seleccionada == nil { //no hay cpu libre
		if hay_que_chequear_desalojo {
			k.ChequearSiHayQueDesalojar()
			return
		}
	}

	cpu_seleccionada.Esta_libre = false
	cpu_seleccionada.Pid = pcb.Pid
	cpu_seleccionada.Pc = pcb.Pc

	if err := handleDispatch(cpu_seleccionada); err != nil {
		fmt.Printf("Error al despachar proceso a la cpu: %v\n", err)
		cpu_seleccionada.Esta_libre = true //Revertir si falla
	}

	k.MoverDeEstadoPorPid(EstadoReady, EstadoExecute, pcb.Pid)

	fmt.Printf("Proceso %d a Execute en CPU %d\n", pcb.Pid, cpu_seleccionada.ID)
}

func handleDispatch(cpu_seleccionada *CPU) error {

	fullURL := fmt.Sprintf("%s/dispatch", cpu_seleccionada.Url)

	datos := fmt.Sprintf("%d %d", cpu_seleccionada.Pid, cpu_seleccionada.Pc)
	_, err := utils.EnviarSolicitudHTTPString("POST", fullURL, datos)
	if err != nil {
		return err
	}

	return nil
}

func (k *Kernel) MemoHayEspacio(pid int, tamanio int, archivoPseudo string) (bool, error) {

	solicitud_memoria := fmt.Sprintf("%d %d %s", pid, tamanio, archivoPseudo)
	url := fmt.Sprintf("http://%s:%d/memoria/hay_lugar", k.ConfigKernel.Ip_memoria, k.ConfigKernel.Puerto_Memoria)
	respuesta, err := utils.EnviarSolicitudHTTPString("POST", url, solicitud_memoria)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return false, err
	}

	if respuesta == "Si kernel, hay espacio" {
		fmt.Println("PRUEBA - efectivamente, habia espacio")
		return true, nil
	}
	return false, nil

}

func RegistrarCPUaLibre(cpu_a_liberar *CPU) {
	cpu_a_liberar.Esta_libre = true
	cpu_a_liberar.Pid = -1
}

func (k *Kernel) temporizadorSuspension(pid int) {
	suspension := time.Duration(k.ConfigKernel.Tiempo_Suspension)
	time.Sleep(suspension)

	pcb := k.BuscarPorPid(EstadoBlock, pid)
	if pcb != nil {
		k.MoverDeEstadoPorPid(EstadoBlock, EstadoBlockSuspended, pid)
	}
}

func (k *Kernel) registrarNuevaCPU(w http.ResponseWriter, r *http.Request) { // Handshake

	fmt.Println("Al menos, entre a registrar nueva cpu")

	var mensajeCPU string
	if err := json.NewDecoder(r.Body).Decode(&mensajeCPU); err != nil {
		fmt.Println("Error recibiendo la solicitud:", err)
		http.Error(w, "Error en el formato de la solicitud", http.StatusBadRequest)
		return
	}

	aux := strings.Split(mensajeCPU, " ") //ID IP PUERTO

	if len(aux) != 3 {
		http.Error(w, "Formato invalido. Esperando: 'ID IP PUERTO'", http.StatusBadRequest)
		return
	}

	nueva_ID_CPU, err := strconv.Atoi(aux[0])

	if err != nil {
		http.Error(w, "ID de CPU inválido", http.StatusBadRequest)
		return
	}

	ip := aux[1]
	puerto := aux[2]
	url := fmt.Sprintf("http://%s:%s/cpu", ip, puerto)

	mutex_cpus_libres.Lock()
	defer mutex_cpus_libres.Unlock()

	if _, existe := k.cpusLibres[nueva_ID_CPU]; existe {
		http.Error(w, "Ya existe una CPU registrada con ese ID", http.StatusConflict)
		return
	}

	k.cpusLibres[nueva_ID_CPU] = crearCPU(nueva_ID_CPU, url)

	fmt.Printf("Se conecto una nueva CPU con ID %d en %s\n", nueva_ID_CPU, url)

	//RESPONDER OK
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

}

func crearCPU(id int, url string) *CPU {
	nueva_cpu := &CPU{
		ID:         id,
		Url:        url,
		Pid:        -1,
		Pc:         0,
		Esta_libre: true,
	}
	return nueva_cpu

}
