package logging

import "fmt"

var (
	LogLevelDebug = false
)

func Log(msg string) {
	fmt.Println(msg)
}

func Logf(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	fmt.Println(msg)
}

func LogDebug(msg string) {
	if LogLevelDebug {
		Log(msg)
	}
}

func LogDebugf(msg string, args ...interface{}) {
	if LogLevelDebug {
		Logf(msg, args...)
	}
}
