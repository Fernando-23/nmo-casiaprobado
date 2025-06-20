package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func main() {
	fmt.Println("Iniciando I/O..")

	args := os.Args

	if len(os.Args) != 2 { // ruta archivo-pseudo tamanio
		fmt.Println("Cantidad de argumentos incorrecta. Uso: ruta <archivo-pseudo> <tamanio>")
		os.Exit(1)
	}

	nombre_io = args[1]
	config_IO = &ConfigIO{}

	err := utils.IniciarConfiguracion("config.json", config_IO)

	if err != nil {
		slog.Info("Error al iniciar config")
		return
	}

	// Url por si hay que desconectar IO
	url_io = fmt.Sprintf("http://%s:%s", config_IO.Ip_io, strconv.Itoa(config_IO.Puerto_io))

	//peticion registar io para kernel
	RegistrarIO(nombre_io)

	//Iniciando servidor para peticiones I/O - Kernel
	mux := http.NewServeMux()
	mux.HandleFunc("/io/hace_algo", AtenderPeticion)

	url_socket := fmt.Sprintf(":%d", config_IO.Puerto_io)

	go func() {
		if err := http.ListenAndServe(url_socket, mux); err != nil {
			fmt.Println("Error al iniciar el servidor HTTP", err)
			os.Exit(1)
		}
	}()

	//canal peticiones sistema operativo
	senial_sistema_operativo := make(chan os.Signal, 1)
	signal.Notify(senial_sistema_operativo, syscall.SIGTERM, syscall.SIGINT)

	//canal para terminar

	signal_ch_terminar_io := make(chan struct{})

	go func() {
		senial := <-senial_sistema_operativo
		fmt.Println("Senial desconexion IO recibida:", senial)

		AvisarDesconexionIO()

		signal_ch_terminar_io <- struct{}{}
	}()

	fmt.Println("Modulo IO esperando seniales (SIGINT o SIGTERM)..")

	<-signal_ch_terminar_io

	fmt.Printf("IO %s finalizada.", nombre_io)

}
