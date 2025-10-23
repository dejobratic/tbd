package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config captures runtime configuration for the API service.
type Config struct {
	HTTPPort      int
	DatabaseURL   string
	KafkaBrokers  []string
	ServiceName   string
	MetricsPath   string
	ShutdownGrace int
}

const (
	defaultPort          = 8080
	defaultServiceName   = "tbd-api"
	defaultMetricsPath   = "/metrics"
	defaultShutdownGrace = 15
)

// Load reads configuration from environment variables, applying defaults when needed.
func Load() (*Config, error) {
	port := defaultPort
	if value, ok := os.LookupEnv("API_HTTP_PORT"); ok {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid API_HTTP_PORT: %w", err)
		}
		port = parsed
	}

	shutdownGrace := defaultShutdownGrace
	if value, ok := os.LookupEnv("API_SHUTDOWN_GRACE_SECONDS"); ok {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid API_SHUTDOWN_GRACE_SECONDS: %w", err)
		}
		shutdownGrace = parsed
	}

	serviceName := defaultServiceName
	if value, ok := os.LookupEnv("API_SERVICE_NAME"); ok && value != "" {
		serviceName = value
	}

	metricsPath := defaultMetricsPath
	if value, ok := os.LookupEnv("API_METRICS_PATH"); ok && value != "" {
		metricsPath = value
	}

	databaseURL := os.Getenv("DATABASE_URL")

	brokers := []string{}
	if value, ok := os.LookupEnv("KAFKA_BROKERS"); ok && value != "" {
		brokers = strings.Split(value, ",")
	}

	return &Config{
		HTTPPort:      port,
		DatabaseURL:   databaseURL,
		KafkaBrokers:  brokers,
		ServiceName:   serviceName,
		MetricsPath:   metricsPath,
		ShutdownGrace: shutdownGrace,
	}, nil
}
