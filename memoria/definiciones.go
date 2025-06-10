package main

type ConfigMemo struct {
	Puerto_mem        int    `json:"port_memory"`
	Ip_memoria        string `json:"ip_memory"`
	Tamanio_memoria   int    `json:"memory_size"`
	Tamanio_pag       int    `json:"page_size"`
	Cant_entradasXpag int    `json:"entries_per_page"`
	Cant_niveles      int    `json:"number_of_levels"`
	Delay_memoria     int    `json:"memory_delay"`
	Path_swap         string `json:"swapfile_path"`
	Delay_swap        int    `json:"swap_delay"`
	Log_level         string `json:"log_level"`
	Path_dump         string `json:"dump_path"`
}

type Memo struct {
	memoria_sistema map[int][]string
	l_proc          map[int]*Proceso
	ptrs_raiz_tpag  map[int]*NivelTPag
	tabla_frames    []int //sincronizacion
}

type Proceso struct {
	ptr_a_frames_asignados []*int //apunta a elementos de la tabla_frames
}

/*
Proceso marcos asignados

ptr_a_frames_asignados  map[int]*Frame        ->1    ->6     ->8   ->34


frames
   1	   --
   2	   --
   3	   --
   4
   5
   6
   7

[0]  p1 'HOLAMUN'
[1]  p1 'DOCOMO'
.
.
.
[79] p1 'ESTAS'

*/

type Metricas struct {
	accesos_a_tpags        int
	cant_instr_solicitadas int
	bajadas_de_swap        int
	subidas_a_memoria      int
	cant_read              int
	cant_write             int
}

type NivelTPag struct {
	lv_tabla   int
	entradas   []*int
	sgte_nivel *NivelTPag
}

var (
	memoria_principal  []byte
	config_memo        *ConfigMemo
	frames_disponibles int
	tam_memo_actual    int
)
