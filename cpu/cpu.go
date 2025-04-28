package main

import (
	"fmt"

	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
)

func main() {
	fmt.Println("Iniciando CPU...")
	config_CPU = iniciarConfiguracionIO("config.json")
	cliente.EnviarMensaje(config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria, "Conexion hecha con modulo CPU")
	cliente.EnviarMensaje(config_CPU.Ip_Kernel, config_CPU.Puerto_Kernel, "Conexion hecha con modulo CPU")

	var instruccion string

	cliente.ConfigurarLogger("cpu")

	PID, PC := pedirDatosCiclo(config_CPU.Ip_Kernel, config_CPU.Puerto_Kernel)

	var interrupcion bool = true

	//Ciclo de instruccion

	for !interrupcion {
		instruccion = fetch(&PC, &PID)
		cod_op, operacion := decode(instruccion)
		execute(cod_op, operacion, &PC)
		interrupcion = checkInterrupt(&PC, &PID)

	}

}
