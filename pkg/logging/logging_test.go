package logging

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/jgfranco17/reposcout/pkg/environment"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestApplyToContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := New(&buf, logrus.TraceLevel)
	ctx := AddToContext(context.Background(), logger)
	assert.Equal(t, logger, FromContext(ctx))
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := New(&buf, logrus.TraceLevel)
	ctx := AddToContext(context.Background(), logger)
	assert.Equal(t, logger, FromContext(ctx))
}

func TestGetLogLevel(t *testing.T) {
	testCases := []struct {
		envValue      string
		expectedLevel logrus.Level
	}{
		{envValue: "DEBUG", expectedLevel: logrus.DebugLevel},
		{envValue: "INFO", expectedLevel: logrus.InfoLevel},
		{envValue: "WARN", expectedLevel: logrus.WarnLevel},
		{envValue: "ERROR", expectedLevel: logrus.ErrorLevel},
		{envValue: "PANIC", expectedLevel: logrus.PanicLevel},
		{envValue: "FATAL", expectedLevel: logrus.FatalLevel},
		{envValue: "TRACE", expectedLevel: logrus.TraceLevel},
		{envValue: "UNKNOWN", expectedLevel: logrus.InfoLevel}, // Default case
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Using log level [%s]", tc.envValue), func(t *testing.T) {
			t.Setenv(environment.ENV_KEY_LOG_LEVEL, tc.envValue)
			logLevel := GetLogLevel()
			assert.Equal(t, tc.expectedLevel, logLevel)
		})
	}
}
