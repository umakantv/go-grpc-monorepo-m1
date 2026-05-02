package config

import (
	"github.com/yourorg/monorepo/pkg/config"
)

// Config extends the base config with notification-service specific configuration.
type Config struct {
	config.Config `mapstructure:",squash"`
	Firebase      FirebaseConfig `mapstructure:"firebase"`
}

// FirebaseConfig contains Firebase credentials for FCM.
type FirebaseConfig struct {
	CredentialsFile string `mapstructure:"credentials_file"` // path to service account JSON
	ProjectID       string `mapstructure:"project_id"`
}

// Load loads configuration from file and environment variables.
func Load(configPath string, configName string) (*Config, error) {
	baseCfg, err := config.Load(configPath, configName)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Config: *baseCfg,
	}

	return cfg, nil
}