package log

import (
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
