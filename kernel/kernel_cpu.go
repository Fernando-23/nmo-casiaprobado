package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

func (k *Kernel) LlegaNuevaCPU(w http.ResponseWriter, r *http.Request) { // Handshake

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (LlegaNuevaCPU) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensajeCPU := string(body_Bytes)

	slog.Debug("Llegó nueva cpu", "mensaje", mensajeCPU)

	if !k.registrarNuevaCPU(mensajeCPU) {
		http.Error(w, "No se pudo registar la CPU", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(RESPUESTA_OK))
}

func (k *Kernel) registrarNuevaCPU(mensajeCPU string) bool {
	fmt.Println(mensajeCPU)
	aux := strings.Split(mensajeCPU, " ") //ID IP PUERTO

	if len(aux) != 3 {
		fmt.Println("Formato invalido. Esperando: 'ID IP PUERTO'")
		return false
	}

	nueva_ID_CPU, err := strconv.Atoi(aux[0])

	if err != nil {
		fmt.Println("ID de CPU inválido")
		return false
	}

	ip := aux[1]
	puerto := aux[2]
	url := fmt.Sprintf("http://%s:%s/cpu", ip, puerto)

	mutex_CPUsConectadas.Lock()
	defer mutex_CPUsConectadas.Unlock()

	if _, existe := k.CPUsConectadas[nueva_ID_CPU]; existe {
		fmt.Println("Ya existe una CPU registrada con ese ID")
		return false
	}

	k.CPUsConectadas[nueva_ID_CPU] = crearCPU(nueva_ID_CPU, url)

	fmt.Printf("Se conecto una nueva CPU con ID %d en %s\n", nueva_ID_CPU, url)

	return true
}

func crearCPU(id int, url string) *CPU {
	nueva_cpu := &CPU{
		ID:            id,
		Url:           url,
		Pid:           -1,
		Pc:            0,
		ADesalojarPor: -1,
		Esta_libre:    true,
	}
	return nueva_cpu

}

//	FUNCIONES QUE ACTUALIZAN  ELEMENTOS DEL KERNEL

func (k *Kernel) ObtenerCPULibre() *CPU {

	for _, cpu := range k.CPUsConectadas {
		if cpu.Esta_libre && cpu.Pid == -1 {
			return cpu // La primera CPU que esta libre
		}
	}
	return nil // No hay CPU libre
}

// PLANIFICACION

func (k *Kernel) BuscarCPUPorID(id int) *CPU {
	cpu, existe := k.CPUsConectadas[id]
	if !existe {
		return nil
	}
	return cpu
}

func actualizarCPU(cpu *CPU, pid int, pc int, liberar bool) {
	cpu.Esta_libre = liberar
	cpu.Pid = pid
	cpu.Pc = pc
	cpu.ADesalojarPor = -1
}

func RegistrarCPUaLibre(cpu_a_liberar *CPU) {
	cpu_a_liberar.Esta_libre = true
	cpu_a_liberar.Pid = -1

}

func handleDispatch(pid int, pc int, url string) {

	fullURL := fmt.Sprintf("%s/dispatch", url)

	datos := fmt.Sprintf("%d %d", pid, pc)
	utils.EnviarStringSinEsperar("POST", fullURL, datos)
}

func reservarCPU(cpu *CPU, pid int) {
	cpu.ADesalojarPor = pid
}
