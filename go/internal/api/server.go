package api

import (
	"context"
	"fmt"
	"net/http"

	"veriChat/go/internal/metrics"
	"veriChat/go/internal/service"
)

type Server struct {
	httpServer *http.Server
	service    *service.MessageService
}

func NewServer(addr string, svc *service.MessageService) *Server {
	mux := http.NewServeMux()

	// Handlers
	mux.Handle("/metrics", metrics.MetricsHandler())
	mux.Handle("/messages", metrics.InstrumentHandler(makePostMessageHandler(svc)))
	mux.Handle("/merkle", metrics.InstrumentHandler(http.HandlerFunc(PostMerkleHandler)))
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	fmt.Printf("API server listening on %s\n", addr)
	return &Server{
		httpServer: srv,
		service:    svc,
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
