package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config captures runtime configuration for the API service.
type Config struct {
	HTTP      HTTPConfig
	Database  DatabaseConfig
	Kafka     KafkaConfig
	Telemetry TelemetryConfig
	Service   ServiceConfig
}

type HTTPConfig struct {
	Port          int
	MetricsPath   string
	ShutdownGrace int
}

type DatabaseConfig struct {
	URL            string
	AutoMigrate    bool
	MigrationsPath string
}

type KafkaConfig struct {
	Brokers []string
}

type TelemetryConfig struct {
	LogLevel      string
	OTelEndpoint  string
	EnableTracing bool
	EnableMetrics bool
	SampleRate    float64
}

type ServiceConfig struct {
	Name        string
	Version     string
	Environment string
}

const (
	defaultHTTPPort       = 8080
	defaultMetricsPath    = "/metrics"
	defaultShutdownGrace  = 15
	defaultMigrationsPath = "migrations"
	defaultAutoMigrate    = true
	defaultServiceName    = "tbd-api"
	defaultServiceVersion = "0.1.0"
	defaultEnvironment    = "development"
	defaultLogLevel       = "info"
	defaultOTelSampleRate = 1.0
)

// Load reads configuration from environment variables, applying defaults when needed.
func Load() (*Config, error) {
	httpCfg, err := loadHTTPConfig()
	if err != nil {
		return nil, fmt.Errorf("loading HTTP config: %w", err)
	}

	dbCfg := loadDatabaseConfig()
	kafkaCfg := loadKafkaConfig()
	telCfg, err := loadTelemetryConfig()
	if err != nil {
		return nil, fmt.Errorf("loading telemetry config: %w", err)
	}

	serviceCfg := loadServiceConfig()

	return &Config{
		HTTP:      httpCfg,
		Database:  dbCfg,
		Kafka:     kafkaCfg,
		Telemetry: telCfg,
		Service:   serviceCfg,
	}, nil
}

func loadHTTPConfig() (HTTPConfig, error) {
	port := defaultHTTPPort
	if value, ok := os.LookupEnv("API_HTTP_PORT"); ok {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return HTTPConfig{}, fmt.Errorf("invalid API_HTTP_PORT: %w", err)
		}
		port = parsed
	}

	shutdownGrace := defaultShutdownGrace
	if value, ok := os.LookupEnv("API_SHUTDOWN_GRACE_SECONDS"); ok {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return HTTPConfig{}, fmt.Errorf("invalid API_SHUTDOWN_GRACE_SECONDS: %w", err)
		}
		shutdownGrace = parsed
	}

	metricsPath := getEnvOrDefault("API_METRICS_PATH", defaultMetricsPath)

	return HTTPConfig{
		Port:          port,
		MetricsPath:   metricsPath,
		ShutdownGrace: shutdownGrace,
	}, nil
}

func loadDatabaseConfig() DatabaseConfig {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = buildDatabaseURL()
	}

	autoMigrate := defaultAutoMigrate
	if value, ok := os.LookupEnv("AUTO_MIGRATE"); ok {
		autoMigrate = value == "true"
	}

	migrationsPath := getEnvOrDefault("MIGRATIONS_PATH", defaultMigrationsPath)

	return DatabaseConfig{
		URL:            databaseURL,
		AutoMigrate:    autoMigrate,
		MigrationsPath: migrationsPath,
	}
}

func loadKafkaConfig() KafkaConfig {
	var brokers []string
	if value, ok := os.LookupEnv("KAFKA_BROKERS"); ok && value != "" {
		brokers = strings.Split(value, ",")
	}

	return KafkaConfig{
		Brokers: brokers,
	}
}

func loadTelemetryConfig() (TelemetryConfig, error) {
	logLevel := getEnvOrDefault("LOG_LEVEL", defaultLogLevel)
	otelEndpoint := getEnvOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	enableTracing := getBoolEnv("OTEL_ENABLE_TRACING", true)
	enableMetrics := getBoolEnv("OTEL_ENABLE_METRICS", true)

	sampleRate := defaultOTelSampleRate
	if value, ok := os.LookupEnv("OTEL_SAMPLE_RATE"); ok {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return TelemetryConfig{}, fmt.Errorf("invalid OTEL_SAMPLE_RATE: %w", err)
		}
		sampleRate = parsed
	}

	return TelemetryConfig{
		LogLevel:      logLevel,
		OTelEndpoint:  otelEndpoint,
		EnableTracing: enableTracing,
		EnableMetrics: enableMetrics,
		SampleRate:    sampleRate,
	}, nil
}

func loadServiceConfig() ServiceConfig {
	return ServiceConfig{
		Name:        getEnvOrDefault("API_SERVICE_NAME", defaultServiceName),
		Version:     getEnvOrDefault("SERVICE_VERSION", defaultServiceVersion),
		Environment: getEnvOrDefault("ENVIRONMENT", defaultEnvironment),
	}
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

func getBoolEnv(key string, defaultValue bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		return value == "true"
	}
	return defaultValue
}
