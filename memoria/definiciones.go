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
	memoria_sistema     map[int][]string
	tabla_global_nivel0 map[int]*Tabla //Lo hice puntero porque tiene toda la pinta
} //pero no se si para los de nivel0 haga falta, asumo que si

type Tabla struct {
	nivel_tabla int
	nro_marco   int // Este creo q era la cantidad de entradas, estoy medio matado asi q para nosotros del futuro
	offset      int
	sgte_tabla  *Tabla
}

var (
	memoria_usuario []byte
	config_memo     *ConfigMemo
	tam_memo_actual int
)
