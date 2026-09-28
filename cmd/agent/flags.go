package main

import (
	"flag"
)

var apiAddress string
var reportInterval int
var pollInterval int

func parseFlags() {
	flag.StringVar(&apiAddress, "a", "localhost:8080/", "api address and port")
	flag.IntVar(&reportInterval, "r", 10, "how often send metrics via api")
	flag.IntVar(&pollInterval, "p", 2, "how often get metrics via runtime package")
	flag.Parse()
}
