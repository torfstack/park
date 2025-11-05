package logging

import (
	"errors"
	"fmt"
)

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

func LogError(msg string, err error) {
	Log(msg)
	for err != nil {
		Log(err.Error())
		if unwrap := errors.Unwrap(err); unwrap != nil {
			err = unwrap
		} else {
			break
		}
	}
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
