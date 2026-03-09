package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/okshelters/shelternav/gateway/handler"
	"github.com/okshelters/shelternav/gateway/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	geoAddr := os.Getenv("GEO_SERVICE_ADDR")
	if geoAddr == "" {
		geoAddr = "localhost:50051"
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	// Establish gRPC connection to geo-service.
	// TODO: Replace insecure credentials with TLS 1.3 for production.
	conn, err := grpc.NewClient(
		geoAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("failed to create gRPC client", slog.String("addr", geoAddr), slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer conn.Close()

	logger.Info("connected to geo-service", slog.String("addr", geoAddr))

	// Build handler dependencies.
	shelterHandler := handler.NewShelterHandler(conn, logger)

	// Build middleware chain: outermost wraps first.
	// Order of execution: logging -> auth -> rate-limit -> cache -> handler
	rateLimiter := middleware.NewRateLimiter(100, 20) // 100 req/s, burst of 20
	cache := middleware.NewResponseCache(1 * time.Second, 1024)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/shelters/nearest", shelterHandler.HandleFindNearest)
	mux.HandleFunc("GET /v1/route", shelterHandler.HandleGetRoute)
	mux.HandleFunc("GET /healthz", handleHealthz)

	// Serve the web UI from the web/ directory. Registered last so API
	// routes take precedence. Uses http.Dir for safe, rooted file access.
	mux.Handle("/", http.FileServer(http.Dir("web")))

	// Compose middleware chain.
	var chain http.Handler = mux
	chain = middleware.Cache(cache, chain)
	chain = middleware.RateLimit(rateLimiter, chain)
	chain = middleware.Auth(chain)
	chain = middleware.Logging(logger, chain)

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           chain,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown via signal handling.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting HTTP server", slog.String("addr", listenAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Block until shutdown signal.
	<-ctx.Done()
	logger.Info("shutdown signal received, draining connections")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
