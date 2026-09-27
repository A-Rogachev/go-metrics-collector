package main

import (
	"flag"
)

// адрес и порт для запуска сервера
var serverAddress string

func parseFlags() {
	flag.StringVar(&serverAddress, "a", ":8080", "address and port to run server")
	flag.Parse()
}
