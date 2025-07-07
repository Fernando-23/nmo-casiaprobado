package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"slices"

	utils "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

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

	go func() {
		k.registrarNuevaIO(nombre, ip, puerto)
		utils.LoggerConFormato("Fin de registrarNuevaIO, nombre io: %s", nombre)
	}()

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
		utils.LoggerConFormato("Se agregó nueva instancia de IO '%s' conectada en %s\n", nombre, url)

	}
}

func (k *Kernel) llegaFinIO(w http.ResponseWriter, r *http.Request) {
	var mensajeIO string
	if err := json.NewDecoder(r.Body).Decode(&mensajeIO); err != nil {
		slog.Error("Error recibiendo la solicitud", "error", err)
		http.Error(w, "Error en el formato de la solicitud", http.StatusBadRequest)
		return
	}
	pid, nombre, err := decodificarMensajeFinIO(mensajeIO)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	go func() {
		k.liberarInstanciaIO(pid, nombre)
		utils.LoggerConFormato("Fin de ejecución de liberarInstanciaIO para PID %d (%s)", pid, nombre)
	}()
}

func (k *Kernel) liberarInstanciaIO(pid int, nombre string) {

	if !k.MoverDeEstadoPorPid(EstadoBlock, EstadoReady, pid, true) {
		if !k.MoverDeEstadoPorPid(EstadoBlockSuspended, EstadoReadySuspended, pid, true) {
			utils.LoggerConFormato("ERROR (liberarInstanciaIO) PID %d no estaba en BLOCK ni BLOCK_SUSP", pid)
			return //no estaba en ninguno de los estados posibles
		}
	}

	if !k.ActualizarIO(nombre, pid) {
		utils.LoggerConFormato("ERROR (liberarInstanciaIO) Falló la actualización de IO para %s y PID %d", nombre, pid)
		return
	}

	utils.LoggerConFormato("## (%d) finalizo IO y pasa a READY", pid)
}

func (k *Kernel) ActualizarIO(nombre_io string, pid_asociado_io int) bool {
	//mutex IOS
	mutex_DispositivosIO.Lock()
	defer mutex_DispositivosIO.Unlock()

	ios, existe := k.DispositivosIO[nombre_io]

	if !existe {
		utils.LoggerConFormato("ERROR (ActualizarIO) No existe dispositivo IO con nombre %s\n", nombre_io)
		return false
	}

	//Actualizar IO actual
	var io_usada *DispositivoIO
	for _, io := range ios.Instancias {
		if io.PidOcupante == pid_asociado_io {
			io_usada = io
			break
		}
	}

	if io_usada == nil {
		utils.LoggerConFormato(" ERROR (liberarInstanciaIO) No se encontró IO en uso con PID %d en %s\n", pid_asociado_io, nombre_io)
		return false
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
	return true
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

func enviarProcesoAIO(dispositivo *DispositivoIO, duracion int) {

	fullURL := fmt.Sprintf("%s/io/hace_algo", dispositivo.Url)
	datos := fmt.Sprintf("%d %d", dispositivo.PidOcupante, duracion)

	utils.EnviarStringSinEsperar("POST", fullURL, datos)

	utils.LoggerConFormato("## (%d) - Se envió proceso a IO en %s", dispositivo.PidOcupante, dispositivo.Url)
}
