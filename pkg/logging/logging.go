package logging

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/jgfranco17/reposcout/pkg/environment"
	"github.com/sirupsen/logrus"
)

type contextLogKey string

const contextKey contextLogKey = "logger"

func New(stream io.Writer, level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(stream)
	logger.SetLevel(level)

	formatter := &logrus.TextFormatter{
		DisableColors:          false,
		PadLevelText:           true,
		QuoteEmptyFields:       true,
		FullTimestamp:          true,
		DisableSorting:         true,
		DisableLevelTruncation: true,
		TimestampFormat:        time.TimeOnly,
	}
	logger.SetFormatter(formatter)

	return logger
}

type RequestMetadata struct {
	RequestID   string
	Environment string
	Version     string
}

func AddToContext(ctx context.Context, logger *logrus.Logger) context.Context {
	return context.WithValue(ctx, contextKey, logger)
}

func FromContext(ctx context.Context) *logrus.Logger {
	if logger, ok := ctx.Value(contextKey).(*logrus.Logger); ok {
		return logger
	}

	panic("no logger set in context")
}

func GetLogLevel() logrus.Level {
	appEnv := environment.GetEnvWithDefault(environment.ENV_KEY_LOG_LEVEL, "INFO")
	stringToLogLevel := map[string]logrus.Level{
		"DEBUG": logrus.DebugLevel,
		"INFO":  logrus.InfoLevel,
		"WARN":  logrus.WarnLevel,
		"ERROR": logrus.ErrorLevel,
		"PANIC": logrus.PanicLevel,
		"FATAL": logrus.FatalLevel,
		"TRACE": logrus.TraceLevel,
	}

	level, exists := stringToLogLevel[strings.ToUpper(appEnv)]
	if !exists {
		return logrus.InfoLevel
	}
	return level
}
