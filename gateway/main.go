package main

import (
	"context"
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/okshelters/shelternav/gateway/client"
	"github.com/okshelters/shelternav/gateway/handler"
	"github.com/okshelters/shelternav/gateway/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type config struct {
	geoServiceAddr string
	port           string
	jwtSecret      string
	rateLimitRPS   float64
	rateLimitBurst int
	cacheSizeMB    int
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		logger.Error("invalid configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	conn, err := grpc.NewClient(
		cfg.geoServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		logger.Error("failed to create gRPC client", slog.String("addr", cfg.geoServiceAddr), slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer conn.Close()

	geoClient := client.NewGRPCGeoClient(conn)
	shelterHandler := handler.NewShelterHandler(geoClient, logger)

	cacheMaxEntries := cfg.cacheSizeMB * 256
	if cacheMaxEntries < 256 {
		cacheMaxEntries = 256
	}

	rateLimiter := middleware.NewRateLimiter(cfg.rateLimitRPS, cfg.rateLimitBurst)
	cache := middleware.NewResponseCache(1*time.Second, cacheMaxEntries)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/shelters/nearest", shelterHandler.HandleFindNearest)
	mux.HandleFunc("GET /api/v1/route", shelterHandler.HandleGetRoute)
	mux.HandleFunc("GET /healthz", handler.HandleHealthz)
	mux.Handle("GET /readyz", newReadyHandler(conn))
	mux.Handle("GET /debug/vars", expvar.Handler())

	var chain http.Handler = mux
	chain = middleware.Cache(cache, chain)
	chain = middleware.RateLimit(rateLimiter, chain)
	chain = middleware.Auth(cfg.jwtSecret, chain)
	chain = middleware.Logging(logger, chain)
	chain = middleware.Recovery(logger, chain)

	server := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           chain,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting HTTP server", slog.String("addr", server.Addr))
		if serveErr := server.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Error("HTTP server failed", slog.String("error", serveErr.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown server", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func loadConfig() (config, error) {
	cfg := config{
		geoServiceAddr: getenv("GEO_SERVICE_ADDR", "geo-service:9001"),
		port:           strings.TrimPrefix(getenv("PORT", "8080"), ":"),
		jwtSecret:      os.Getenv("JWT_SECRET"),
		rateLimitRPS:   100,
		rateLimitBurst: 100,
		cacheSizeMB:    64,
	}

	if cfg.port == "" {
		return config{}, fmt.Errorf("PORT cannot be empty")
	}

	if raw := os.Getenv("RATE_LIMIT_RPS"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || v <= 0 {
			return config{}, fmt.Errorf("RATE_LIMIT_RPS must be a positive number")
		}
		cfg.rateLimitRPS = v
		cfg.rateLimitBurst = int(v)
		if cfg.rateLimitBurst < 1 {
			cfg.rateLimitBurst = 1
		}
	}

	if raw := os.Getenv("CACHE_SIZE_MB"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return config{}, fmt.Errorf("CACHE_SIZE_MB must be a positive integer")
		}
		cfg.cacheSizeMB = v
	}

	return cfg, nil
}

func newReadyHandler(conn interface{ GetState() connectivity.State }) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		state := conn.GetState()
		if state == connectivity.TransientFailure || state == connectivity.Shutdown {
			handler.WriteJSONError(w, http.StatusServiceUnavailable, "geo service unavailable")
			return
		}

		handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
