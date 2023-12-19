// The package log provides an abstraction above a specific logger implementation.
package log

import (
	"go.uber.org/zap"
)

const (
	DebugLevel                 = Level(zap.DebugLevel)
	InfoLevel                  = Level(zap.InfoLevel)
	WarnLevel                  = Level(zap.WarnLevel)
	ErrorLevel                 = Level(zap.ErrorLevel)
	OutputConsole OutputFormat = "console"
	OutputJSON    OutputFormat = "json"
)

type (
	Level        int8   // Log level used by the logger.
	OutputFormat string // Output format used by the logger.

	// Logger used to log things out
	Logger interface {
		Error(args ...any)
		Errorw(msg string, keysAndValues ...any)

		Debug(args ...any)
		Debugw(msg string, keysAndValues ...any)

		Info(args ...any)
		Infow(msg string, keysAndValues ...any)

		Warn(args ...any)
		Warnw(msg string, keysAndValues ...any)
	}

	// Configurable logger to define additional settings.
	ConfigurableLogger interface {
		Logger
		Configure(OutputFormat, Level, bool) error // Configure the logger output format, level and debug informations.
	}

	wrappedLogger struct {
		*zap.SugaredLogger
	}
)

// Builds a new logger.
func NewLogger() (ConfigurableLogger, error) {
	l, err := configure(OutputConsole, ErrorLevel, false)

	if err != nil {
		return nil, err
	}

	return &wrappedLogger{l.Sugar()}, nil
}

func (l *wrappedLogger) Configure(format OutputFormat, lvl Level, debug bool) error {
	newLogger, err := configure(format, lvl, debug)

	if err != nil {
		return err
	}

	l.SugaredLogger = newLogger.Sugar()

	return nil
}

func configure(format OutputFormat, lvl Level, debug bool) (*zap.Logger, error) {
	conf := zap.NewProductionConfig()
	conf.Development = debug
	conf.Encoding = string(format)

	switch format {
	case OutputConsole:
		conf.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	case OutputJSON:
		conf.EncoderConfig = zap.NewProductionEncoderConfig()
	}

	return conf.Build()
}
