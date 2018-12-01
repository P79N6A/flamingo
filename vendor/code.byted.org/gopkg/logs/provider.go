package logs

type LogProvider interface {
	Init() error
	SetLevel(l int)
	WriteMsg(msg string, level int) error
	Destroy() error
	Flush() error
}
