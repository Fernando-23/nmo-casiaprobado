package main

import (
	"time"
)

type ConfigIO struct {
	Ip_kernel     string `json:"ip_kernel"`
	Puerto_kernel int    `json:"port_kernel"`
	Puerto_io     int    `json:"port_io"`
	Ip_io         string `json:"ip_io"`
	Log_level     string `json:"log_level"`
}

var (
	config_io      *ConfigIO
	ch_cancelar_io chan struct{}
	url_kernel     string
)

var (
	url_io         string
	nombre_io      string
	tiempo_en_io   time.Time
	hay_proceso_io bool
	duracion_en_io float64
)
