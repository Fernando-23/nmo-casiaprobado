package utils

import(
	"net/http"
	"encoding/json"
	"fmt"
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