// Аргументы агента:

//     Флаг -a=<ЗНАЧЕНИЕ> отвечает за адрес эндпоинта HTTP-сервера (по умолчанию localhost:8080).
//     Флаг -r=<ЗНАЧЕНИЕ> позволяет переопределять reportInterval — частоту отправки метрик на сервер (по умолчанию 10 секунд).
//     Флаг -p=<ЗНАЧЕНИЕ> позволяет переопределять pollInterval — частоту опроса метрик из пакета runtime (по умолчанию 2 секунды).

// При попытке передать приложению незвестные флаги оно должно завершаться с сообщением о соответствующей ошибке.
// Значения интервалов времени должны задаваться в секундах.
// Во всех случаях должны присутствовать значения по умолчанию.
//

package main

import (
	"flag"
)

var apiAddress string
var reportInterval int
var pollInterval int

func parseFlags() {
	flag.StringVar(&apiAddress, "a", "http://localhost:8080", "api address and port")
	flag.IntVar(&reportInterval, "r", 10, "how often send metrics via api")
	flag.IntVar(&pollInterval, "p", 2, "how often get metrics via runtime package")
	flag.Parse()
}
