package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Listen    string          `mapstructure:"listen"`
	Providers ProvidersConfig `mapstructure:"providers"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Log       LogConfig       `mapstructure:"log"`
	Recording RecordingConfig `mapstructure:"recording"`
}

// ProvidersConfig holds per-provider settings.
type ProvidersConfig struct {
	OpenAI    ProviderConfig `mapstructure:"openai"`
	Anthropic ProviderConfig `mapstructure:"anthropic"`
}

// ProviderConfig holds settings for a single upstream provider.
type ProviderConfig struct {
	Upstream string `mapstructure:"upstream"`
	Enabled  bool   `mapstructure:"enabled"`
}

// StorageConfig holds database settings.
type StorageConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// RecordingConfig holds async recording pipeline settings.
type RecordingConfig struct {
	BufferSize int `mapstructure:"buffer_size"`
	Workers    int `mapstructure:"workers"`
}

// Load reads configuration from the given file path, environment variables,
// and defaults.
func Load(path string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("listen", ":8787")
	v.SetDefault("providers.openai.upstream", "https://api.openai.com")
	v.SetDefault("providers.openai.enabled", true)
	v.SetDefault("providers.anthropic.upstream", "https://api.anthropic.com")
	v.SetDefault("providers.anthropic.enabled", true)
	v.SetDefault("storage.driver", "sqlite")
	v.SetDefault("storage.dsn", "data/agentledger.db")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("recording.buffer_size", 10000)
	v.SetDefault("recording.workers", 4)

	// Environment variables: AGENTLEDGER_LISTEN, AGENTLEDGER_STORAGE_DSN, etc.
	v.SetEnvPrefix("AGENTLEDGER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Config file
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("agentledger")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("$HOME/.config/agentledger")
		v.AddConfigPath("/etc/agentledger")
	}

	if err := v.ReadInConfig(); err != nil {
		// Missing config file is fine when using defaults + env vars,
		// but an explicit path must exist.
		if path != "" {
			return nil, fmt.Errorf("reading config %s: %w", path, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	return &cfg, nil
}
