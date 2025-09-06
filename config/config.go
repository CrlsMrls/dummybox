package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Port        int    `mapstructure:"port"`
	LogLevel    string `mapstructure:"log-level"`
	MetricsPath string `mapstructure:"metrics-path"`
	TLSCertFile string `mapstructure:"tls-cert-file"`
	TLSKeyFile  string `mapstructure:"tls-key-file"`
	AuthToken   string `mapstructure:"auth-token"`
}

// New creates a new Config object
func New() (*Config, error) {
	v := viper.New()

	// Set default values
	v.SetDefault("port", 8080)
	v.SetDefault("log-level", "info")
	v.SetDefault("metrics-path", "/metrics")
	v.SetDefault("tls-cert-file", "")
	v.SetDefault("tls-key-file", "")
	v.SetDefault("auth-token", "")

	// Define command-line flags
	pflag.Int("port", 8080, "Listening port")
	pflag.String("log-level", "info", "Logging level (debug, info, warning, error)")
	pflag.String("metrics-path", "/metrics", "Metrics endpoint path")
	pflag.String("tls-cert-file", "", "Path to TLS certificate file")
	pflag.String("tls-key-file", "", "Path to TLS key file")
	pflag.String("auth-token", "", "Authentication token for command endpoints")
	pflag.String("config-file", "", "Path to JSON config file. Can also be set with DUMMYBOX_CONFIG_FILE env var.")
	pflag.Parse()
	v.BindPFlags(pflag.CommandLine)

	// Set up environment variable binding
	v.SetEnvPrefix("DUMMYBOX")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	// Handle config file
	if configFile := v.GetString("config-file"); configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func getEnvOrDefault(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

// DefaultConfig returns a Config struct with default values.
func DefaultConfig() *Config {
	return &Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		TLSCertFile: "",
		TLSKeyFile:  "",
		AuthToken:   "",
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate LogLevel
	validLogLevels := []string{"debug", "info", "warn", "error"}
	isValidLogLevel := false
	for _, level := range validLogLevels {
		if c.LogLevel == level {
			isValidLogLevel = true
			break
		}
	}
	if !isValidLogLevel {
		return fmt.Errorf("invalid log-level: %s, must be one of %v", c.LogLevel, validLogLevels)
	}

	// Validate Port
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d, must be between 1 and 65535", c.Port)
	}

	return nil
}
