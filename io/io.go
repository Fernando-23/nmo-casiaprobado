package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func main() {
	fmt.Println("Iniciando I/O..")

	args := os.Args

	if len(os.Args) < 3 { // ruta archivo-pseudo tamanio
		fmt.Println("Cantidad de argumentos incorrecta. Uso: ruta <archivo-pseudo> <tamanio>")
		os.Exit(1)
		return
	}

	nombre_io = args[1]
	instancia := args[2]
	ruta_config := fmt.Sprintf("io%s.json", instancia)
	config_io = &ConfigIO{}

	err := utils.IniciarConfiguracion(ruta_config, config_io)

	if err != nil {
		fmt.Printf("Error al iniciar config, error: %e", err)
		return
	}

	utils.ConfigurarLogger(nombre_io, config_io.Log_level)

	// Url por si hay que desconectar IO

	url_io = fmt.Sprintf("http://%s:%d", config_io.Ip_io, config_io.Puerto_io)

	//Url base de kernel
	url_kernel = fmt.Sprintf("http://%s:%d/kernel", config_io.Ip_kernel, config_io.Puerto_kernel)

	//Petición registrar io para kernel
	RegistrarIO(nombre_io)

	//Iniciando servidor para peticiones I/O - Kernel
	mux := http.NewServeMux()
	mux.HandleFunc("/io/hace_algo", AtenderPeticion)
	url_socket := fmt.Sprintf(":%d", config_io.Puerto_io)

	go func() {
		if err := http.ListenAndServe(url_socket, mux); err != nil {
			fmt.Println("Error al iniciar el servidor HTTP", err)
			os.Exit(1)
		}
	}()

	//canal peticiones sistema operativo
	senial_sistema_operativo := make(chan os.Signal, 1)

	fmt.Println("Modulo IO esperando seniales (SIGINT o SIGTERM)..")

	//canal para terminar

	ch_cancelar_io = make(chan struct{})
	signal.Notify(senial_sistema_operativo, syscall.SIGTERM, syscall.SIGINT)

	fmt.Println("Modulo IO esperando seniales (SIGINT o SIGTERM)..")

	senial := <-senial_sistema_operativo
	fmt.Println("Senial desconexion IO recibida:", senial)
	close(ch_cancelar_io)
	AvisarDesconexionIO()

	fmt.Printf("IO %s finalizada.", nombre_io)

}
