package grass

//go:generate msgp
type LogEntry struct {
	Category string `msg:"category"`
	Hostname string `msg:"hostname"`
	Message  string `msg:"message"`
}
