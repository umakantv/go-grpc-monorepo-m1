package config

import (
	"github.com/yourorg/monorepo/pkg/config"
)

// Config extends the base config with user-api specific configuration
type Config struct {
	config.Config `mapstructure:",squash"`
	Auth          AuthConfig `mapstructure:"auth"`
}

// AuthConfig contains auth service client configuration
type AuthConfig struct {
	ServiceAddr string `mapstructure:"service_addr"`
}
