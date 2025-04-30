package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	//utils "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

type datosCiclo struct {
	Pid int
	Pc  int
}

type HandshakeRequest struct {
	NombreCPU string `json:"nombre_cpu"`
	PuertoCPU int    `json:"puerto_cpu`
	IpCPU     string `json:"ip_cpu"`
}

func pedirDatosCiclo() (datosCiclo, error) {

	url := fmt.Sprintf("http://%s:%d/pid", config_CPU.Ip_Kernel, config_CPU.Puerto_CPU)

	resp, err := http.Get(url)

	if err != nil {
		log.Printf("Error pidiendo datos de ciclo de inst. a ip:%s puerto:%d", config_CPU.Ip_Kernel, config_CPU.Puerto_Kernel)
		return datosCiclo{}, err
	}

	bodyByte, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return datosCiclo{}, err
	}
	var datos_a_retornar datosCiclo
	err = json.Unmarshal(bodyByte, &datos_a_retornar)

	if err != nil {
		log.Printf("Error, me enviaste cualquier cosa: %s", err.Error())
		return datosCiclo{}, err
	}

	return datos_a_retornar, nil
}

func handshakeKernel(nombre string) {
	peticion := HandshakeRequest{
		NombreCPU: nombre,
		PuertoCPU: config_CPU.Puerto_CPU,
		IpCPU:     config_CPU.Ip_CPU,
	}

	url := fmt.Sprintf("http://%s:%d/", config_CPU.Ip_CPU, config_CPU.Puerto_Kernel)
	body, err := json.Marshal(peticion)
	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Println("Error enviando datos", resp)
		return
	}

	log.Println("Hanshake realizado correctamente!")
}
func enviarSyscall(syscall_a_enviar string) {
	url := fmt.Sprintf("http://%s:%d/", config_CPU.Ip_CPU, config_CPU.Puerto_Kernel)
	body, err := json.Marshal(syscall_a_enviar)
	if err != nil {
		log.Printf("Error codificando mensaje: %s", err.Error())
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Println("Error enviando datos", resp)
		return
	}

	log.Println("Se envio una Syscall correctamente")
}

// func checkInterrupt(PC *int, PID *int) bool {
// 	var interrupcion bool = kernel.hayInterrupcion()
// 	if interrupcion {
// 		kernel.recibirInterrupcion(PC, PID)
// 	}
// 	return interrupcion
// }
