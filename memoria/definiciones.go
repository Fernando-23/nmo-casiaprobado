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
	memoria_sistema  map[int][]string
	global_ptrs_tpag map[int][]*Tabla
	//------------------[pid][0,1,2,...n-niveles], cada nivel apunta a *Tabla

}

type Tabla struct {
	lv_tabla      int
	Entradas      []*EntradaTPag
	bit_presencia int
	sgte_tabla    *Tabla
}

type EntradaTPag struct {
	nro_marco int
	offset    int
}

var (
	memoria_usuario []byte
	config_memo     *ConfigMemo
	tam_memo_actual int
)
