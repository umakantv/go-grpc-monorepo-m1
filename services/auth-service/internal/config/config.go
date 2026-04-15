package config

import (
	"github.com/yourorg/monorepo/pkg/config"
)

// Config extends the base config with auth-service specific configuration
type Config struct {
	config.Config `mapstructure:",squash"`
	JWT           JWTConfig `mapstructure:"jwt"`
}

// JWTConfig contains JWT configuration
type JWTConfig struct {
	Secret          string `mapstructure:"secret"`
	AccessTokenTTL  int    `mapstructure:"access_token_ttl"`  // in seconds
	RefreshTokenTTL int    `mapstructure:"refresh_token_ttl"` // in seconds
	Issuer          string `mapstructure:"issuer"`
}

// Load loads configuration from file and environment variables
func Load(configPath string, configName string) (*Config, error) {
	baseCfg, err := config.Load(configPath, configName)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Config: *baseCfg,
		JWT: JWTConfig{
			Secret:          "your-super-secret-key-change-in-production",
			AccessTokenTTL:  3600,
			RefreshTokenTTL: 604800,
			Issuer:          "yourorg-monorepo",
		},
	}

	return cfg, nil
}
