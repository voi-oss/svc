package svc

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var _ Worker = (*httpServer)(nil)

// httpServer defines the internal HTTP Server worker.
type httpServer struct {
	logger     *zap.Logger
	addr       string
	httpServer *http.Server
}

func newHTTPServer(port string, handler http.Handler, logger *log.Logger) *httpServer {
	addr := net.JoinHostPort("", port)
	return &httpServer{
		addr: addr,
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ErrorLog:          logger,
			ReadHeaderTimeout: 5 * time.Second, // https://medium.com/a-journey-with-go/go-understand-and-mitigate-slowloris-attack-711c1b1403f6
		},
	}
}

// Init implements the Worker interface.
func (s *httpServer) Init(logger *zap.Logger) error {
	s.logger = logger

	return nil
}

// Healthy implements the Healther interface.
func (s *httpServer) Healthy() error {
	return nil
}

// Run implements the Worker interface.
func (s *httpServer) Run() error {
	s.logger.Info("Listening and serving HTTP", zap.String("address", s.addr))
	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		s.logger.Error("Failed to serve HTTP", zap.Error(err))
	}
	return nil
}

// Terminate implements the Worker interface.
func (s *httpServer) Terminate() error {
	return s.httpServer.Shutdown(context.Background())
}
