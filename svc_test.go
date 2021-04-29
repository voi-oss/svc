package svc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWorkerInitOrder(t *testing.T) {
	// Arrange

	s, err := New("dummy-name", "dummy-version")
	require.NoError(t, err)

	var actualSeq []string

	w1 := &WorkerMock{
		InitFunc: func(*zap.Logger) error {
			actualSeq = append(actualSeq, "w1Init")
			return nil
		},
		RunFunc:       func() error { return nil },
		TerminateFunc: func() error { return nil },
	}
	w2 := &WorkerMock{
		InitFunc: func(*zap.Logger) error {
			actualSeq = append(actualSeq, "w2Init")
			return nil
		},
		RunFunc:       func() error { return nil },
		TerminateFunc: func() error { return nil },
	}
	w3 := &WorkerMock{
		InitFunc: func(*zap.Logger) error {
			actualSeq = append(actualSeq, "w3Init")
			return nil
		},
		RunFunc:       func() error { return nil },
		TerminateFunc: func() error { return nil },
	}

	// Act

	s.AddWorker("w1", w1)
	s.AddWorker("w2", w2)
	s.AddWorker("w3", w3)
	s.Run()

	// Assert

	expectedSeq := []string{
		"w1Init",
		"w2Init",
		"w3Init",
	}

	assert.Equal(t, expectedSeq, actualSeq)
}

func TestShutdown(t *testing.T) {
	// Arrange

	termWorkerCh := make(chan struct{})
	dummyWorker := &WorkerMock{
		InitFunc:      func(*zap.Logger) error { return nil },
		RunFunc:       func() error { <-termWorkerCh; return nil },
		TerminateFunc: func() error { termWorkerCh <- struct{}{}; return nil },
	}

	s, err := New("dummy-service", "v0.0.0")
	require.NoError(t, err)

	s.AddWorker("dummy-worker", dummyWorker)

	termSvcCh := make(chan struct{})
	go func() { s.Run(); termSvcCh <- struct{}{} }()

	s.Shutdown()

	select {
	case <-termSvcCh: // Success
	case <-time.After(3 * time.Second): // time is arbitrary, just "long enough"
		require.FailNow(t, "Service has not been shut down")
	}
}

func TestContextCanceled(t *testing.T) {
	dummyWorker := &WorkerMock{
		InitFunc: func(*zap.Logger) error { return nil },
		RunFunc: func() error {
			return fmt.Errorf("stopped: %w", context.Canceled)
		},
		TerminateFunc: func() error { return nil },
	}

	s, err := New("dummy-service", "v0.0.1")
	require.NoError(t, err)
	s.AddWorker("dummy-worker", dummyWorker)

	// Run should not log fatal
	s.Run()
}

var _ Worker = (*WorkerMock)(nil)

type WorkerMock struct {
	InitFunc      func(*zap.Logger) error
	RunFunc       func() error
	TerminateFunc func() error
}

func (w *WorkerMock) Init(l *zap.Logger) error {
	if w.InitFunc == nil {
		panic("WorkerMock: Init was called but InitFunc was not mocked!")
	}
	return w.InitFunc(l)
}
func (w *WorkerMock) Run() error {
	if w.RunFunc == nil {
		panic("WorkerMock: Run was called but RunFunc was not mocked!")
	}
	return w.RunFunc()
}

func (w *WorkerMock) Terminate() error {
	if w.TerminateFunc == nil {
		panic("WorkerMock: Terminate was called but TerminateFunc was not mocked!")
	}
	return w.TerminateFunc()
}
