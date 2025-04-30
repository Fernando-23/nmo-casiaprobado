package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	cliente "github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils/Cliente"
)

func main() {
	fmt.Println("Iniciando CPU...")
	config_CPU = iniciarConfiguracionIO("config.json")
	args := os.Args
	nombre_cpu := args[1]

	cliente.EnviarMensaje(config_CPU.Ip_Memoria, config_CPU.Puerto_Memoria, "Conexion hecha con modulo CPU")
	handshakeKernel(nombre_cpu)

	var instruccion string

	cliente.ConfigurarLogger("cpu")

	var interrupcion bool = true

	// Ciclo de instruccion
	var datos_ciclo datosCiclo
	datos_ciclo, err := pedirDatosCiclo()
	for err != nil {
		datos_ciclo, err = pedirDatosCiclo()
	}

	pid_log := strconv.Itoa(datos_ciclo.Pid)
	pc_log := strconv.Itoa(datos_ciclo.Pc)

	for !interrupcion {

		instruccion = fetch(&datos_ciclo.Pc, &datos_ciclo.Pid)
		slog.Info("## PID: %s - FETCH - Program Counter: %s", pid_log, pc_log)
		// chequeamos si recibimos algo valido de memo
		for instruccion == "" {
			instruccion = fetch(&datos_ciclo.Pc, &datos_ciclo.Pid)
		}

		cod_op, operacion := decode(instruccion)
		execute(cod_op, operacion, &datos_ciclo.Pc, datos_ciclo.Pid)
		//interrupcion = checkInterrupt(&datos_ciclo.Pc, &datos_ciclo.Pid)

	}

}
