package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	utils "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (k *Kernel) registrarNuevaIO(w http.ResponseWriter, r *http.Request) { // Handshake
	var io_string string
	if err := json.NewDecoder(r.Body).Decode(&io_string); err != nil {
		fmt.Println("Error recibiendo la solicitud:", err)
		return
	}

	aux := strings.Split(io_string, " ") // NOMBRE IP PUERTO

	if len(aux) != 3 {
		http.Error(w, "Formato invalido", http.StatusBadRequest)
		return
	}

	nombre_io := aux[0] //Para mas claridad :p
	url := fmt.Sprintf("http://%s:%s", aux[1], aux[2])

	nuevaIO := &IO{
		Url:        url,
		Pid:        -1,
		Esta_libre: true,
	}

	//mutex IOS
	mutex_ios.Lock()
	defer mutex_ios.Unlock()

	// Si no existe una io con ese nombre, lo agrego nuevito
	if _, ok := k.ios[nombre_io]; !ok {
		// Agrego y sincronizo el nuevo dispositivo io

		k.ios[nombre_io] = &IOS{
			io:                 []*IO{nuevaIO},
			procEsperandoPorIO: []*ProcesoEsperandoIO{},
		}
	} else {
		// Sino, actualizo los valores en esa posicion
		k.ios[nombre_io].io = append(k.ios[nombre_io].io, nuevaIO)
	}

}

func enviarProcesoAIO(io_seleccionada *IO, duracion int) {

	fullURL := fmt.Sprintf("%s/io/hace_algo", io_seleccionada.Url)
	datos := fmt.Sprintf("%d %d", io_seleccionada.Pid, duracion)

	utils.EnviarSolicitudHTTPString("POST", fullURL, datos)
}

func (k *Kernel) RecibirFinIO(w http.ResponseWriter, r *http.Request) {
	var respuesta string
	if err := json.NewDecoder(r.Body).Decode(&respuesta); err != nil {
		fmt.Println("Error recibiendo la solicitud:", err)
		return
	}

	data := strings.Split(respuesta, " ")

	if len(data) < 2 {
		fmt.Printf("Respuesta IO mal formada: %s", respuesta)
		return
	}

	pid_io, _ := strconv.Atoi(data[0])
	nombre_io := data[1]

	k.MoverDeEstadoPorPid(EstadoBlock, EstadoReady, pid_io, true)
	k.MoverDeEstadoPorPid(EstadoBlockSuspended, EstadoReadySuspended, pid_io, true)

	k.ActualizarIO(nombre_io, pid_io)

	utils.LoggerConFormato("## (%d) finalizo IO y pasa a READY", pid_io)

}

func (k *Kernel) ActualizarIO(nombre_io string, pid_asociado_io int) {
	//mutex IOS
	mutex_ios.Lock()
	defer mutex_ios.Unlock()

	ios := k.ios[nombre_io]
	//Actualizar IO actual
	var io_usada *IO
	for _, io := range ios.io {
		if io.Pid == pid_asociado_io {
			io_usada = io
			break
		}
	}

	io_usada.Pid = -1
	io_usada.Esta_libre = true //la io termino ahora esta libre

	if len(ios.procEsperandoPorIO) != 0 {
		//desencolar el 1ro
		proceso_sgte := ios.procEsperandoPorIO[0]
		ios.procEsperandoPorIO = ios.procEsperandoPorIO[1:]

		io_usada.Pid = proceso_sgte.pid
		io_usada.Esta_libre = false

		// mandarlo a io
		enviarProcesoAIO(io_usada, proceso_sgte.tiempo_io)
	}

}

func (k *Kernel) buscarIOLibre(nombre string) *IO {

	if iosDispo, ok := k.ios[nombre]; ok {
		for _, instancia := range iosDispo.io {
			if instancia.Esta_libre {
				return instancia
			}
		}
	}

	return nil

}

func (k *Kernel) FinalizarIO(w http.ResponseWriter, r *http.Request) {

	var respuesta string
	if err := json.NewDecoder(r.Body).Decode(&respuesta); err != nil {
		fmt.Println("Error recibiendo la solicitud:", err)
		return
	}

	data := strings.Split(respuesta, " ")
	nombre_io := data[0]
	url_io := data[1]

	//mutex IOS
	mutex_ios.Lock()
	defer mutex_ios.Unlock()

	ios_del_mismo_nombre := k.ios[nombre_io]
	for i, valor := range ios_del_mismo_nombre.io {
		if valor.Url == url_io {
			ios_del_mismo_nombre.io = append(ios_del_mismo_nombre.io[:i], ios_del_mismo_nombre.io[i+1:]...)

			utils.LoggerConFormato("Una instancia de IO %s fue finalizada", nombre_io)
			return
		}
	}

	fmt.Printf("No se encontro para desconectar una instancia %s de IO pedida", nombre_io)
}

func (k *Kernel) ManejarIO(nombre_io string, cpu_ejecutando *CPU, duracion int) {
	defer RegistrarCPUaLibre(cpu_ejecutando)

	//mutex IOs
	mutex_ios.Lock()
	defer mutex_ios.Unlock()

	io, existeIO := k.ios[nombre_io]

	if !existeIO {
		k.GestionarEXIT(cpu_ejecutando)
		return
	}

	pcb := k.BuscarPorPidSeguro(EstadoExecute, cpu_ejecutando.Pid)
	pcb.Pc = cpu_ejecutando.Pc
	k.MoverDeEstadoPorPid(EstadoExecute, EstadoBlock, cpu_ejecutando.Pid, true)
	go k.temporizadorSuspension(pcb.Pid) // ta raro

	IO_seleccionada := k.buscarIOLibre(nombre_io)

	if IO_seleccionada == nil { //no hay io libre
		nuevo_proc_esperando := &ProcesoEsperandoIO{
			pid:       cpu_ejecutando.Pid,
			tiempo_io: duracion,
		}
		io.procEsperandoPorIO = append(io.procEsperandoPorIO, nuevo_proc_esperando)
		return
	}
	// si hay io libre
	IO_seleccionada.Pid = cpu_ejecutando.Pid
	//enviar a io
	IO_seleccionada.Esta_libre = false
	utils.LoggerConFormato("## (%d) - Bloqueado por IO: %s", IO_seleccionada.Pid, nombre_io)
	enviarProcesoAIO(IO_seleccionada, duracion)
}
