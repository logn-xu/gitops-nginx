package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Log  LoggingConfig `mapstructure:"logging"`
	Etcd EtcdConfig
}

// LoggingConfig holds the logging configuration
type LoggingConfig struct {
	Level string
}

// EtcdConfig holds the etcd client configuration.
type EtcdConfig struct {
	Endpoints []string `mapstructure:"endpoints"`
}

// LoadConfig loads the configuration from a file
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("etcd.endpoints", []string{"localhost:2379"})

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found: %w", err)
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
