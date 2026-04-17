package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the base configuration for all services
type Config struct {
	Service  ServiceConfig  `mapstructure:"service"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	HTTP     HTTPConfig     `mapstructure:"http"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Database DatabaseConfig `mapstructure:"database"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
}

// ServiceConfig contains service-level configuration
type ServiceConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
}

// GRPCConfig contains gRPC server configuration
type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

// HTTPConfig contains HTTP server configuration (for public services)
type HTTPConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // json or console
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

// Load loads configuration from file and environment variables
func Load(configPath string, configName string) (*Config, error) {
	v := viper.New()

	v.SetConfigName(configName)
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)
	v.AddConfigPath(".")

	// Set defaults
	setDefaults(v)

	// Enable environment variable override
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use defaults and env vars
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// LoadFromPath loads configuration from a specific path
func LoadFromPath(path string) (*Config, error) {
	return Load(".", path)
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("service.env", "development")
	v.SetDefault("grpc.port", 9090)
	v.SetDefault("http.enabled", false)
	v.SetDefault("http.port", 8080)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("database.enabled", false)
	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.port", 9091)
	v.SetDefault("metrics.path", "/metrics")
}
