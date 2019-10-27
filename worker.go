package svc

import (
	"go.uber.org/zap"
)

// Worker defines a SVC worker.
type Worker interface {
	Init(*zap.Logger) error
	Run() error
	Terminate() error
}

// Healther defines a worker that can report his healthz status.
type Healther interface {
	Healthy() error
}
