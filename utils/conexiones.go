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

func EnviarSolicitudHTTPString(method string, url string, body interface{}) (string, error) {
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
