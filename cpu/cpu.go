package main

import (
	"fmt"

	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
)

func main() {
	fmt.Println("Iniciando CPU...")
	config_CPU = iniciarConfiguracionIO("config.json")
	cliente.EnviarMensaje(config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria, "Hola soy modulo CPU")
	cliente.EnviarMensaje(config_CPU.Ip_Kernel, config_CPU.Puerto_Kernel, "Hola soy modulo CPU")
}
