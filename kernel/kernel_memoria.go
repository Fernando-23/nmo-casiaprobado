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

func (k *Kernel) solicitudEliminarProceso(pid int) error {
	respuestaMemo, err := utils.FormatearUrlYEnviar(url_memo, "/EXIT_PROC", true, "%d", pid)

	if err != nil {
		slog.Error("Error - (SolicitudEliminarProceso) - Fallo al enviar solicitud",
			"pid", pid,
			"error", err,
		)
		return err
	}

	if respuestaMemo != RESPUESTA_OK {
		slog.Warn("Advertencia - (SolicitudEliminarProceso) - Respuesta inesperada",
			"pid", pid,
			"respuesta", respuestaMemo,
		)
		return fmt.Errorf("respuesta inesperada: %s", respuestaMemo)
	}

	return nil
}
func EnviarMemoryDump(pid int) bool {
	respuesta, err := utils.FormatearUrlYEnviar(url_memo, "/MEMORY_DUMP", true, "%d", pid)

	if err != nil {
		slog.Error("Error - (EnviarMemoryDump) - Fallo al enviar solicitud",
			"pid", pid,
			"error", err,
		)
		return false
	}

	if respuesta != RESPUESTA_OK {
		slog.Warn("Advertencia - (EnviarMemoryDump) - Respuesta inesperada",
			"pid", pid,
			"respuesta", respuesta,
		)
		return false
	}

	slog.Info("Memory Dump exitoso", "pid", pid)
	return true
}

func EnviarSuspension(pid int) {
	resp, err := utils.FormatearUrlYEnviar(url_memo, "/SUSPEND_PROC", true, "%d", pid)

	if err != nil {
		slog.Error("Error - (EnviarSuspension) - Fallo al enviar solicitud", "pid", pid, "error", err)
		return
	}

	if resp != RESPUESTA_OK {
		slog.Warn("Advertencia - (EnviarSuspension) - Respuesta inesperada", "pid", pid, "respuesta", resp)
		return
	}

	slog.Info("Proceso suspendido exitosamente", "pid", pid)
}

func EnviarDesuspension(pid int) (bool, error) {
	resp, err := utils.FormatearUrlYEnviar(url_memo, "/DE_SUSPEND_PROC", true, "%d", pid)

	if err != nil {
		slog.Error("Error - (EnviarDesuspension) - Fallo al enviar solicitud", "pid", pid, "error", err)
		return false, err
	}

	if resp != "OK" {
		slog.Warn("Advertencia - (EnviarDesuspension) - Respuesta inesperada", "pid", pid, "respuesta", resp)
		return false, nil
	}

	return true, nil
}

func (k *Kernel) MemoHayEspacio(pid int, tamanio int, archivoPseudo string) (bool, error) {

	resp, err := utils.FormatearUrlYEnviar(url_memo, "/hay_lugar", true, "%d %d %s", pid, tamanio, archivoPseudo)

	if err != nil {
		slog.Error("Error - (MemoHayEspacio) - Fallo al enviar solicitud", "pid", pid, "error", err)
		return false, err
	}

	if resp != "OK" {
		slog.Warn("Advertencia - (MemoHayEspacio) - Respuesta inesperada", "pid", pid, "respuesta", resp)
		return false, nil
	}

	return true, nil
}
