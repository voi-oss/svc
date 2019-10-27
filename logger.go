package svc

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger(name, version string, level zapcore.Level) (*zap.Logger, zap.AtomicLevel) {
	atom := zap.NewAtomicLevel()
	atom.SetLevel(level)

	logger := zap.New(zapcore.NewSampler(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(os.Stdout),
		atom,
	), time.Second, 100, 10),
		zap.ErrorOutput(zapcore.Lock(os.Stderr)),
		zap.AddCaller(),
		zap.Fields(zap.String("app", name), zap.String("version", version)),
	)

	return logger, atom
}
