package logging

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestInitLogging(t *testing.T) {
	tests := []struct {
		name      string
		levelStr  string
		wantLevel zerolog.Level
	}{
		{
			name:      "debug level",
			levelStr:  "debug",
			wantLevel: zerolog.DebugLevel,
		},
		{
			name:      "info level",
			levelStr:  "info",
			wantLevel: zerolog.InfoLevel,
		},
		{
			name:      "error level",
			levelStr:  "error",
			wantLevel: zerolog.ErrorLevel,
		},
		{
			name:      "invalid level",
			levelStr:  "invalid",
			wantLevel: zerolog.ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitLogging(tt.levelStr)
			assert.Equal(t, tt.wantLevel, zerolog.GlobalLevel())
		})
	}
}

func TestLogOutput(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	zerolog.DefaultContextLogger = &log

	// Test debug logging
	InitLogging("debug")
	log.Debug().Str("key", "value").Msg("debug message")

	var output map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &output)
	assert.NoError(t, err)
	assert.Equal(t, "debug message", output["message"])
	assert.Equal(t, "value", output["key"])
	assert.Equal(t, "debug", output["level"])

	// Clear buffer
	buf.Reset()

	// Test info logging
	InitLogging("info")
	log.Info().Str("key", "value").Msg("info message")

	err = json.Unmarshal(buf.Bytes(), &output)
	assert.NoError(t, err)
	assert.Equal(t, "info message", output["message"])
	assert.Equal(t, "value", output["key"])
	assert.Equal(t, "info", output["level"])
}
