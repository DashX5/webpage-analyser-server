package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"

	"github.com/webpage-analyser-server/internal/constants"
)

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig
	Cache     CacheConfig
	Analyzer  AnalyzerConfig
	Logging   LoggingConfig
	Metrics   MetricsConfig
	RateLimit RateLimitConfig
	CORS      CORSConfig
}

type ServerConfig struct {
	Port    int
	Timeout time.Duration
	Mode    string
}

type CacheConfig struct {
	Enabled bool
	TTL     time.Duration
	Redis   RedisConfig
}

type RedisConfig struct {
	Host     string
	Port     int
	DB       int
	Password string
}

type AnalyzerConfig struct {
	MaxLinks     int
	LinkTimeout  time.Duration
	MaxWorkers   int
	MaxRedirects int
}

type LoggingConfig struct {
	Level  string
	Format string
}

type MetricsConfig struct {
	Enabled    bool
	Prometheus PrometheusConfig
}

type PrometheusConfig struct {
	Buckets []float64
}

type RateLimitConfig struct {
	Enabled           bool
	RequestsPerMinute float64
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}


func Load(configPath string, env string) (*Config, error) {
	if env == "" {
		env = constants.EnvDevelopment 
	}

	viper.SetConfigName(env)
	viper.SetConfigType(constants.ConfigFileType)
	viper.AddConfigPath(configPath)
	viper.AutomaticEnv()

	
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}


func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", constants.DefaultServerPort)
	viper.SetDefault("server.timeout", constants.DefaultServerTimeout)
	viper.SetDefault("server.mode", constants.DefaultServerMode)

	// Cache defaults
	viper.SetDefault("cache.enabled", true)
	viper.SetDefault("cache.ttl", constants.DefaultCacheTTL)
	viper.SetDefault("cache.redis.host", constants.DefaultRedisHost)
	viper.SetDefault("cache.redis.port", constants.DefaultRedisPort)
	viper.SetDefault("cache.redis.db", constants.DefaultRedisDB)
	viper.SetDefault("cache.redis.password", "")

	// Analyzer defaults
	viper.SetDefault("analyzer.max_links", constants.DefaultMaxLinks)
	viper.SetDefault("analyzer.link_timeout", constants.DefaultLinkTimeout)
	viper.SetDefault("analyzer.max_workers", constants.DefaultMaxWorkers)
	viper.SetDefault("analyzer.max_redirects", constants.DefaultMaxRedirects)

	// Rate limit defaults
	viper.SetDefault("rate_limit.enabled", constants.DefaultRateLimitEnabled)
	viper.SetDefault("rate_limit.requests_per_minute", constants.DefaultRequestsPerMinute)

	// CORS defaults
	viper.SetDefault("cors.allowed_origins", []string{"*"})
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "OPTIONS"})
	viper.SetDefault("cors.allowed_headers", []string{"Origin", "Content-Type", "Accept"})

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "console")

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.prometheus.buckets", []float64{0.1, 0.5, 1, 2, 5, 10})
} 