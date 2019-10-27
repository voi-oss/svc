package svc

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/gin-gonic/gin"
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

// WithProductionRouter is an option that uses Gin as HTTP router in release
// mode.
func WithProductionRouter() Option {
	return func(s *SVC) error {
		// Has to be done before `gin.New()` and overrides `GIN_MODE` env!
		gin.SetMode(gin.ReleaseMode)
		return WithRouter(gin.New())(s)
	}
}

// WithDevelopmentRouter is an option that uses Gin as HTTP router in debug
// mode.
func WithDevelopmentRouter() Option {
	return func(s *SVC) error {
		// Has to be done before `gin.New()` and overrides `GIN_MODE` env!
		gin.SetMode(gin.DebugMode)
		return WithRouter(gin.New())(s)
	}
}

// WithRouter is an option that replaces the HTTP router with the given Gin
// router.
func WithRouter(router *gin.Engine) Option {
	return func(s *SVC) error {
		s.Router = router
		return nil
	}
}

// WithProductionLogger is an option that uses a zap Logger with configurations
// set meant to be used for production.
func WithProductionLogger() Option {
	return func(s *SVC) error {
		logger, atom := newLogger(s.Name, s.Version, zapcore.InfoLevel)
		return assignLogger(s, logger, atom)
	}
}

// WithDevelopmentLogger is an option that uses a zap Logger with
// configurations set meant to be used for development.
func WithDevelopmentLogger() Option {
	return func(s *SVC) error {
		logger, atom := newLogger(s.Name, s.Version, zapcore.DebugLevel)
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
		s.Router.GET("/loglevel", gin.WrapH(s.atom))
		s.Router.PUT("/loglevel", gin.WrapH(s.atom))

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
		s.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

		return nil
	}
}

// WithPProfHandlers is an option that exposes Go's Performance Profiler via
// HTTP routes.
func WithPProfHandlers() Option {
	return func(s *SVC) error {
		// See https://github.com/golang/go/blob/master/src/net/http/pprof/pprof.go#L72-L77
		s.Router.GET("/debug/pprof/", gin.WrapF(pprof.Index))
		s.Router.GET("/debug/pprof/cmdline", gin.WrapF(pprof.Cmdline))
		s.Router.GET("/debug/pprof/profile", gin.WrapF(pprof.Profile))
		s.Router.GET("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
		s.Router.GET("/debug/pprof/trace", gin.WrapF(pprof.Trace))
		// See https://github.com/golang/go/blob/master/src/net/http/pprof/pprof.go#L248-L258
		s.Router.GET("/debug/pprof/allocs", gin.WrapH(pprof.Handler("allocs")))
		s.Router.GET("/debug/pprof/block", gin.WrapH(pprof.Handler("block")))
		s.Router.GET("/debug/pprof/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		s.Router.GET("/debug/pprof/heap", gin.WrapH(pprof.Handler("heap")))
		s.Router.GET("/debug/pprof/mutex", gin.WrapH(pprof.Handler("mutex")))
		s.Router.GET("/debug/pprof/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))

		return nil
	}
}

// WithHealthz is an option that exposes Kubernetes conform Healthz HTTP
// routes.
func WithHealthz() Option {
	return func(s *SVC) error {
		// Register live probe handler
		s.Router.GET("/live", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{"status": "Still Alive!"})
		})

		// Register ready probe handler
		s.Router.GET("/ready", func(c *gin.Context) {
			var errs []error
			for n, w := range s.workers {
				if hw, ok := w.(Healther); ok {
					if err := hw.Healthy(); err != nil {
						errs = append(errs, fmt.Errorf("worker %s: %s", n, err))
					}
				}
			}
			if len(errs) > 0 {
				s.logger.Warn("Ready check failed", zap.Any("errs", errs))
				c.JSON(http.StatusServiceUnavailable, gin.H{"errors": errs})
				return
			}
			c.Status(http.StatusOK)
		})

		return nil
	}
}
