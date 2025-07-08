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

	err := utils.IniciarConfiguracion("io.json", config_IO)

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

	fmt.Println("Modulo IO esperando seniales (SIGINT o SIGTERM)..")

	//canal para terminar

	ch_cancelar_IO = make(chan struct{})
	signal.Notify(senial_sistema_operativo, syscall.SIGTERM, syscall.SIGINT)

	fmt.Println("Modulo IO esperando seniales (SIGINT o SIGTERM)..")

	senial := <-senial_sistema_operativo
	fmt.Println("Senial desconexion IO recibida:", senial)
	close(ch_cancelar_IO)
	AvisarDesconexionIO()

	fmt.Printf("IO %s finalizada.", nombre_io)

}
