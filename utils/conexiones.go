package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

func enviarSolicitudHTTP[T any](method string, url string, body interface{}, respuesta *T) error {
	var requestBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("error al serializar JSON: %w", err)
		}
		requestBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return fmt.Errorf("error creando la solicitud: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error realizando la solicitud: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(respuesta); err != nil {
		return fmt.Errorf("error")
	}
	return nil
}

func EnviarSolicitudHTTPString(method string, url string, body interface{}) (string, error) { //creo que lee mal por el Content-Type application/json del json
	var requestBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("error al serializar JSON: %w", err)
		}
		requestBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return "", fmt.Errorf("error creando la solicitud: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error realizando la solicitud: %w", err)
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error leyendo la respuesta: %w", err)
	}

	return string(resBody), nil
}

//Creo dos variantes de EnviarSolicitudHTTPString

// Esta es sincónica o sea se queda esperando el mensaje del server y te lo retorna
func EnviarStringConEspera(method string, url string, body string) (string, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("error creando la solicitud: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error realizando la solicitud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		resBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error servidor: %s", string(resBody))
	}

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error leyendo la respuesta: %w", err)
	}

	return string(resBody), nil
}

// Esta es asincónica o sea se envia el mensaje al server y se las toma (usando una go rutine que esa si espera)
func EnviarStringSinEsperar(method string, url string, body string) {
	go func() {
		req, err := http.NewRequest(method, url, strings.NewReader(body))

		if err != nil {
			fmt.Printf("ERROR (EnviarSinEsperar) Error creando solicitud HTTP: %v\n", err)
			return
		}
		req.Header.Set("Content-Type", "text/plain")

		client := &http.Client{}
		_, err = client.Do(req)
		if err != nil {
			fmt.Printf("ERROR (EnviarSinEsperar) enviando solicitud HTTP: %v\n", err)
		}
	}()
}

func IniciarConfiguracion[T any](ruta string, estructuraDeConfig *T) error {
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

func LoggerConFormato(format string, args ...interface{}) {
	slog.Info(fmt.Sprintf(format, args...))
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

func FormatearUrlYEnviar(urlBase string, urlAgregar string, esperar bool, peticion string, args ...any) (string, error) {
	fullURL := fmt.Sprintf("%s%s", urlBase, urlAgregar)
	fullPeticion := fmt.Sprintf(peticion, args...)
	if esperar {
		respuesta, err := EnviarStringConEspera("POST", fullURL, fullPeticion)
		return respuesta, err
	}

	EnviarStringSinEsperar("POST", fullURL, fullPeticion)
	return "", nil
}
