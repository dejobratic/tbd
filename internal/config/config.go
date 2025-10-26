package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config captures runtime configuration for the API service.
type Config struct {
	HTTPPort       int
	DatabaseURL    string
	AutoMigrate    bool
	MigrationsPath string
	KafkaBrokers   []string
	ServiceName    string
	MetricsPath    string
	ShutdownGrace  int
}

const (
	defaultPort           = 8080
	defaultServiceName    = "tbd-api"
	defaultMetricsPath    = "/metrics"
	defaultShutdownGrace  = 15
	defaultMigrationsPath = "migrations"
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

	migrationsPath := defaultMigrationsPath
	if value, ok := os.LookupEnv("MIGRATIONS_PATH"); ok && value != "" {
		migrationsPath = value
	}

	autoMigrate := true
	if value, ok := os.LookupEnv("AUTO_MIGRATE"); ok {
		autoMigrate = value == "true"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = buildDatabaseURL()
	}

	brokers := []string{}
	if value, ok := os.LookupEnv("KAFKA_BROKERS"); ok && value != "" {
		brokers = strings.Split(value, ",")
	}

	return &Config{
		HTTPPort:       port,
		DatabaseURL:    databaseURL,
		AutoMigrate:    autoMigrate,
		MigrationsPath: migrationsPath,
		KafkaBrokers:   brokers,
		ServiceName:    serviceName,
		MetricsPath:    metricsPath,
		ShutdownGrace:  shutdownGrace,
	}, nil
}

func buildDatabaseURL() string {
	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "5432")
	user := getEnvOrDefault("DB_USER", "postgres")
	password := getEnvOrDefault("DB_PASSWORD", "postgres")
	dbName := getEnvOrDefault("DB_NAME", "tbd")
	sslMode := getEnvOrDefault("DB_SSLMODE", "disable")

	maxConns := getEnvOrDefault("DB_MAX_CONNS", "25")
	minConns := getEnvOrDefault("DB_MIN_CONNS", "5")
	maxLifetime := getEnvOrDefault("DB_MAX_CONN_LIFETIME", "5m")

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&pool_max_conns=%s&pool_min_conns=%s&pool_max_conn_lifetime=%s",
		user, password, host, port, dbName, sslMode, maxConns, minConns, maxLifetime,
	)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
