package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ServerConfig struct {
	Name string
	Port int
}

type Server struct {
	config     ServerConfig
	engine     *gin.Engine
	httpServer *http.Server
}

func New(cfg ServerConfig) *Server {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(traceFilter())
	engine.Use(accessLogFilter())
	engine.Use(metricsFilter())

	engine.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           engine,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{
		config:     cfg,
		engine:     engine,
		httpServer: httpServer,
	}
}

func (s *Server) Engine() *gin.Engine {
	return s.engine
}

func (s *Server) Start() error {
	fmt.Printf("[%s] HTTP server listening on :%d\n", s.config.Name, s.config.Port)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[%s] listen error: %v\n", s.config.Name, err)
		}
	}()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func GracefulStop(srv *Server, timeout time.Duration) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-quit

	fmt.Printf("\n[%s] received signal %v, shutting down...\n", srv.config.Name, sig)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("[%s] server shutdown error: %v\n", srv.config.Name, err)
	} else {
		fmt.Printf("[%s] HTTP server stopped gracefully\n", srv.config.Name)
	}
}
