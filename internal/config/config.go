package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	API          APIConfig          `mapstructure:"api"`
	Log          LoggingConfig      `mapstructure:"logging"`
	Etcd         EtcdConfig         `mapstructure:"etcd"`
	NginxServers []NginxServerGroup `mapstructure:"nginx_servers"`
	Sync         SyncConfig         `mapstructure:"sync"`
	Git          GitConfig          `mapstructure:"git"`
}

// APIConfig holds the API server configuration
type APIConfig struct {
	Listen       string   `mapstructure:"listen"`
	AllowOrigins []string `mapstructure:"allow_origins"`
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
	BackupDir       string           `mapstructure:"backup_dir"`
	// TestCmd         string           `mapstructure:"test_cmd"`
	// ReloadCmd       string           `mapstructure:"reload_cmd"`
}

// ServerAuthConfig holds the authentication configuration for a server
type ServerAuthConfig struct {
	Method   string `mapstructure:"method"`
	KeyPath  string `mapstructure:"key_path"`
	Password string `mapstructure:"password"`
}

// Sync configuration
type SyncConfig struct {
	NginxSyncer   NginxSyncer   `mapstructure:"nginx_syncer"`
	GitSyncer     GitSyncer     `mapstructure:"git_syncer"`
	PreviewSyncer PreviewSyncer `mapstructure:"preview_syncer"`
}

type NginxSyncer struct {
	KeyPrefix       string   `mapstructure:"key_prefix"`
	IntervalSeconds int      `mapstructure:"interval_seconds"`
	IgnorePatterns  []string `mapstructure:"ignore_patterns"`
}

type GitSyncer struct {
	KeyPrefix       string   `mapstructure:"key_prefix"`
	IntervalSeconds int      `mapstructure:"interval_seconds"`
	IgnorePatterns  []string `mapstructure:"ignore_patterns"`
}

type PreviewSyncer struct {
	KeyPrefix       string   `mapstructure:"key_prefix"`
	IntervalSeconds int      `mapstructure:"interval_seconds"`
	IgnorePatterns  []string `mapstructure:"ignore_patterns"`
}

// GitConfig holds the Git repository configuration
type GitConfig struct {
	RepoURL    string        `mapstructure:"repo_url"`
	RepoPath   string        `mapstructure:"repo_path"`
	Branch     string        `mapstructure:"branch"`
	RemoteName string        `mapstructure:"remote_name"`
	SyncMode   string        `mapstructure:"sync_mode"`
	Auth       GitAuthConfig `mapstructure:"auth"`
	Poll       GitPollConfig `mapstructure:"poll"`
}

// GitAuthConfig holds the git authentication configuration
type GitAuthConfig struct {
	Type           string `mapstructure:"type"` // "basic", "ssh", "none"
	Username       string `mapstructure:"username"`
	Password       string `mapstructure:"password"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
}

// GitPollConfig holds the git polling configuration
type GitPollConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	IntervalSeconds int  `mapstructure:"interval_seconds"`
}

// LoadConfig loads the configuration from multiple files
func LoadConfig() (*Config, error) {
	// 1. Load main config.yaml
	vMain := viper.New()
	vMain.SetConfigName("config")
	vMain.SetConfigType("yaml")
	vMain.AddConfigPath("./configs")
	vMain.AddConfigPath(".")
	vMain.AutomaticEnv()
	vMain.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values for main config
	vMain.SetDefault("api.listen", ":8080")
	vMain.SetDefault("sync.nginx_syncer.key_prefix", "/gitops-nginx-remote")
	vMain.SetDefault("sync.git_syncer.key_prefix", "/gitops-nginx")
	vMain.SetDefault("sync.preview_syncer.key_prefix", "/gitops-nginx-preview")
	vMain.SetDefault("logging.level", "info")
	vMain.SetDefault("etcd.endpoints", []string{"localhost:2379"})

	if err := vMain.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read main config file: %w", err)
		}
	}

	var config Config
	if err := vMain.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal main config: %w", err)
	}

	// 2. Load servers.yaml (standalone file for nginx_servers)
	vServers := viper.New()
	vServers.SetConfigName("servers")
	vServers.SetConfigType("yaml")
	vServers.AddConfigPath("./configs")
	vServers.AddConfigPath(".")

	if err := vServers.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read servers config file: %w", err)
		}
	} else {
		var serverGroups struct {
			NginxServers []NginxServerGroup `mapstructure:"nginx_servers"`
		}
		if err := vServers.Unmarshal(&serverGroups); err != nil {
			return nil, fmt.Errorf("failed to unmarshal servers config: %w", err)
		}
		// Merge or set nginx_servers if present in servers.yaml
		if len(serverGroups.NginxServers) > 0 {
			config.NginxServers = serverGroups.NginxServers
		}
	}

	return &config, nil
}
