package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/webpage-analyser-server/internal/config"
	"github.com/webpage-analyser-server/internal/handlers"
	"github.com/webpage-analyser-server/internal/metrics"
	"github.com/webpage-analyser-server/internal/middleware"
	"github.com/webpage-analyser-server/internal/router"
	"github.com/webpage-analyser-server/internal/services"
)


type App struct {
	config      *config.Config
	logger      *zap.Logger
	metrics     *metrics.Metrics
	cache       *services.Cache
	analyzer    *services.Analyzer
	handler     *handlers.AnalyzeHandler
	rateLimiter *middleware.RateLimiter
	router      *router.Router
	server      *http.Server
}


func New(configPath string, env string) (*App, error) {
	
	cfg, err := config.Load(configPath, env)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	
	logger, err := initLogger(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	
	m := metrics.New()

	
	var cache *services.Cache
	if cfg.Cache.Enabled {
		cache, err = services.NewCache(cfg, logger, m)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize cache: %w", err)
		}
		logger.Info("Cache enabled", zap.String("host", cfg.Cache.Redis.Host), zap.Int("port", cfg.Cache.Redis.Port))
	} else {
		cache = services.NewNoOpCache(logger)
		logger.Info("Cache disabled - using no-op cache")
	}

	
	analyzer := services.NewAnalyzer(cfg, logger, m, cache)

	
	handler := handlers.NewAnalyzeHandler(logger, analyzer)

	
	rateLimiter := middleware.NewRateLimiter()

	
	r := router.New(cfg, logger, m, handler, rateLimiter)

	
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r.Handler(),
		ReadTimeout:  cfg.Server.Timeout,
		WriteTimeout: cfg.Server.Timeout,
	}

	return &App{
		config:      cfg,
		logger:      logger,
		metrics:     m,
		cache:       cache,
		analyzer:    analyzer,
		handler:     handler,
		rateLimiter: rateLimiter,
		router:      r,
		server:      srv,
	}, nil
}


func (a *App) Start() error {
	
	go func() {
		a.logger.Info("Starting server...",
			zap.String("address", a.server.Addr),
			zap.String("mode", a.config.Server.Mode),
		)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	return nil
}


func (a *App) Stop() error {
	a.logger.Info("Shutting down server...")

	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	
	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	
	if err := a.cache.Close(); err != nil {
		return fmt.Errorf("cache shutdown failed: %w", err)
	}

	
	if err := a.logger.Sync(); err != nil {
		return fmt.Errorf("logger sync failed: %w", err)
	}

	return nil
}


func (a *App) WaitForSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

func initLogger(cfg *config.Config) (*zap.Logger, error) {
	var config zap.Config

	
	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(cfg.Logging.Level)); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	if cfg.Logging.Format == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	config.Level = level
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
} 