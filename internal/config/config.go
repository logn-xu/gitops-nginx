package config

import (
	"fmt"
	"net"
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
	Listen               string   `mapstructure:"listen"`
	AllowOrigins         []string `mapstructure:"allow_origins"`
	EnableEmbeddedServer bool     `mapstructure:"enable_embedded_server"`
}

// LoggingConfig holds the logging configuration
type LoggingConfig struct {
	Level     string        `mapstructure:"level"`
	AppLog    LogFileConfig `mapstructure:"app_log"`
	AccessLog LogFileConfig `mapstructure:"access_log"`
}

// LogFileConfig holds configuration for log files and rotation
type LogFileConfig struct {
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`    // megabytes
	MaxBackups int    `mapstructure:"max_backups"` // number of backups
	MaxAge     int    `mapstructure:"max_age"`     // days
	Compress   bool   `mapstructure:"compress"`    // whether to compress
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
	vMain.SetDefault("api.allow_origins", []string{"*"})
	vMain.SetDefault("api.enable_embedded_server", true)
	// set sync default values
	vMain.SetDefault("sync.nginx_syncer.key_prefix", "/gitops-nginx-remote")
	vMain.SetDefault("sync.git_syncer.key_prefix", "/gitops-nginx")
	vMain.SetDefault("sync.preview_syncer.key_prefix", "/gitops-nginx-preview")
	// set logging default values
	vMain.SetDefault("logging.level", "info")
	vMain.SetDefault("logging.app_log.filename", "logs/gitops-nginx.log")
	vMain.SetDefault("logging.app_log.max_size", 100)
	vMain.SetDefault("logging.app_log.max_backups", 5)
	vMain.SetDefault("logging.app_log.max_age", 30)
	vMain.SetDefault("logging.app_log.compress", true)
	vMain.SetDefault("logging.access_log.filename", "logs/access.log")
	vMain.SetDefault("logging.access_log.max_size", 100)
	vMain.SetDefault("logging.access_log.max_backups", 5)
	vMain.SetDefault("logging.access_log.max_age", 30)
	vMain.SetDefault("logging.access_log.compress", true)
	// set etcd default values
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

	// 2. Load and validate servers.yaml (standalone file for nginx_servers)
	serverGroups, err := ValidateServersConfig()
	if err != nil {
		return nil, err
	}
	config.NginxServers = serverGroups

	return &config, nil
}

// ValidateServersConfig validates the servers.yaml configuration
func ValidateServersConfig() ([]NginxServerGroup, error) {
	vServers := viper.New()
	vServers.SetConfigName("servers")
	vServers.SetConfigType("yaml")
	vServers.AddConfigPath("./configs")
	vServers.AddConfigPath(".")

	if err := vServers.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read servers config file: %w", err)
	}

	var serverGroups struct {
		NginxServers []NginxServerGroup `mapstructure:"nginx_servers"`
	}
	if err := vServers.Unmarshal(&serverGroups); err != nil {
		return nil, fmt.Errorf("failed to unmarshal servers config: %w", err)
	}

	if len(serverGroups.NginxServers) == 0 {
		return nil, fmt.Errorf("no nginx server groups defined in configuration")
	}

	var errs []string
	for _, group := range serverGroups.NginxServers {
		if group.Group == "" {
			errs = append(errs, "found group with empty name")
		}
		if len(group.Servers) == 0 {
			errs = append(errs, fmt.Sprintf("group '%s' has no servers defined", group.Group))
			continue
		}

		for _, server := range group.Servers {
			prefix := fmt.Sprintf("group '%s' server '%s'", group.Group, server.Name)
			if server.Host == "" {
				errs = append(errs, fmt.Sprintf("%s: host is empty", prefix))
			} else if net.ParseIP(server.Host) == nil {
				errs = append(errs, fmt.Sprintf("%s: host '%s' is not a valid IP address", prefix, server.Host))
			}
			if server.Port <= 0 || server.Port > 65535 {
				errs = append(errs, fmt.Sprintf("%s: port %d is invalid (must be 1-65535)", prefix, server.Port))
			}
			if server.User == "" {
				errs = append(errs, fmt.Sprintf("%s: user is empty", prefix))
			}
			if server.Auth.Method == "" {
				errs = append(errs, fmt.Sprintf("%s: auth method is not set", prefix))
			}
			if server.NginxConfigDir == "" {
				errs = append(errs, fmt.Sprintf("%s: nginx_config_dir is empty", prefix))
			}
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return serverGroups.NginxServers, nil
}
