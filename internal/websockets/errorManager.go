package websockets

import(
	"time"
)

//Error Event
type ErrorEvent struct{ 
	Time   time.Time
	Client 	 string
	Source 	 string
	Severity Severity 
	Message  string 
	Error 	 error
}

// Types of severity
type Severity string
const (
	Info 	Severity = "info"
	Fatal 	Severity = "fatal"
	Warning Severity = "warning"
	Error 	Severity = "error"
)
