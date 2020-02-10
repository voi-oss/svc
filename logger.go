package svc

import (
	"os"
	"strings"
	"time"

	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger(name, version string, level zapcore.Level) (*zap.Logger, zap.AtomicLevel) {
	logFormat := strings.ToLower(os.Getenv("LOG_FORMAT"))

	atom := zap.NewAtomicLevel()
	atom.SetLevel(level)

	// stackdriver
	if logFormat == "stackdriver" {
		logger := zap.New(zapcore.NewSampler(zapcore.NewCore(
			zapcore.NewJSONEncoder(zapdriver.NewProductionEncoderConfig()),
			zapcore.Lock(os.Stdout),
			atom,
		), time.Second, 100, 10),
			zap.ErrorOutput(zapcore.Lock(os.Stderr)),
			zap.AddCaller(),
		)
		logger = logger.With(zapdriver.ServiceContext(name), zapdriver.Label("version", version))
		return logger, atom
	}

	// console text logger or fallback to default generic JSON
	var enc zapcore.Encoder
	if logFormat == "console" {
		config := zap.NewProductionEncoderConfig()
		config.EncodeTime = zapcore.RFC3339TimeEncoder
		enc = zapcore.NewConsoleEncoder(config)
	} else {
		enc = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	}
	logger := zap.New(zapcore.NewSampler(zapcore.NewCore(
		enc,
		zapcore.Lock(os.Stdout),
		atom,
	), time.Second, 100, 10),
		zap.ErrorOutput(zapcore.Lock(os.Stderr)),
		zap.AddCaller(),
		zap.Fields(zap.String("app", name), zap.String("version", version)),
	)
	return logger, atom
}
