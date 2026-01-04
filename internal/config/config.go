package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the cart service
type Config struct {
	// Server settings
	ServerPort string

	// Redis settings
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// Cart settings
	CartTTL time.Duration

	// OAuth2 settings
	OAuth2Enabled   bool
	OAuth2IssuerURI string
	OAuth2ClientID  string

	// RabbitMQ settings
	RabbitMQHost     string
	RabbitMQPort     string
	RabbitMQVHost    string
	RabbitMQUsername string
	RabbitMQPassword string
	RabbitMQUseTLS   bool

	// Vault settings
	VaultEnabled bool
	VaultAddr    string
	VaultToken   string
	VaultRole    string

	// Logging
	LogLevel string

	// Rate limiting
	RateLimitRPS   int
	RateLimitBurst int
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		// Server
		ServerPort: getEnv("SERVER_PORT", "8083"),

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		// Cart
		CartTTL: getEnvAsDuration("CART_TTL", 168*time.Hour), // 7 days

		// OAuth2
		OAuth2Enabled:   getEnvAsBool("OAUTH2_ENABLED", false),
		OAuth2IssuerURI: getEnv("OAUTH2_ISSUER_URI", ""),
		OAuth2ClientID:  getEnv("OAUTH2_CLIENT_ID", "cart-service"),

		// RabbitMQ
		RabbitMQHost:     getEnv("RABBITMQ_HOST", "localhost"),
		RabbitMQPort:     getEnv("RABBITMQ_PORT", "5672"),
		RabbitMQVHost:    getEnv("RABBITMQ_VHOST", "/"),
		RabbitMQUsername: getEnv("RABBITMQ_USERNAME", "guest"),
		RabbitMQPassword: getEnv("RABBITMQ_PASSWORD", "guest"),
		RabbitMQUseTLS:   getEnvAsBool("RABBITMQ_USE_TLS", false),

		// Vault
		VaultEnabled: getEnvAsBool("VAULT_ENABLED", false),
		VaultAddr:    getEnv("VAULT_ADDR", "http://localhost:8200"),
		VaultToken:   getEnv("VAULT_TOKEN", ""),
		VaultRole:    getEnv("VAULT_ROLE", "cart-service"),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),

		// Rate limiting
		RateLimitRPS:   getEnvAsInt("RATE_LIMIT_RPS", 100),
		RateLimitBurst: getEnvAsInt("RATE_LIMIT_BURST", 50),
	}
}

// RedisAddr returns the Redis address in host:port format
func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

// RabbitMQAddr returns the RabbitMQ address in host:port format
func (c *Config) RabbitMQAddr() string {
	return c.RabbitMQHost + ":" + c.RabbitMQPort
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
