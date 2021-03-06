package logger

import (
	"fmt"

	endure "github.com/spiral/endure/pkg/container"
)

type Logger struct {
}

type SuperLogger interface {
	SuperLogToStdOut(message string)
}

func (l *Logger) SuperLogToStdOut(message string) {
	// BOOM
	fmt.Println("logger says: " + message)
}

func (l *Logger) Init() error {
	return nil
}

func (l *Logger) Provides() []interface{} {
	return []interface{}{
		l.LoggerInstance,
	}
}

func (l *Logger) LoggerInstance(name endure.Named) (*Logger, error) {
	println(name.Name() + " invoke " + "logger")
	return l, nil
}
