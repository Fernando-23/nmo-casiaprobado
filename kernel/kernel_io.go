package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"slices"

	utils "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (k *Kernel) llegaDesconexionIO(w http.ResponseWriter, r *http.Request) {

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		slog.Error("Error - (FinalizarIO) - Error al decodificar cuerpo", "error", err)
		return
	}

	mensajeIO := string(bodyBytes)
	nombre, url, err := decodificarMensajeDesconeccionIO(mensajeIO)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	go k.desconectarIO(nombre, url)
}

func (k *Kernel) desconectarIO(nombre, url string) {

	//mutex IOS
	mutex_DispositivosIO.Lock()
	defer mutex_DispositivosIO.Unlock()

	iosMismoNombre, ok := k.DispositivosIO[nombre]

	if !ok {
		slog.Error("Error - (FinalizarIO) - No existe el dispositivo IO", "nombre_dispositivo", nombre)
		return
	}

	for i, instancia := range iosMismoNombre.Instancias {
		if instancia.Url == url {
			pid := instancia.PidOcupante
			if !k.EliminarProcesoDeIO(pid) {
				slog.Error("Error - (desconectarIO) - Proceso ejecutando en IO no encotrado en colas de bloqueo al intentar eliminar",
					"pid", pid,
					"url", url,
					"dispositivo", nombre,
				)
				return
			}

			// Borro la instancia primero
			iosMismoNombre.Instancias = slices.Delete(iosMismoNombre.Instancias, i, i+1)

			if len(iosMismoNombre.Instancias) == 0 {
				// No quedan instancias, limpio cola de espera también
				for _, procEsperando := range iosMismoNombre.ColaEspera {
					if !k.EliminarProcesoDeIO(procEsperando.Pid) {
						slog.Error("Error - (desconectarIO) - Proceso en cola de espera por IO no encontrado en colas de bloqueo al intentar eliminar",
							"pid", procEsperando.Pid,
							"url", url,
							"dispositivo", nombre,
						)
						return
					}
				}
				delete(k.DispositivosIO, nombre)
				utils.LoggerConFormato("El dispositivo %s fue eliminado porque ya no tiene instancias", nombre)
				return
			}
			// Todavía quedan instancias, actualizo el map
			k.DispositivosIO[nombre] = iosMismoNombre
			utils.LoggerConFormato("Una instancia de IO %s fue finalizada", nombre)
			return
		}
	}

	slog.Error("Error - (desconectarIO) - No se encontro la instancia de IO pedida",
		"nombre", nombre,
		"url", url,
	)
}

func (k *Kernel) EliminarProcesoDeIO(pid int) bool {
	var proceso *PCB

	proceso = k.QuitarYObtenerPCB(EstadoBlock, pid, true)
	if proceso != nil {
		k.EliminarProceso(proceso, true)
		return true
	}

	proceso = k.QuitarYObtenerPCB(EstadoBlockSuspended, pid, true)
	if proceso != nil {
		k.EliminarProceso(proceso, false)
		return true
	}

	return false
}

func (k *Kernel) llegaNuevaIO(w http.ResponseWriter, r *http.Request) { // Handshake
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		slog.Error("Error - (llegaNuevaIO) - Error al decodificar cuerpo", "error", err)
		return
	}

	mensajeIO := string(bodyBytes)
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
	if actual := k.DispositivosIO[nombre]; actual == nil {
		// Agrego y sincronizo el nuevo dispositivo io
		k.DispositivosIO[nombre] = &InstanciasPorDispositivo{
			Instancias: []*DispositivoIO{nuevaInstancia},
			ColaEspera: []*ProcesoEsperandoIO{},
		}
		slog.Debug("Se agregó nuevo dispositivo de IO",
			"nombre", nombre,
			"url", url,
		)
	} else {
		// Sino, actualizo los valores en esa posicion
		k.DispositivosIO[nombre].Instancias = append(k.DispositivosIO[nombre].Instancias, nuevaInstancia)
		slog.Debug("Se agregó nueva instancia de IO",
			"nombre", nombre,
			"url", url,
		)

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
		if k.MoverDeEstadoPorPid(EstadoBlockSuspended, EstadoReadySuspended, pid, true) {
			//hola
			return
		}
		slog.Error("Error -(liberarInstanciaIO) - Pid no estaba en BLOCK ni BLOCK_SUSP",
			"pid_io", pid,
		)
		return //no estaba en ninguno de los estados posibles
	}

	if err := k.ActualizarIO(nombre, pid); err != nil {
		slog.Error("Error - (liberarInstanciaIO) - Falló actualización de IO", "error", err)
		return
	}

	utils.LoggerConFormato("## (%d) finalizo IO y pasa a READY", pid)
}

func (k *Kernel) ActualizarIO(nombreIO string, pid int) error {
	//mutex IOS
	mutex_DispositivosIO.Lock()
	defer mutex_DispositivosIO.Unlock()

	ios, existe := k.DispositivosIO[nombreIO]

	if !existe {
		return fmt.Errorf("no existe dispositivo IO %q", nombreIO)
	}

	//Actualizar IO actual
	var ioUsada *DispositivoIO
	for _, io := range ios.Instancias {
		if io.PidOcupante == pid {
			ioUsada = io
			break
		}
	}

	if ioUsada == nil {
		return fmt.Errorf("no se encontró instancia con pid %d", pid)
	}
	ioUsada.Actualizar(-1, true)

	if len(ios.ColaEspera) != 0 {
		//desencolar el 1ro
		proceso_sgte := ios.ColaEspera[0]
		ios.ColaEspera = ios.ColaEspera[1:]

		ioUsada.Actualizar(proceso_sgte.Pid, false)

		// mandarlo a io
		enviarProcesoAIO(ioUsada, proceso_sgte.TiempoIO)
	}
	return nil
}

func (d *DispositivoIO) Actualizar(pid int, libre bool) {
	d.PidOcupante = pid
	d.Libre = libre
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
