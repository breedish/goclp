// Package server contains everything for setting up and running the HTTP server.
package server

import (
	"canvas/messaging"
	"canvas/storage"
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	address  string
	database *storage.Database
	mux      chi.Router
	server   *http.Server
	queue    *messaging.Queue
	log      *zap.Logger
}

type Options struct {
	Database *storage.Database
	Host     string
	Port     int
	Queue    *messaging.Queue
	Log      *zap.Logger
}

func New(opts Options) *Server {
	address := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	mux := chi.NewMux()

	if opts.Log == nil {
		opts.Log = zap.NewNop()
	}

	return &Server{
		address:  address,
		database: opts.Database,
		mux:      mux,
		server: &http.Server{
			Addr:              address,
			Handler:           mux,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       5 * time.Second,
		},
		queue: opts.Queue,
		log:   opts.Log,
	}
}

// Start the Server by setting up routes and listening for HTTP requests on the given address
func (s *Server) Start() error {
	if err := s.database.Connect(); err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}

	s.setupRoutes()

	s.log.Info("Starting", zap.String("address", s.address))
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("error starting server: %w", err)
	}
	return nil
}

// Stop the Server gracefully within the timeout.
func (s *Server) Stop() error {
	s.log.Info("Stopping")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error stopping server: %w", err)
	}

	return nil
}
