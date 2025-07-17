package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/webpage-analyser-server/internal/config"
	"github.com/webpage-analyser-server/internal/constants"
	"github.com/webpage-analyser-server/internal/metrics"
	"github.com/webpage-analyser-server/internal/models"
)


type Cache struct {
	client  *redis.Client
	logger  *zap.Logger
	metrics *metrics.Metrics
	ttl     time.Duration
}

// NoOpCache implements the CacheInterface but doesn't cache anything
type NoOpCache struct {
	logger *zap.Logger
}

// NewNoOpCache creates a new no-op cache that doesn't actually cache anything
func NewNoOpCache(logger *zap.Logger) *Cache {
	return &Cache{
		client:  nil,
		logger:  logger,
		metrics: nil,
		ttl:     0,
	}
}


func NewCache(cfg *config.Config, logger *zap.Logger, metrics *metrics.Metrics) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Cache.Redis.Host, cfg.Cache.Redis.Port),
		DB:       cfg.Cache.Redis.DB,
		Password: cfg.Cache.Redis.Password,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), constants.CacheConnectionTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Cache{
		client:  client,
		logger:  logger,
		metrics: metrics,
		ttl:     cfg.Cache.TTL,
	}, nil
}

// Get retrieves cached analysis results
func (c *Cache) Get(ctx context.Context, url string) (*models.AnalyzeResponse, error) {
	// If this is a no-op cache (client is nil), always return cache miss
	if c.client == nil {
		c.logger.Debug("No-op cache: skipping get", zap.String("url", url))
		return nil, nil
	}

	data, err := c.client.Get(ctx, c.key(url)).Bytes()
	if err == redis.Nil {
		if c.metrics != nil {
			c.metrics.CacheMisses.Inc()
		}
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	var result models.AnalyzeResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	if c.metrics != nil {
		c.metrics.CacheHits.Inc()
	}
	c.logger.Debug("Cache hit", zap.String("url", url))
	return &result, nil
}

// Set stores analysis results in cache
func (c *Cache) Set(ctx context.Context, url string, result *models.AnalyzeResponse) error {
	// If this is a no-op cache (client is nil), do nothing
	if c.client == nil {
		c.logger.Debug("No-op cache: skipping set", zap.String("url", url))
		return nil
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := c.client.Set(ctx, c.key(url), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	// If this is a no-op cache (client is nil), do nothing
	if c.client == nil {
		c.logger.Debug("No-op cache: skipping close")
		return nil
	}
	return c.client.Close()
}

// key generates a cache key for a URL
func (c *Cache) key(url string) string {
	return fmt.Sprintf("webpage:%s", url)
} 