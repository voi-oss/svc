package svc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
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
	AliveFunc     func() error
	HealthyFunc   func() error
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

func (w *WorkerMock) Alive() error {
	if w.AliveFunc == nil {
		panic("WorkerMock: Alive was called but AliveFunc was not mocked!")
	}
	return w.AliveFunc()
}

func (w *WorkerMock) Healthy() error {
	if w.HealthyFunc == nil {
		panic("WorkerMock: Healthy was called but HealthyFunc was not mocked!")
	}
	return w.HealthyFunc()
}

func TestSVC_AddWorkerWithInitRetry(t *testing.T) {
	var attempts uint
	tests := []struct {
		name             string
		w                Worker
		retryOpts        []retry.Option
		expectedAttempts uint
	}{
		{
			name: "succeeds after 3 attempts, with max  10 attempts",
			w: &WorkerMock{
				InitFunc: func(*zap.Logger) error {
					if attempts < 3 {
						attempts++
						return fmt.Errorf("failed")
					}
					return nil
				},
				TerminateFunc: func() error { return nil },
				RunFunc:       func() error { return nil },
			},
			retryOpts:        []retry.Option{retry.Attempts(10), retry.MaxDelay(1 * time.Millisecond), retry.Delay(1 * time.Millisecond)},
			expectedAttempts: 3,
		},
		{
			name: "fails after 3 attempts, with max 3 attempts",
			w: &WorkerMock{
				InitFunc: func(*zap.Logger) error {
					attempts++
					return fmt.Errorf("failed")
				},
				TerminateFunc: func() error { return nil },
				RunFunc:       func() error { return nil },
			},
			retryOpts:        []retry.Option{retry.Attempts(3), retry.MaxDelay(1 * time.Millisecond), retry.Delay(1 * time.Millisecond)},
			expectedAttempts: 3,
		},
	}
	for _, tt := range tests {
		attempts = 0
		t.Run(tt.name, func(t *testing.T) {
			s, err := New("dummy-name", "dummy-version")
			require.NoError(t, err)

			s.AddWorkerWithInitRetry("test", tt.w, tt.retryOpts)
			s.Run()
			require.Equal(t, tt.expectedAttempts, attempts)
		})
	}
}
