package main

import (
	"fmt"

	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
)

func main() {
	fmt.Println("Iniciando I/O..")
	config_IO = iniciarConfiguracionIO("config.json")

	cliente.EnviarMensaje(config_IO.Ip_kernel, config_IO.Puerto_kernel, "Hola soy modulo IO")

}
