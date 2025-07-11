package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Nombre-muy-original/utils"
)

// =================================================================================
// =================================================================================
// Hola soy liam, lo comentado es porque ya no se usa y me dio cosa borrarlo mirenlo
// =================================================================================
// =================================================================================

func CargarArchivoPseudocodigo(path string) ([]string, error) {
	path_completo := "/home/utnso/pruebas/" + path
	//path_completo := "/home/liam/Data/Ftd/ISI/Proyectos/tp-2025-1c-Nombre-muy-original/pruebas/" + path

	archivo, err := os.Open(path_completo)

	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir archivo %s: %w", path_completo, err)
	}
	defer archivo.Close()

	var instrucciones []string
	scanner := bufio.NewScanner(archivo)

	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		instrucciones = append(instrucciones, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error leyendo archivo %s: %w", path_completo, err)
	}

	return instrucciones, nil

}

func (memo *Memo) Fetch(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (Fetch) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llegó petición fetch", "mensaje", mensaje)

	aux := strings.Split(mensaje, " ")

	if len(aux) < 2 {
		slog.Error("Error - (Fetch) - Formato invalido", "mensaje", mensaje)
		http.Error(w, "Formato invalido", http.StatusBadRequest)
		return
	}

	pid, err := strconv.Atoi(aux[0])
	if err != nil {
		slog.Error("Error - (Fetch) - Transformando pid", "pid", pid)
		http.Error(w, "PID inválido", http.StatusBadRequest)
		return
	}

	pc, err := strconv.Atoi(aux[1])
	if err != nil {
		slog.Error("Error - (Fetch) - Transformando pc", "pc", pc)
		http.Error(w, "PC inválido", http.StatusBadRequest)
		return
	}

	mutex_memoriaSistema.Lock()
	elemento_en_memo_sistema, ok := memo.memoria_sistema[pid]
	mutex_memoriaSistema.Unlock()

	if !ok || len(elemento_en_memo_sistema) == 0 {
		slog.Debug("No hay mas instrucciones")
		w.Write([]byte("TODO MAL")) //la ultima deberia ser un exit
		return
	}

	//para pruebas nomas
	//====================================================
	//====================================================
	//---------------ESTO LO VAMOS A BORRAR---------------
	//----https://www.youtube.com/watch?v=S94C7rR429s-----
	//====================================================
	//====================================================
	// for _, linea_a_leer := range elemento_en_memo_sistema {
	// 	slog.Debug(linea_a_leer)
	// }

	instruccion := elemento_en_memo_sistema[pc]

	slog.Debug("Instruccion a enviar",
		"instruccion", instruccion,
	)

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Cant_instr_solicitadas)
	mutex_metricas.Unlock()
	//==================== LOG OBLIGATORIO ====================
	utils.LoggerConFormato("## PID: %d - Obtener instrucción: %d - Instrucción: %s", pid, pc, instruccion)
	//=========================================================

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(instruccion))
}

func (memo *Memo) VerificarHayLugar(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (VerificarHayLugar) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Debug - (VerificarHayLugar) - Llego peticion verificar hay lugar", "mensaje", mensaje)

	aux := strings.Split(mensaje, " ")

	if len(aux) < 3 {
		http.Error(w, "Formato invalido: se requieren pid, tamanio y arch_pseudo", http.StatusBadRequest)
		return
	}

	pid, err := strconv.Atoi(aux[0])
	if err != nil {
		http.Error(w, "PID invalido", http.StatusBadRequest)
		return
	}

	tamanio, err := strconv.Atoi(aux[1])
	if err != nil {
		http.Error(w, "Tamanio invalido", http.StatusBadRequest)
		return
	}

	arch_pseudo := aux[2]

	mutex_tamanioMemoActual.Lock()
	hayEspacio := HayEspacio(tamanio)
	mutex_tamanioMemoActual.Unlock()

	if !hayEspacio {
		slog.Debug("Debug - (VerificarHayLugar) - No hay espacio suficiente para crear el proceso pedido por kernel", "pid", pid)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_OK"))
		return
	}

	// Intentar crear el proceso
	mutex_lprocs.Lock()
	err = memo.CrearNuevoProceso(pid, tamanio)
	if err != nil {
		slog.Error("Error - (VerificarHayLugar) - Error creando proceso", "pid", pid, "error", err)
		http.Error(w, "Error creando el proceso", http.StatusInternalServerError)
		mutex_lprocs.Unlock()
		return
	}
	mutex_lprocs.Unlock()

	// Cargar archivo pseudo
	nuevoElemento, err := CargarArchivoPseudocodigo(arch_pseudo)
	if err != nil {
		slog.Error("Error - (VerificarHayLugar) - Error al cargar archivo pseudocodigo", "pid", pid, "error", err)
		http.Error(w, "Error al cargar pseudocodigo", http.StatusInternalServerError)
		return
	}

	mutex_memoriaSistema.Lock()
	memo.memoria_sistema[pid] = nuevoElemento
	mutex_memoriaSistema.Unlock()

	// Actualizar memoria usada (la variable global)
	mutex_tamanioMemoActual.Lock()
	gb_tam_memo_actual -= tamanio
	mutex_tamanioMemoActual.Unlock()

	// Inicializar metricas
	mutex_metricas.Lock()
	memo.InicializarMetricasPor(pid)
	mutex_metricas.Unlock()

	// Digo que esta bien porque ya verifique todo
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

}

func HayEspacio(tamanio int) bool {
	return gb_tam_memo_actual >= tamanio
}
func HayFramesLibresPara(cant_frames_pedidos int) bool {
	return gb_frames_disponibles > cant_frames_pedidos
}

func (memo *Memo) CrearNuevoProceso(pid int, tamanio int) error {

	proc := &Proceso{
		Pid:           pid,
		TablaPagsRaiz: memo.crearTablasParaProceso(pid, tamanio),
		Tamanio:       tamanio,
	}

	memo.Procesos[pid] = proc

	err := memo.AsignarFramesAProceso(pid)
	if err != nil {
		return err
	}

	utils.LoggerConFormato("PID: %d - Proceso Creado - Tamaño: %d", pid, tamanio)
	return nil
}

func (memo *Memo) crearTablasParaProceso(pid int, tamanioProceso int) *TablaDePaginas {
	numPaginas := memo.LaCuentitaMaestro(tamanioProceso)

	entradaRaiz := &TablaDePaginas{
		Entradas: make([]*EntradaTablaDePaginas, memo.Config.EntradasPorNivel),
	}

	for pagina := 0; pagina < numPaginas; pagina++ {
		indices := memo.indicesParaPagina(pagina)

		actual := entradaRaiz
		for nivel := 0; nivel < memo.Config.Cant_niveles-1; nivel++ {
			idx := indices[nivel]
			if actual.Entradas[idx] == nil {
				actual.Entradas[idx] = &EntradaTablaDePaginas{
					SiguienteNivel: &TablaDePaginas{
						Entradas: make([]*EntradaTablaDePaginas, memo.Config.EntradasPorNivel),
					},
				}
			}
			actual = actual.Entradas[idx].SiguienteNivel
		}

		// Último nivel creo entrada vacia (sin frame asignado)
		ultimoIdx := indices[len(indices)-1]
		if actual.Entradas[ultimoIdx] == nil {
			actual.Entradas[ultimoIdx] = &EntradaTablaDePaginas{}
		}
	}

	return entradaRaiz
}

// Puede volar
func (memo *Memo) buscarFrameLibre() int {
	for i, frame := range memo.Frames {
		if !frame.Usado {
			return i
		}
	}
	return -1 // No hay frames disponibles
}

func (memo *Memo) indicesParaPagina(pagina int) []int {
	indices := make([]int, memo.Config.Cant_niveles)
	divisor := 1
	for i := memo.Config.Cant_niveles - 1; i >= 0; i-- {
		indices[i] = (pagina / divisor) % memo.Config.EntradasPorNivel
		divisor *= memo.Config.EntradasPorNivel
	}
	return indices
}

func (memo *Memo) traducir(pid int, direccionLogica int) (int, error) {
	proc := memo.Procesos[pid]
	pagina := direccionLogica / memo.Config.Tamanio_pag
	offset := direccionLogica % memo.Config.Tamanio_pag

	indices := memo.indicesParaPagina(pagina)
	actual := proc.TablaPagsRaiz
	for nivel := 0; nivel < memo.Config.Cant_niveles-1; nivel++ {
		entry := actual.Entradas[indices[nivel]]
		if entry == nil || entry.SiguienteNivel == nil {
			return -1, fmt.Errorf("page fault")
		}
		actual = entry.SiguienteNivel
	}

	finalEntry := actual.Entradas[indices[len(indices)-1]]
	if finalEntry == nil || finalEntry.NumeroDeFrame == nil {
		return -1, fmt.Errorf("page fault")
	}

	frameBase := (*finalEntry.NumeroDeFrame) * memo.Config.Tamanio_pag
	return frameBase + offset, nil
}

func (memo *Memo) LiberarFramesDeProceso(pid int) {
	mutex_framesDisponibles.Lock()
	defer mutex_framesDisponibles.Unlock()

	for _, frame := range memo.Frames {
		if frame != nil && frame.Usado && frame.PidOcupante == pid {
			frame.Usado = false
			frame.PidOcupante = -1
			frame.NumeroPagina = -1
		}
	}
}

func (memo *Memo) InicializarMetricasPor(pid int) {
	memo.metricas[pid] = make([]int, cant_metricas)
}

func (memo *Memo) Hanshake(w http.ResponseWriter, r *http.Request) {
	var string_modulo string
	body_bytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Error leyendo la solicitud:", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	string_modulo = string(body_bytes)

	memo.ResponderHandshakeA(string_modulo, w)

}

func (memo *Memo) ResponderHandshakeA(modulo string, w http.ResponseWriter) {
	switch modulo {
	case "CPU":
		respuesta := fmt.Sprintf("%d %d %d", memo.Config.Cant_niveles, memo.Config.EntradasPorNivel, memo.Config.Tamanio_pag)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(respuesta))

	case "Kernel":
		w.Write([]byte("OK"))
	default:
		w.Write([]byte("NO_OK"))
	}

}

func (memo *Memo) buscarFrameLibrePara(pid int, pageNum int) int {
	for i, frame := range memo.Frames {
		if !frame.Usado {
			memo.Frames[i] = &Frame{
				Usado:        true,
				PidOcupante:  pid,
				NumeroPagina: pageNum,
			}
			return i
		}
	}
	return -1
}

func (memo *Memo) AsignarFramesAProceso(pid int) error {
	mutex_framesDisponibles.Lock()
	defer mutex_framesDisponibles.Unlock()

	proc, ok := memo.Procesos[pid]
	if !ok {
		return fmt.Errorf("Proceso %d no existe", pid)
	}

	numPaginas := memo.LaCuentitaMaestro(proc.Tamanio)

	for pagina := 0; pagina < numPaginas; pagina++ {
		indices := memo.indicesParaPagina(pagina)

		actual := proc.TablaPagsRaiz
		for nivel := 0; nivel < memo.Config.Cant_niveles-1; nivel++ {
			idx := indices[nivel]
			if actual.Entradas[idx] == nil || actual.Entradas[idx].SiguienteNivel == nil {
				return fmt.Errorf("estructura inconsistente para PID %d en nivel %d", pid, nivel)
			}
			actual = actual.Entradas[idx].SiguienteNivel
		}

		// Último nivel
		finalIdx := indices[len(indices)-1]
		entrada := actual.Entradas[finalIdx]

		if entrada.NumeroDeFrame == nil {
			frameIdx := memo.buscarFrameLibrePara(pid, pagina)
			if frameIdx == -1 {
				return fmt.Errorf("sin frames libres para la pagina %d del proceso %d", pagina, pid)
			}
			entrada.NumeroDeFrame = &frameIdx
		}
	}

	slog.Debug("Debug - (AsignarFramesAProceso) -  Se asigno correctamente frames al proceso", "pid", pid)
	return nil
}

// Traeme la dolorosa, la juguetona pa
func (memo *Memo) LaCuentitaMaestro(tamanio_proc int) int {
	la_dolorosa := tamanio_proc / memo.Config.Tamanio_pag
	if (tamanio_proc % memo.Config.Tamanio_pag) != 0 {
		return la_dolorosa + 1
	} else if tamanio_proc == 0 {
		return 1
	}

	return la_dolorosa
}

// imprime las lablas de paginas un proceso
func (memo *Memo) ImprimirTablasProceso(pid int) {
	proc, ok := memo.Procesos[pid]
	if !ok {
		slog.Warn("Cuidadito - (ImprimirTablasProceso) - Proceso NO encontrado", "pid", pid)
		return
	}

	fmt.Printf("Proceso %d - Arbol de tablas:\n\n", pid)
	memo.recorrerTabla(proc.TablaPagsRaiz, []int{}, 0)
}

func (memo *Memo) recorrerTabla(tabla *TablaDePaginas, path []int, nivel int) {
	for i, entrada := range tabla.Entradas {
		if entrada == nil {
			continue
		}

		nuevoPath := append([]int{}, path...)
		nuevoPath = append(nuevoPath, i)

		if entrada.SiguienteNivel != nil {
			indent := strings.Repeat("  ", nivel)
			fmt.Printf("%sNivel %d, Entrada %d \n", indent, nivel, i)
			memo.recorrerTabla(entrada.SiguienteNivel, nuevoPath, nivel+1)
		} else if entrada.NumeroDeFrame != nil {
			indent := strings.Repeat("  ", nivel)
			fmt.Printf("%sNivel %d, Entrada %d → Frame %d (Página virtual %s)\n", indent, nivel, i, *entrada.NumeroDeFrame, formatPath(nuevoPath))
		}
	}
}

// Imprime lindo
func formatPath(indices []int) string {
	strs := make([]string, len(indices))
	for i, val := range indices {
		strs[i] = strconv.Itoa(val)
	}
	return strings.Join(strs, ".")
}

// Acceso a espacio de usuario
func (memo *Memo) buscarEnTablaAsociadoAProceso(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (buscarEnTablaAsociadoAProceso) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Debug - (buscarEnTablaAsociadoAProceso) - Llegó petición buscar en tabla asociado a proceso", "pid lv_actual entrada", mensaje)

	aux := strings.Split(mensaje, " ")

	if len(aux) < 3 {
		slog.Error("Error - (buscarEnTablaAsociadoAProceso) - Faltan argumentos")
		http.Error(w, "Faltan argumentos", http.StatusBadRequest)
		return
	}

	pid, err1 := strconv.Atoi(aux[0])
	nivelSolicitado, err2 := strconv.Atoi(aux[1])
	entrada, err3 := strconv.Atoi(aux[2])

	if err1 != nil || err2 != nil || err3 != nil {
		slog.Error("Error - (buscarEnTablaAsociadoAProceso) - Argumentos inválidos")
		http.Error(w, "Argumentos inválidos", http.StatusBadRequest)
		return
	}

	mutex_lprocs.Lock()
	defer mutex_lprocs.Unlock()
	proc, ok := memo.Procesos[pid]

	if !ok || proc == nil {
		slog.Error("Error - (buscarEnTablaAsociadoAProceso) - NO se encontro el proceos", "pid", pid)
		http.Error(w, "Proceso no encontrado", http.StatusNotFound)
		return
	}

	tablaActual := proc.TablaPagsRaiz
	if tablaActual == nil {
		slog.Error("Error - (buscarEnTablaAsociadoAProceso) - NO se inicializo la tabla de paginas", "pid", pid)
		http.Error(w, "Tabla de páginas no inicializada", http.StatusInternalServerError)
		return
	}

	// Navegar desde nivel 1 hasta nivelSolicitado - 1
	for nivel := 1; nivel < nivelSolicitado; nivel++ {
		if tablaActual.Entradas == nil || entrada >= len(tablaActual.Entradas) {
			slog.Error("Error - (buscarEnTablaAsociadoAProceso) - Entrada invalida de Nivel", "nivel", nivel, "pid", pid)
			http.Error(w, "Entrada invalida en nivel", http.StatusBadRequest)
			return
		}
		entradaTabla := tablaActual.Entradas[entrada]
		if entradaTabla == nil || entradaTabla.SiguienteNivel == nil {
			slog.Error("Error - (buscarEnTablaAsociadoAProceso) - NO hay siguiente nivel", "nivel", nivel, "pid", pid)
			http.Error(w, "Siguiente nivel no encontrado", http.StatusBadRequest)
			return
		}
		tablaActual = entradaTabla.SiguienteNivel
	}

	// Si no es último nivel, decir que siga consultando
	if nivelSolicitado != memo.Config.Cant_niveles {
		memo.HacerRetardo()
		mutex_metricas.Lock()
		memo.IncrementarMetrica(pid, Accesos_a_tpags)
		mutex_metricas.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("SEGUI"))
		return
	}

	// Último nivel: obtener frame
	if entrada < 0 || entrada >= len(tablaActual.Entradas) {
		http.Error(w, "Error - (buscarEnTablaAsociadoAsociadoAProceso) - Entrada invalida en ultimo nivel", http.StatusBadRequest)
		return
	}

	entradaFinal := tablaActual.Entradas[entrada]
	if entradaFinal == nil || entradaFinal.NumeroDeFrame == nil {
		http.Error(w, "Error - (buscarEnTablaAsociadoAsociadoAProceso) - Frame no asignado", http.StatusNotFound)
		return
	}

	frameStr := strconv.Itoa(*entradaFinal.NumeroDeFrame)

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Accesos_a_tpags)
	mutex_metricas.Unlock()

	memo.HacerRetardo()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(frameStr))
}

func (memo *Memo) HacerRetardo() {
	time.Sleep(time.Duration(memo.Config.Delay_memoria) * time.Millisecond)
}

func (memo *Memo) LeerEnMemoria(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (LeerEnMemoria) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Debug - (LeerEnMemoria) - Llego peticion leer en memoria", "mensaje", mensaje)

	aux := strings.Split(mensaje, " ")

	if len(aux) < 4 {
		http.Error(w, "Error - (LeerEnMemoria) Faltan argumentos (pid frame offset tamanio)", http.StatusBadRequest)
		return
	}

	pid, err1 := strconv.Atoi(aux[0])
	frameIdx, err2 := strconv.Atoi(aux[1])
	offset, err3 := strconv.Atoi(aux[2])
	tamanioLeer, err4 := strconv.Atoi(aux[3])

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		slog.Error("Error - (LeerEnMemoria) - Argumentos invalidos")
		http.Error(w, "Error - (LeerEnMemoria) - Argumentos invalidos", http.StatusBadRequest)
		return
	}
	mutex_framesDisponibles.Lock()
	defer mutex_framesDisponibles.Unlock()
	// Validar que el frame exista y este asignado al proceso
	if frameIdx < 0 || frameIdx >= len(memo.Frames) {
		http.Error(w, "Error - (LeerEnMemoria) - Frame fuera de rango", http.StatusBadRequest)
		return
	}

	frame := memo.Frames[frameIdx]

	if !frame.Usado || frame.PidOcupante != pid { //quedo muy clean
		http.Error(w, "Error - (LeerEnMemoria) - Frame no asignado a ese proceso", http.StatusForbidden)
		return
	}

	// Validar offset y tamaño
	tamPag := memo.Config.Tamanio_pag
	if offset < 0 || offset >= tamPag {
		http.Error(w, "Error - (LeerEnMemoria) - Offset fuera de rango", http.StatusBadRequest)
		return
	}

	if tamanioLeer <= 0 || offset+tamanioLeer > tamPag {
		http.Error(w, "Error - (LeerEnMemoria) - Tamanio a leer fuera de rango", http.StatusBadRequest)
		return
	}

	// Calcular dirección física en memoria principal
	direccionFisica := frameIdx*tamPag + offset

	mutex_memoriaPrincipal.Lock()
	defer mutex_memoriaPrincipal.Unlock()

	if direccionFisica+tamanioLeer > len(memo.memoria_principal) {
		http.Error(w, "Error - (LeerEnMemoria) - Lectura fuera del rango de memoria fisica", http.StatusBadRequest)
		return
	}

	// Leer bytes de memoria fisica
	datos := memo.memoria_principal[direccionFisica : direccionFisica+tamanioLeer]

	w.WriteHeader(http.StatusOK)
	w.Write(datos)

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Cant_read)
	mutex_metricas.Unlock()

	utils.LoggerConFormato("## PID: %d - Lectura - Dir. Fisica: [ %d |  %d  ] - Tamaño: %d", pid, frameIdx, offset, tamanioLeer)
}

func (memo *Memo) EscribirEnMemoria(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (EscribirEnMemoria) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Debug - (EscribirEnMemoria) -Llego peticion escribir en memoria", "mensaje", mensaje)

	aux := strings.Split(mensaje, " ")

	if len(aux) < 4 {
		http.Error(w, "Faltan argumentos (pid frame offset datos)", http.StatusBadRequest)
		return
	}

	pid, err1 := strconv.Atoi(aux[0])
	frameIdx, err2 := strconv.Atoi(aux[1])
	offset, err3 := strconv.Atoi(aux[2])
	datosStr := aux[3] // El resto es el dato a escribir

	if err1 != nil || err2 != nil || err3 != nil {
		slog.Error("Error - (EscribirEnMemoria) - Conversiones a int")
		return
	}

	datos := []byte(datosStr)
	tamanioEscritura := len(datos)
	mutex_framesDisponibles.Lock()
	defer mutex_framesDisponibles.Unlock()

	if frameIdx < 0 || frameIdx >= len(memo.Frames) {
		http.Error(w, "Frame fuera de rango", http.StatusBadRequest)
		return
	}

	frame := memo.Frames[frameIdx]

	if !frame.Usado || frame.PidOcupante != pid {
		http.Error(w, "Frame no asignado a ese proceso", http.StatusForbidden)
		return
	}

	tamPag := memo.Config.Tamanio_pag
	if offset < 0 || offset >= tamPag {
		http.Error(w, "Offset fuera de rango", http.StatusBadRequest)
		return
	}
	if offset+tamanioEscritura > tamPag {
		http.Error(w, "Datos exceden tamanio de pagina", http.StatusBadRequest)
		return
	}

	direccionFisica := frameIdx*tamPag + offset

	mutex_memoriaPrincipal.Lock()
	defer mutex_memoriaPrincipal.Unlock()

	if direccionFisica+tamanioEscritura > len(memo.memoria_principal) {
		http.Error(w, "Escritura fuera del rango de memoria fisica", http.StatusBadRequest)
		return
	}

	// Copiar datos a memoria física
	copy(memo.memoria_principal[direccionFisica:direccionFisica+tamanioEscritura], datos)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	mutex_metricas.Lock()
	memo.IncrementarMetrica(pid, Cant_write)
	mutex_metricas.Unlock()

	utils.LoggerConFormato("## PID: %d - Escritura - Dir. Fisica: [ %d |  %d  ] - Tamaño: %d ", pid, frameIdx, offset, tamanioEscritura)
}

func (memo *Memo) DumpMemory(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (DumpMemory) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Debug - (DumpMemory) - Llego peticion dump memory", "mensaje", mensaje)

	pid, err := strconv.Atoi(mensaje)
	if err != nil {
		slog.Error("Error - (DumpMemory) - Conversion PID invalida", "mensaje", mensaje)
		http.Error(w, "PID invalido", http.StatusBadRequest)
		return
	}

	mutex_lprocs.Lock()
	defer mutex_lprocs.Unlock()
	proc, ok := memo.Procesos[pid]
	if !ok {
		http.Error(w, "Proceso no encontrado", http.StatusNotFound)
		return
	}
	tamanio := proc.Tamanio

	timestamp := time.Now().Format(time.RFC3339) //me lo ensenio mi amigo luchin guita facil
	nombre := fmt.Sprintf("%s%d-%s.dmp", memo.Config.Path_dump, pid, timestamp)

	file, err := os.Create(nombre)
	if err != nil {
		slog.Error("Error - (DumpMemory) - Error creando archivo dump", "path", nombre, "error", err)
		http.Error(w, "No se pudo crear archivo dump", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	err = file.Truncate(int64(tamanio))

	if err != nil {
		slog.Error("Error - (DumpMemory) - Error truncando archivo dump", "error", err)
		http.Error(w, "No se pudo preparar archivo", http.StatusInternalServerError)
		return
	}

	mutex_memoriaPrincipal.Lock()
	defer mutex_memoriaPrincipal.Unlock()

	mutex_framesDisponibles.Lock()
	defer mutex_framesDisponibles.Unlock()
	tamPag := memo.Config.Tamanio_pag
	for idx, frame := range memo.Frames {
		if frame.Usado && frame.PidOcupante == pid {
			inicio := idx * tamPag
			fin := inicio + tamPag
			if fin > len(memo.memoria_principal) {
				slog.Warn("Cuidadito - (DumpMemory) - Dump truncado: intento de leer mas alla del limite de memoria")
				continue
			}
			_, err := file.Write(memo.memoria_principal[inicio:fin])
			if err != nil {
				slog.Error("Error - (DumpMemory) - Error escribiendo en archivo dump", "frame", idx, "error", err)
				http.Error(w, "Error escribiendo dump", http.StatusInternalServerError)
				return
			}
		}
	}

	utils.LoggerConFormato("## PID: %d - Memory Dump realizado correctamente", pid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (memo *Memo) FinalizarProceso(w http.ResponseWriter, r *http.Request) {

	body_Bytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error - (FinalizarProceso) - Leyendo la solicitud", "error", err)
		http.Error(w, "Error leyendo el body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mensaje := string(body_Bytes)

	slog.Debug("Llego peticion finalizar proceso", "mensaje", mensaje)

	pid, err := strconv.Atoi(mensaje)

	if err != nil {
		slog.Error("Error - (FinalizarProceso) - Conversiones a int", "mensaje", mensaje, "error", err)
		http.Error(w, "PID invalido", http.StatusBadRequest)
		return
	}

	eliminado_correctamente := memo.EliminarProceso(pid)

	if eliminado_correctamente != nil {
		slog.Error("Error - (FinalizarProceso/EliminarProceso) - no se pudo eliminar, anda a saber porque",
			"pid", pid)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("NO_OK"))
		return
	}

	mutex_metricas.Lock()
	defer mutex_metricas.Unlock()

	mt_a_log, ok := memo.metricas[pid]
	if ok {
		utils.LoggerConFormato(
			"## PID: %d - Proceso Destruido - Metricas - Acc.T.Pag: %d; Inst.Sol.: %d; SWAP: %d; Mem.Prin.: %d; Lec.Mem.: %d; Esc.Mem.: %d",
			pid, mt_a_log[Accesos_a_tpags], mt_a_log[Cant_instr_solicitadas], mt_a_log[Bajadas_de_swap],
			mt_a_log[Subidas_a_memoria], mt_a_log[Cant_read], mt_a_log[Cant_write])
		delete(memo.metricas, pid)
	} else {
		slog.Warn("Advertencia - (FinalizarProceso) - Metricas no encontradas para PID", "pid", pid)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

}

func (memo *Memo) IncrementarMetrica(pid int, cod_metrica int) {
	memo.metricas[pid][cod_metrica]++
}
