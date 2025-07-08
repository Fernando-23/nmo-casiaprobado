package main

import (
	"fmt"
	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (k *Kernel) HandshakeMemoria() error {
	respuesta, err := utils.FormatearUrlYEnviar(url_memo, "/handshake", true, "holi memo que lindo estas hoy")
	if err != nil {
		return fmt.Errorf("memoria no responde: %w", err)
	}

	if respuesta != "OK" {
		return fmt.Errorf("respuesta inesperada de memoria: %s", respuesta)
	}

	return nil
}

func (k *Kernel) MemoHayEspacio(pid int, tamanio int, archivoPseudo string) (bool, error) {

	respuesta, err := utils.FormatearUrlYEnviar(url_memo, "/hay_lugar", true, "%d %d %s", pid, tamanio, archivoPseudo)
	if err != nil {
		slog.Error("Error - (MemoHayEspacio) - codificando mensaje", "error", err.Error())
		return false, err
	}

	if respuesta == "OK" {
		slog.Debug("PRUEBA - efectivamente, habia espacio")
		return true, nil
	}
	return false, nil

}

func (k *Kernel) solicitudEliminarProceso(pid int) (string, error) {

	respuestaMemo, err := utils.FormatearUrlYEnviar(url_memo, "/EXIT_PROC", true, "%d", pid)

	if err != nil || respuestaMemo != "OK" {
		slog.Error("Error - (SolicitudEliminarProceso) - Error codificando mensaje",
			"error", err.Error(),
		)
		return "", err
	}

	//Deberia responder "OK"
	return respuestaMemo, nil
}

func EnviarMemoryDump(pid int) bool {
	respuesta, err := utils.FormatearUrlYEnviar(url_memo, "/MEMORY_DUMP", true, "%d", pid)

	if err != nil || respuesta != "OK" {
		slog.Error("Error - (GestionarDUMP_MEMORY) - Dump fallido o respuesta inesperada",
			"pid", pid,
			"error", err,
			"respuesta", respuesta,
		)
		return false
	}
	return true
}

func EnviarSuspension(pid int) {
	resp, err := utils.FormatearUrlYEnviar(url_memo, "/SUSPEND_PROC", true, "%d", pid)

	if err != nil || resp != "OK" {
		slog.Error("Error - (EnviarSuspension) - Dump fallido o respuesta inesperada",
			"pid", pid,
			"error", err,
			"respuesta", resp,
		)
	}
}

func EnviarDesuspension(pid int) (bool, error) {
	resp, err := utils.FormatearUrlYEnviar(url_memo, "/DE_SUSPEND_PROC", true, "%d", pid)

	if err != nil || resp != "OK" {
		slog.Error("Error - (EnviarDesuspension) - Desuspirar fallido o respuesta inesperada",
			"pid", pid,
			"error", err,
			"respuesta", resp,
		)
		return false, err
	}
	return true, nil
}
