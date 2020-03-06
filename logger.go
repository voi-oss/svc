package svc

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger(level zapcore.Level, encoder zapcore.Encoder) (*zap.Logger, zap.AtomicLevel) {
	atom := zap.NewAtomicLevel()
	atom.SetLevel(level)

	logger := zap.New(zapcore.NewSampler(zapcore.NewCore(
		encoder,
		zapcore.Lock(os.Stdout),
		atom,
	), time.Second, 100, 10),
		zap.ErrorOutput(zapcore.Lock(os.Stderr)),
		zap.AddCaller(),
	)

	return logger, atom
}
