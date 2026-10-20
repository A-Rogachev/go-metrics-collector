package logger

type Event interface {
	Msg(string)
}

type Logger interface {
	Error() Event
	Info() Event
}
