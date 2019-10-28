package main

import (
	"github.com/voi-go/svc"
	"go.uber.org/zap"
)

var _ svc.Worker = (*dummyWorker)(nil)

type dummyWorker struct{}

func (d *dummyWorker) Init(*zap.Logger) error { return nil }
func (d *dummyWorker) Terminate() error       { return nil }
func (d *dummyWorker) Run() error             { select {} }

func main() {
	s, err := svc.New("minimal-service", "1.0.0")
	svc.MustInit(s, err)

	w := &dummyWorker{}
	s.AddWorker("dummy-worker", w)

	s.Run()
}
