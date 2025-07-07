package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// ===========================
// Funciones de Configuración
// ===========================

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

// ===========================
// Funciones del Logger
// ===========================

func ConfigurarLogger(nombre string, nivel string) error {
	logFile, err := os.OpenFile(nombre+".log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo de log: %w", err)
	}

	mw := io.MultiWriter(os.Stdout, logFile)

	var level slog.Level
	switch strings.ToLower(nivel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	default:
		fmt.Println("[WAR][Configurar Logger] se ingresó un logLevel rarito, se usará INFO por defecto")
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(mw, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler).With("modulo", nombre)
	slog.SetDefault(logger)

	return nil
}

// ===========================
// Decodificadores de mensajes
// ===========================

func decodificarMensajeFinInterrupcion(mensaje string) (idCPU, pid, pc int, err error) {
	aux := strings.Split(mensaje, " ")
	if len(aux) != 3 {
		return 0, 0, 0, fmt.Errorf("esperando formato 'ID PID PC'")
	}
	idCPU, err1 := strconv.Atoi(aux[0])
	pid, err2 := strconv.Atoi(aux[1])
	pc, err3 := strconv.Atoi(aux[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, fmt.Errorf("valores inválidos: %v %v %v", err1, err2, err3)
	}
	return idCPU, pid, pc, nil
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

func decodificarMensajeFinIO(mensaje string) (pid int, nombre string, err error) {
	partes := strings.Split(mensaje, " ")
	if len(partes) < 2 {
		return 0, "", fmt.Errorf("formato inválido, se espera 'PID NOMBRE_IO'")
	}

	pid, err = strconv.Atoi(partes[0])
	if err != nil {
		return 0, "", fmt.Errorf("PID inválido: %v", err)
	}

	nombre = partes[1]
	if nombre == "" {
		return 0, "", fmt.Errorf("nombre de IO vacío")
	}

	return pid, nombre, nil
}

func decodificarMensajeDesconeccionIO(mensaje string) (nombre, url string, tiempo int, err error) {
	partes := strings.Split(mensaje, " ") // Esperado: "NOMBRE_IO URL_IO TIEMPO_IO"

	if len(partes) < 3 {
		return "", "", 0, fmt.Errorf("formato inválido, se espera 'NOMBRE_IO URL_IO TIEMPO_IO'")
	}

	nombre = partes[0]
	if nombre == "" {
		return "", "", 0, fmt.Errorf("nombre de IO vacío")
	}
	url = partes[1]
	tiempo, err = strconv.Atoi(partes[2])

	if err != nil {
		return "", "", 0, fmt.Errorf("tiempo inválido: %v", err)
	}

	return nombre, url, tiempo, nil

}
