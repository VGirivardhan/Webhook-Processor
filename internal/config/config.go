package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the webhook processor
type Config struct {
	Database   DatabaseConfig   `json:"database"`
	HTTPClient HTTPClientConfig `json:"http_client"`
	HTTPServer HTTPServerConfig `json:"http_server"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	User            string        `json:"user"`
	Password        string        `json:"password"`
	DBName          string        `json:"db_name"`
	SSLMode         string        `json:"ssl_mode"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}

// WorkerConfig holds configuration for a specific retry level worker
type WorkerConfig struct {
	RetryLevel   int           `json:"retry_level"`
	PollInterval time.Duration `json:"poll_interval"`
	Description  string        `json:"description"`
}

// WorkerPoolConfig holds configuration for the worker pool
type WorkerPoolConfig struct {
	Workers []WorkerConfig `json:"workers"`
}

// HTTPClientConfig holds HTTP client configuration for external webhook requests
type HTTPClientConfig struct {
	Timeout         time.Duration `json:"timeout"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	IdleConnTimeout time.Duration `json:"idle_conn_timeout"`
}

// HTTPServerConfig holds HTTP server configuration for our API server
type HTTPServerConfig struct {
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	_ = godotenv.Load()
	config := &Config{
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "root"),
			DBName:          getEnv("DB_NAME", "webhook_processor"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		HTTPClient: HTTPClientConfig{
			Timeout:         getEnvAsDuration("HTTP_CLIENT_TIMEOUT", 30*time.Second),
			MaxIdleConns:    getEnvAsInt("HTTP_CLIENT_MAX_IDLE_CONNS", 100),
			IdleConnTimeout: getEnvAsDuration("HTTP_CLIENT_IDLE_CONN_TIMEOUT", 90*time.Second),
		},
		HTTPServer: HTTPServerConfig{
			Port:         getEnvAsInt("API_PORT", 8080),
			ReadTimeout:  getEnvAsDuration("HTTP_SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvAsDuration("HTTP_SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvAsDuration("HTTP_SERVER_IDLE_TIMEOUT", 120*time.Second),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	if c.HTTPClient.Timeout <= 0 {
		return fmt.Errorf("HTTP client timeout must be positive")
	}
	if c.HTTPServer.Port <= 0 || c.HTTPServer.Port > 65535 {
		return fmt.Errorf("HTTP server port must be between 1 and 65535")
	}
	return nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s timezone=UTC",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetDefaultWorkerPoolConfig returns the default configuration with 3 level-0 workers and other retry levels
func GetDefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		Workers: []WorkerConfig{
			// 3 dedicated workers for level 0 (immediate processing)
			// These workers will compete for level 0 webhooks using SELECT FOR UPDATE SKIP LOCKED
			{
				RetryLevel:   0,
				PollInterval: 5 * time.Second,
				Description:  "Level 0 Worker #1 - Immediate webhook attempts",
			},
			{
				RetryLevel:   0,
				PollInterval: 5 * time.Second,
				Description:  "Level 0 Worker #2 - Immediate webhook attempts",
			},
			{
				RetryLevel:   0,
				PollInterval: 5 * time.Second,
				Description:  "Level 0 Worker #3 - Immediate webhook attempts",
			},
			// Single workers for higher retry levels (less frequent polling)
			{
				RetryLevel:   1,
				PollInterval: 30 * time.Second, // 1st retry - after 1 minute
				Description:  "Level 1 Worker - First retry attempts (1 min delay)",
			},
			{
				RetryLevel:   2,
				PollInterval: 2 * time.Minute, // 2nd retry - after 5 minutes
				Description:  "Level 2 Worker - Second retry attempts (5 min delay)",
			},
			{
				RetryLevel:   3,
				PollInterval: 5 * time.Minute, // 3rd retry - after 10 minutes
				Description:  "Level 3 Worker - Third retry attempts (10 min delay)",
			},
			{
				RetryLevel:   4,
				PollInterval: 15 * time.Minute, // 4th retry - after 30 minutes
				Description:  "Level 4 Worker - Fourth retry attempts (30 min delay)",
			},
			{
				RetryLevel:   5,
				PollInterval: 30 * time.Minute, // 5th retry - after 1 hour
				Description:  "Level 5 Worker - Fifth retry attempts (1 hour delay)",
			},
			{
				RetryLevel:   6,
				PollInterval: 60 * time.Minute, // 6th retry - after 2 hours
				Description:  "Level 6 Worker - Final retry attempts (2 hour delay)",
			},
		},
	}
}
