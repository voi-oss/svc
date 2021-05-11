package svc

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogger(t *testing.T) {
	s, err := New("dummy-name", "dummy-version")
	require.NoError(t, err)

	logger, atom := s.newLogger(zap.InfoLevel, zapcore.NewConsoleEncoder(zapcore.EncoderConfig{}))
	err = assignLogger(s, logger, atom)
	require.NoError(t, err)
}
