package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"slices"

	utils "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (k *Kernel) llegaNuevaIO(w http.ResponseWriter, r *http.Request) { // Handshake
	var mensajeIO string
	if err := json.NewDecoder(r.Body).Decode(&mensajeIO); err != nil {
		fmt.Println("Error recibiendo la solicitud:", err)
		return
	}

	nombre, ip, puerto, err := decodificarMensajeNuevaIO(mensajeIO)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	go k.registrarNuevaIO(nombre, ip, puerto)

	utils.LoggerConFormato("Fin de registrarNuevaIO, nombre io: %s", nombre)
}

func decodificarMensajeNuevaIO(mensaje string) (nombre, ip, puerto string, err error) {
	partes := strings.Split(mensaje, " ")
	if len(partes) != 3 {
		return "", "", "", fmt.Errorf("se espera formato: NOMBRE IP PUERTO")
	}
	nombre, ip, puerto = partes[0], partes[1], partes[2]
	if nombre == "" || ip == "" || puerto == "" {
		return "", "", "", fmt.Errorf("campos vacíos en mensaje")
	}
	return nombre, ip, puerto, nil
}

func (k *Kernel) registrarNuevaIO(nombre, ip, puerto string) { // Handshake

	url := fmt.Sprintf("http://%s:%s", ip, puerto)

	nuevaInstancia := &DispositivoIO{
		Url:         url,
		PidOcupante: -1,
		Libre:       true,
	}

	//mutex IOS
	mutex_DispositivosIO.Lock()
	defer mutex_DispositivosIO.Unlock()

	// Si no existe una io con ese nombre, lo agrego nuevito
	if _, existe := k.DispositivosIO[nombre]; !existe {
		// Agrego y sincronizo el nuevo dispositivo io
		k.DispositivosIO[nombre] = &InstanciasPorDispositivo{
			Instancias: []*DispositivoIO{nuevaInstancia},
			ColaEspera: []*ProcesoEsperandoIO{},
		}
		fmt.Printf("Nueva IO registrada: %s en %s\n", nombre, url)
	} else {
		// Sino, actualizo los valores en esa posicion
		k.DispositivosIO[nombre].Instancias = append(k.DispositivosIO[nombre].Instancias, nuevaInstancia)
		fmt.Printf("Se agregó nueva instancia de IO '%s' conectada en %s\n", nombre, url)
	}
}

func (k *Kernel) llegaFinIO(w http.ResponseWriter, r *http.Request) {
	var mensaje_IO string
	if err := json.NewDecoder(r.Body).Decode(&mensaje_IO); err != nil {
		fmt.Println("Error recibiendo la solicitud:", err)
		return
	}

	if !k.liberarInstanciaIO(mensaje_IO) {
		http.Error(w, "No se pudo eliminar la IO", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (k *Kernel) liberarInstanciaIO(mensaje_IO string) bool {

	data := strings.Split(mensaje_IO, " ")

	if len(data) < 2 {
		fmt.Printf("Respuesta IO mal formada: %s", mensaje_IO)
		return false
	}

	pid_io, err := strconv.Atoi(data[0])

	if err != nil {
		fmt.Printf("PID inválido recibido en mensaje IO: %s\n", mensaje_IO)
		return false
	}
	nombre_io := data[1]

	k.MoverDeEstadoPorPid(EstadoBlock, EstadoReady, pid_io, true)
	k.MoverDeEstadoPorPid(EstadoBlockSuspended, EstadoReadySuspended, pid_io, true)

	k.ActualizarIO(nombre_io, pid_io)

	utils.LoggerConFormato("## (%d) finalizo IO y pasa a READY", pid_io)

	return true

}

func (k *Kernel) ActualizarIO(nombre_io string, pid_asociado_io int) {
	//mutex IOS
	mutex_DispositivosIO.Lock()
	defer mutex_DispositivosIO.Unlock()

	ios := k.DispositivosIO[nombre_io]
	//Actualizar IO actual
	var io_usada *DispositivoIO
	for _, io := range ios.Instancias {
		if io.PidOcupante == pid_asociado_io {
			io_usada = io
			break
		}
	}

	if io_usada == nil {
		fmt.Printf("No se encontró IO en uso con PID %d en %s\n", pid_asociado_io, nombre_io)
		return
	}

	io_usada.PidOcupante = -1
	io_usada.Libre = true //la io termino ahora esta libre

	if len(ios.ColaEspera) != 0 {
		//desencolar el 1ro
		proceso_sgte := ios.ColaEspera[0]
		ios.ColaEspera = ios.ColaEspera[1:]

		io_usada.PidOcupante = proceso_sgte.Pid
		io_usada.Libre = false

		// mandarlo a io
		enviarProcesoAIO(io_usada, proceso_sgte.TiempoIO)
	}

}

func enviarProcesoAIO(io_seleccionada *DispositivoIO, duracion int) {

	fullURL := fmt.Sprintf("%s/io/hace_algo", io_seleccionada.Url)
	datos := fmt.Sprintf("%d %d", io_seleccionada.PidOcupante, duracion)

	utils.EnviarSolicitudHTTPString("POST", fullURL, datos)

	utils.LoggerConFormato("## (%d) - Se envió proceso a IO en %s", io_seleccionada.PidOcupante, io_seleccionada.Url)
}

func (k *Kernel) buscarIOLibre(nombre string) *DispositivoIO {

	if iosDispo, ok := k.DispositivosIO[nombre]; ok {
		for _, instancia := range iosDispo.Instancias {
			if instancia.Libre {
				return instancia
			}
		}
	}

	return nil

}

func (k *Kernel) llegaDesconeccionIO(w http.ResponseWriter, r *http.Request) {

	var mensaje_IO string
	if err := json.NewDecoder(r.Body).Decode(&mensaje_IO); err != nil {
		utils.LoggerConFormato("[ERROR] FinalizarIO - Error al decodificar cuerpo: %v", err)
		return
	}
	k.FinalizarIO(mensaje_IO)
}

func (k *Kernel) FinalizarIO(mensaje_IO string) {

	partes := strings.Split(mensaje_IO, " ") // Esperado: "NOMBRE_IO URL_IO TIEMPO_IO"

	if len(partes) != 3 {
		utils.LoggerConFormato("[ERROR] FinalizarIO - Formato inválido: %s -Esperado: NOMBRE_IO URL_IO TIEMPO_IO", mensaje_IO)
		return
	}

	nombreIO := partes[0]
	urlIO := partes[1]
	tiempoIO, err := strconv.Atoi(partes[2])

	if err != nil {
		utils.LoggerConFormato("[ERROR] FinalizarIO - Tiempo inválido: %s", partes[2])
	}

	//mutex IOS
	mutex_DispositivosIO.Lock()
	defer mutex_DispositivosIO.Unlock()

	iosMismoNombre, ok := k.DispositivosIO[nombreIO]

	if !ok {
		utils.LoggerConFormato("[WARN] FinalizarIO - No existe el dispositivo IO con nombre %s", nombreIO)
		return
	}

	for i, instancia := range iosMismoNombre.Instancias {
		if instancia.Url == urlIO {

			if tiempoIO > 0 {
				aux_a_encolar := &ProcesoEsperandoIO{
					Pid:      instancia.PidOcupante,
					TiempoIO: tiempoIO,
				}

				iosMismoNombre.ColaEspera = append(iosMismoNombre.ColaEspera, aux_a_encolar)

			}

			iosMismoNombre.Instancias = slices.Delete(iosMismoNombre.Instancias, i, i+1)
			k.DispositivosIO[nombreIO] = iosMismoNombre //actualizo el map

			utils.LoggerConFormato("Una instancia de IO %s fue finalizada", nombreIO)
			return
		}
	}

	fmt.Printf("No se encontro para desconectar una instancia %s de IO pedida", nombreIO)
}

func (k *Kernel) ManejarIO(nombre_io string, cpu_ejecutando *CPU, duracion int) {
	defer RegistrarCPUaLibre(cpu_ejecutando)

	//mutex IOs
	mutex_DispositivosIO.Lock()
	defer mutex_DispositivosIO.Unlock()

	iosMismoNombre, existeIO := k.DispositivosIO[nombre_io]

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
			Pid:      cpu_ejecutando.Pid,
			TiempoIO: duracion,
		}
		iosMismoNombre.ColaEspera = append(iosMismoNombre.ColaEspera, nuevo_proc_esperando)
		return
	}
	// si hay io libre
	IO_seleccionada.PidOcupante = cpu_ejecutando.Pid
	//enviar a io
	IO_seleccionada.Libre = false
	utils.LoggerConFormato("## (%d) - Bloqueado por IO: %s", IO_seleccionada.PidOcupante, nombre_io)
	enviarProcesoAIO(IO_seleccionada, duracion)
}
