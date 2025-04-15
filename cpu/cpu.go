package main

import (
	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
)

func main() {
	cliente.EnviarMensaje("127.0.0.1", 8002, "Hola soy modulo CPU")
	cliente.EnviarMensaje("127.0.0.1", 8001, "Hola soy modulo CPU")
}
