package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/webpage-analyser-server/internal/config"
	"github.com/webpage-analyser-server/internal/constants"
	"github.com/webpage-analyser-server/internal/handlers"
	"github.com/webpage-analyser-server/internal/metrics"
	"github.com/webpage-analyser-server/internal/middleware"
)


type Router struct {
	engine      *gin.Engine
	config      *config.Config
	logger      *zap.Logger
	metrics     *metrics.Metrics
	handler     *handlers.AnalyzeHandler
	rateLimiter *middleware.RateLimiter
}


func New(
	config *config.Config,
	logger *zap.Logger,
	metrics *metrics.Metrics,
	handler *handlers.AnalyzeHandler,
	rateLimiter *middleware.RateLimiter,
) *Router {
	if config.Server.Mode == "" {
		config.Server.Mode = constants.DefaultServerMode
	}
	gin.SetMode(config.Server.Mode)

	r := &Router{
		engine:      gin.New(),
		config:      config,
		logger:      logger,
		metrics:     metrics,
		handler:     handler,
		rateLimiter: rateLimiter,
	}

	r.setupMiddleware()
	r.setupRoutes()

	return r
}


func (r *Router) Handler() http.Handler {
	return r.engine
}

func (r *Router) setupMiddleware() {
	r.engine.Use(gin.Recovery())

	// Add request logging middleware
	r.engine.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		r.logger.Info("Request processed",
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("method", c.Request.Method),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		r.metrics.RequestDuration.WithLabelValues(fmt.Sprintf("%d", status)).Observe(latency.Seconds())
	})
}

func (r *Router) setupRoutes() {
	// Serve static files
	r.engine.Static("/static", "./web/static")
	r.engine.LoadHTMLGlob("web/templates/*")

	// Serve HTML form
	r.engine.GET("/", func(c *gin.Context) {
		c.HTML(constants.StatusOK, "index.html", nil)
	})

	// API routes
	api := r.engine.Group("/api/v1")
	{
		api.Use(r.rateLimiter.RateLimit())
		api.POST("/analyze", r.handler.Handle)
	}

	// Metrics endpoint
	r.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(constants.StatusOK, gin.H{"status": "ok"})
	})
} 