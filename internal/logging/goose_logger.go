package logging

import "github.com/pressly/goose/v3"

type ParkLoggerGoose struct {
}

var _ goose.Logger = (*ParkLoggerGoose)(nil)

func (p ParkLoggerGoose) Fatalf(format string, v ...interface{}) {
	Fatalf(format, v...)
}

func (p ParkLoggerGoose) Printf(format string, v ...interface{}) {
	Infof(format, v...)
}
