package svc

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name          string
		serviceOption Option
	}{
		{
			name:          "console logger",
			serviceOption: WithConsoleLogger(zap.InfoLevel),
		},
		{
			name: "console logger with options",
			serviceOption: WithConsoleLogger(zap.DebugLevel, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				return core
			})),
		},
		{
			name:          "development logger",
			serviceOption: WithDevelopmentLogger(),
		},
		{
			name:          "development logger with options",
			serviceOption: WithDevelopmentLogger(zap.Development()),
		},
		{
			name:          "production logger",
			serviceOption: WithProductionLogger(),
		},
		{
			name:          "production logger with options",
			serviceOption: WithProductionLogger(zap.WithCaller(true)),
		},
		{
			name:          "stackdriver logger",
			serviceOption: WithStackdriverLogger(zap.WarnLevel),
		},
		{
			name: "stackdriver logger with options",
			serviceOption: WithStackdriverLogger(zap.WarnLevel, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				return core
			}), zap.AddCaller()),
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			_, err := New("dummy-name", "dummy-version", tc.serviceOption)
			require.NoError(t, err)
		})
	}
}
