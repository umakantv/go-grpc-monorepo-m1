package config

import (
	"github.com/yourorg/monorepo/pkg/config"
)

// Config extends the base config with file-service specific configuration
type Config struct {
	config.Config `mapstructure:",squash"`
	S3            S3Config `mapstructure:"s3"`
}

// S3Config contains AWS S3 configuration
type S3Config struct {
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey  string `mapstructure:"secret_access_key"`
	Endpoint        string `mapstructure:"endpoint"`          // optional, for S3-compatible stores (MinIO, etc.)
	ForcePathStyle  bool   `mapstructure:"force_path_style"`  // required for MinIO / local dev
	SignedURLExpiry int    `mapstructure:"signed_url_expiry"` // seconds, default 900 (15 min)
}

// Load loads configuration from file and environment variables
func Load(configPath string, configName string) (*Config, error) {
	baseCfg, err := config.Load(configPath, configName)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Config: *baseCfg,
		S3: S3Config{
			Region:         "us-east-1",
			SignedURLExpiry: 900,
		},
	}

	return cfg, nil
}