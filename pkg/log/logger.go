package log

import (
	"fmt"
	"io"

	"go.uber.org/zap"
)

type Logger interface {
	Error(args ...any)
	Errorw(msg string, keysAndValues ...any)

	Debug(args ...any)
	Debugw(msg string, keysAndValues ...any)

	Info(args ...any)
	Infow(msg string, keysAndValues ...any)

	Warn(args ...any)
	Warnw(msg string, keysAndValues ...any)
}

// Builds a new logger. The format will difer based on the verbose flag.
func NewLogger(verbose bool) Logger {
	var (
		logger *zap.Logger
		err    error
	)

	if verbose {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		panic(err) // It should never happened so we better panic here
	}

	return logger.Sugar()
}

type (
	// Specific logger interface use by deployment jobs to document the deployment process.
	StepLogger interface {
		Step(string)
		Stepf(string, ...any)
		Warn(string)
		Warnf(string, ...any)
		Info(string)
		Infof(string, ...any)
		Error(error)
	}

	stepLogger struct {
		writer io.Writer
	}
)

// Instantiates a new step logger to provide a simple way to build a deployment logfile.
func NewStepLogger(writer io.Writer) StepLogger {
	return &stepLogger{writer}
}

func (l *stepLogger) Step(msg string) {
	l.print("[STEP]", msg)
}

func (l *stepLogger) Stepf(format string, args ...any) {
	l.Step(fmt.Sprintf(format, args...))
}

func (l *stepLogger) Warn(msg string) {
	l.print("[WARN]", msg)
}

func (l *stepLogger) Warnf(format string, args ...any) {
	l.Warn(fmt.Sprintf(format, args...))
}

func (l *stepLogger) Info(msg string) {
	l.print("[INFO]", msg)
}

func (l *stepLogger) Infof(format string, args ...any) {
	l.Info(fmt.Sprintf(format, args...))
}

func (l *stepLogger) Error(err error) {
	l.print("[ERROR]", err)
}

func (l *stepLogger) print(prefix string, msg any) {
	l.writer.Write([]byte(fmt.Sprintf("%s %s\n", prefix, msg)))
}
