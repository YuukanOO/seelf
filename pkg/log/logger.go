// The package log provides an abstraction above a specific logger implementation.
package log

import (
	"errors"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ErrInvalidLevelValue  = errors.New("invalid_level_value")
	ErrInvalidFormatValue = errors.New("invalid_format_value")
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
		Configure(OutputFormat, Level) error // Configure the logger output format and level.
	}

	wrappedLogger struct {
		*zap.SugaredLogger
	}
)

// Builds a new logger.
func NewLogger() (ConfigurableLogger, error) {
	l, err := configure(OutputConsole, InfoLevel)

	if err != nil {
		return nil, err
	}

	return &wrappedLogger{l.Sugar()}, nil
}

// Try to parse the given raw level string into a valid log.Level.
func ParseLevel(lvl string) (Level, error) {
	switch strings.ToLower(lvl) {
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	default:
		return 0, ErrInvalidLevelValue
	}
}

// Try to parse the given raw format string into a valid log.OutputFormat.
func ParseFormat(format string) (OutputFormat, error) {
	switch strings.ToLower(format) {
	case "console":
		return OutputConsole, nil
	case "json":
		return OutputJSON, nil
	default:
		return "", ErrInvalidFormatValue
	}
}

func (l *wrappedLogger) Configure(format OutputFormat, lvl Level) error {
	newLogger, err := configure(format, lvl)

	if err != nil {
		return err
	}

	l.SugaredLogger = newLogger.Sugar()

	return nil
}

func configure(format OutputFormat, lvl Level) (*zap.Logger, error) {
	conf := zap.NewProductionConfig()
	conf.Level.SetLevel(zapcore.Level(lvl))
	conf.Development = lvl == DebugLevel
	conf.Encoding = string(format)

	switch format {
	case OutputConsole:
		conf.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	case OutputJSON:
		conf.EncoderConfig = zap.NewProductionEncoderConfig()
	}

	return conf.Build()
}
