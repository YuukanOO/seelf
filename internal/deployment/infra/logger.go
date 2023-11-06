package infra

import (
	"fmt"
	"io"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type stepLogger struct {
	writer io.Writer
}

// Instantiates a new step logger to provide a simple way to build a deployment logfile.
func NewStepLogger(writer io.Writer) domain.StepLogger {
	return &stepLogger{writer}
}

func (l *stepLogger) Stepf(format string, args ...any) {
	l.print("[STEP]", format, args)
}

func (l *stepLogger) Warnf(format string, args ...any) {
	l.print("[WARN]", format, args)
}

func (l *stepLogger) Infof(format string, args ...any) {
	l.print("[INFO]", format, args)
}

func (l *stepLogger) Error(err error) {
	l.print("[ERROR]", err.Error(), nil)
}

func (l *stepLogger) print(prefix string, format string, args []any) {
	l.writer.Write([]byte(fmt.Sprintf("%s %s\n", prefix, fmt.Sprintf(format, args...))))
}
