package svc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/blendle/zapdriver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Option defines SVC's option type.
type Option func(*SVC) error

// WithTerminationGracePeriod is an option that sets the termination grace
// period.
func WithTerminationGracePeriod(d time.Duration) Option {
	return func(s *SVC) error {
		s.TerminationGracePeriod = d

		return nil
	}
}

// WithRouter is an option that replaces the HTTP router with the given http
// router.
func WithRouter(router *http.ServeMux) Option {
	return func(s *SVC) error {
		s.Router = router
		return nil
	}
}

// WithLogger is an option that allows you to provide your own customized logger.
func WithLogger(logger *zap.Logger, atom zap.AtomicLevel) Option {
	return func(s *SVC) error {
		return assignLogger(s, logger, atom)
	}
}

// WithDevelopmentLogger is an option that uses a zap Logger with
// configurations set meant to be used for development.
func WithDevelopmentLogger() Option {
	return func(s *SVC) error {
		logger, atom := newLogger(
			zapcore.DebugLevel,
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		)
		logger.With(zap.String("app", s.Name), zap.String("version", s.Version))
		return assignLogger(s, logger, atom)
	}
}

// WithProductionLogger is an option that uses a zap Logger with configurations
// set meant to be used for production.
func WithProductionLogger() Option {
	return func(s *SVC) error {
		logger, atom := newLogger(
			zapcore.InfoLevel,
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		)
		logger.With(zap.String("app", s.Name), zap.String("version", s.Version))
		return assignLogger(s, logger, atom)
	}
}

// WithConsoleLogger is an option that uses a zap Logger with configurations
// set meant to be used for debugging in the console.
func WithConsoleLogger(level zapcore.Level) Option {
	return func(s *SVC) error {

		config := zap.NewProductionEncoderConfig()
		config.EncodeTime = zapcore.RFC3339TimeEncoder

		logger, atom := newLogger(
			level,
			zapcore.NewConsoleEncoder(config),
		)
		logger.With(zap.String("app", s.Name), zap.String("version", s.Version))
		return assignLogger(s, logger, atom)
	}
}

// WithStackdriverLogger is an option that uses a zap Logger with configurations
// set meant to be used for production and is compliant with the GCP/Stackdriver format.
func WithStackdriverLogger(level zapcore.Level) Option {
	return func(s *SVC) error {

		logger, atom := newLogger(
			level,
			zapcore.NewJSONEncoder(zapdriver.NewProductionEncoderConfig()),
		)
		logger.With(zapdriver.ServiceContext(s.Name), zapdriver.Label("version", s.Version))
		return assignLogger(s, logger, atom)
	}
}

func assignLogger(s *SVC, logger *zap.Logger, atom zap.AtomicLevel) error {
	stdLogger, err := zap.NewStdLogAt(logger, zapcore.ErrorLevel)
	if err != nil {
		return err
	}
	undo, err := zap.RedirectStdLogAt(logger, zapcore.ErrorLevel)
	if err != nil {
		return err
	}

	s.logger = logger
	s.stdLogger = stdLogger
	s.atom = atom
	s.loggerRedirectUndo = undo

	return nil
}

// WithLogLevelHandlers is an option that sets up HTTP routes to read write the
// log level.
func WithLogLevelHandlers() Option {
	return func(s *SVC) error {
		s.Router.Handle("/loglevel", s.atom)

		return nil
	}
}

// WithHTTPServer is an option that adds an internal HTTP server exposing
// observability routes.
func WithHTTPServer(port string) Option {
	return func(s *SVC) error {
		httpServer := newHTTPServer(port, s.Router, s.stdLogger)
		s.AddWorker("internal-http-server", httpServer)

		return nil
	}
}

// WithMetrics is an option that exports metrics via prometheus.
func WithMetrics() Option {
	return func(s *SVC) error {
		m := prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name:        "svc_up",
				Help:        "Is the service in this pod up.",
				ConstLabels: prometheus.Labels{"version": s.Version, "name": s.Name},
			},
			func() float64 { return 1 },
		)

		if err := prometheus.Register(m); err != nil {
			s.logger.Error("svc_up could not register", zap.Error(err))
		}

		return nil
	}
}

// WithMetricsHandler is an option that exposes Prometheus metrics for a
//Prometheus scraper.
func WithMetricsHandler() Option {
	return func(s *SVC) error {
		s.Router.Handle("/metrics", promhttp.Handler())

		return nil
	}
}

// WithPProfHandlers is an option that exposes Go's Performance Profiler via
// HTTP routes.
func WithPProfHandlers() Option {
	return func(s *SVC) error {
		// See https://github.com/golang/go/blob/master/src/net/http/pprof/pprof.go#L72-L77
		s.Router.HandleFunc("/debug/pprof/", pprof.Index)
		s.Router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.Router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.Router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		s.Router.HandleFunc("/debug/pprof/trace", pprof.Trace)
		// See https://github.com/golang/go/blob/master/src/net/http/pprof/pprof.go#L248-L258
		s.Router.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
		s.Router.Handle("/debug/pprof/block", pprof.Handler("block"))
		s.Router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		s.Router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		s.Router.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		s.Router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

		return nil
	}
}

// WithHealthz is an option that exposes Kubernetes conform Healthz HTTP
// routes.
func WithHealthz() Option {
	return func(s *SVC) error {
		// Register live probe handler
		s.Router.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status": "Still Alive!"}`))
		})

		// Register ready probe handler
		s.Router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
			var errs []error
			for n, w := range s.workers {
				if hw, ok := w.(Healther); ok {
					if err := hw.Healthy(); err != nil {
						errs = append(errs, fmt.Errorf("worker %s: %s", n, err))
					}
				}
			}
			if len(errs) > 0 {
				s.logger.Warn("Ready check failed", zap.Errors("errors", errs))
				b, err := json.Marshal(map[string]interface{}{"errors": errs})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(b)
			}
		})

		return nil
	}
}
