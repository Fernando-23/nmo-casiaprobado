package main

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func RegistrarIO(nombre string) {

	respuesta, err := utils.FormatearUrlYEnviar(url_kernel, "/registrar_io", true, "%s %s %d",
		nombre,
		config_io.Ip_io,
		config_io.Puerto_io,
	)

	if respuesta != "OK" || err != nil {
		slog.Error("Error - (RegistrarIO) - Respuesta Kernel",
			"respuesta", respuesta,
			"error", err,
		)
	}

	slog.Debug("Se registró la IO correctamente")
}

func AtenderPeticion(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (AtenderPeticion) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición desde kernel", "mensaje", mensaje)

	aux := strings.Split(mensaje, " ")

	if len(aux) != 2 {
		slog.Error("Error - (AtenderPeticion) - Cantidad erronea de argumentos", "error", err)
		http.Error(w, "Cantidad de argumentos inválida", http.StatusBadRequest)
		return
	}

	pid_recibido := aux[0]
	tiempo_recibido, err := strconv.Atoi(aux[1])

	if err != nil {
		http.Error(w, "Tiempo inválido", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	utils.LoggerConFormato("## PID: %s - Inicio de IO - Tiempo: %d", pid_recibido, tiempo_recibido)

	hay_proceso_io = true
	duracion_en_io = float64(tiempo_recibido)
	tiempo_en_io = time.Now()

	select {
	case <-time.After(time.Duration(tiempo_recibido) * time.Millisecond):
		utils.LoggerConFormato("Termino correctamente tiempo en io en AtenderPeticion")
		hay_proceso_io = false
		AvisarFinIO(pid_recibido)

	case <-ch_cancelar_io:
		utils.LoggerConFormato("IO desconectada en medio de ejecucion de AtenderPeticion")
	}
}

func AvisarFinIO(pid string) {

	respuesta, err := utils.FormatearUrlYEnviar(url_kernel, "/fin_io", true, "%s %s", pid, nombre_io)

	if respuesta != "OK" || err != nil {
		slog.Error("Error - (AvisarFinIO) - Respuesta Kernel",
			"respuesta", respuesta,
			"error", err,
		)
	}

	//==================== LOG OBLIGATORIO ====================
	utils.LoggerConFormato("## PID: %s - Fin de IO", pid)
	//=========================================================
}

func AvisarDesconexionIO() { //gracias que te aviso pa

	utils.FormatearUrlYEnviar(url_kernel, "/desconectar_io", true, "%s %s",
		nombre_io,
		url_io)

	utils.LoggerConFormato("Avisando desconexion IO: %s", nombre_io)
}
