package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Log          LoggingConfig      `mapstructure:"logging"`
	Etcd         EtcdConfig         `mapstructure:"etcd"`
	NginxServers []NginxServerGroup `mapstructure:"nginx_servers"`
	Sync         SyncConfig         `mapstructure:"sync"`
}

// LoggingConfig holds the logging configuration
type LoggingConfig struct {
	Level string `mapstructure:"level"`
}

// EtcdConfig holds the etcd client configuration.
type EtcdConfig struct {
	Endpoints []string `mapstructure:"endpoints"`
}

// NginxServerGroup represents a group of Nginx servers.
type NginxServerGroup struct {
	Group   string         `mapstructure:"group"`
	Servers []ServerConfig `mapstructure:"servers"`
}

// ServerConfig holds the configuration for a single server
type ServerConfig struct {
	Name            string           `mapstructure:"name"`
	Host            string           `mapstructure:"host"`
	Port            int              `mapstructure:"port"`
	User            string           `mapstructure:"user"`
	Auth            ServerAuthConfig `mapstructure:"auth"`
	NginxBinaryPath string           `mapstructure:"nginx_binary_path"`
	NginxConfigDir  string           `mapstructure:"nginx_config_dir"`
	CheckDir        string           `mapstructure:"check_dir"`
	TestCmd         string           `mapstructure:"test_cmd"`
	ReloadCmd       string           `mapstructure:"reload_cmd"`
	BackupDir       string           `mapstructure:"backup_dir"`
}

// ServerAuthConfig holds the authentication configuration for a server
type ServerAuthConfig struct {
	Method   string `mapstructure:"method"`
	KeyPath  string `mapstructure:"key_path"`
	Password string `mapstructure:"password"`
}

// Sync configuration
type SyncConfig struct {
	NginxSyncer NginxSyncer `mapstructure:"nginx_syncer"`
}

type NginxSyncer struct {
	KeyPrefix       string   `mapstructure:"key_prefix"`
	IntervalSeconds int      `mapstructure:"interval_seconds"`
	IgnorePatterns  []string `mapstructure:"ignore_patterns"`
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
	viper.SetDefault("sync.nginx_syncer.key_prefix", "/gitops-nginx-remote")
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
