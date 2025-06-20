package main

type ConfigIO struct {
	Ip_kernel     string `json:"ip_kernel"`
	Puerto_kernel int    `json:"port_kernel"`
	Puerto_io     int    `json:"port_io"`
	Ip_io         string `json:"ip_io"`
	Log_level     string `json:"log_level"`
}

var (
	config_IO *ConfigIO
	url_io    string
	nombre_io string
)
