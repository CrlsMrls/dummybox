package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
)

func TestNewConfig_Defaults(t *testing.T) {
	resetFlagsAndEnv(t)

	cfg, err := New()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Expected Port 8080, got %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected LogLevel 'info', got %s", cfg.LogLevel)
	}
	if cfg.MetricsPath != "/metrics" {
		t.Errorf("Expected MetricsPath '/metrics', got %s", cfg.MetricsPath)
	}
}

func TestNewConfig_Flags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd", "--port=9090", "--log-level=debug"}

	resetFlagsAndEnv(t)

	cfg, err := New()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("Expected Port 9090, got %d", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel 'debug', got %s", cfg.LogLevel)
	}
}

func TestNewConfig_EnvVars(t *testing.T) {
	resetFlagsAndEnv(t)

	t.Setenv("DUMMYBOX_PORT", "9091")
	t.Setenv("DUMMYBOX_LOG_LEVEL", "warn")

	cfg, err := New()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Port != 9091 {
		t.Errorf("Expected Port 9091, got %d", cfg.Port)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("Expected LogLevel 'warn', got %s", cfg.LogLevel)
	}
}

func TestNewConfig_ConfigFile(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	resetFlagsAndEnv(t)

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")

	configData := map[string]interface{}{
		"port":      9092,
		"log-level": "error",
	}
	fileContent, _ := json.Marshal(configData)
	os.WriteFile(configFile, fileContent, 0644)

	os.Args = []string{"cmd", "--config-file=" + configFile}

	cfg, err := New()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Port != 9092 {
		t.Errorf("Expected Port 9092, got %d", cfg.Port)
	}
	if cfg.LogLevel != "error" {
		t.Errorf("Expected LogLevel 'error', got %s", cfg.LogLevel)
	}
}

func TestNewConfig_Precedence(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	// 4. Flag (highest precedence)
	os.Args = []string{"cmd", "--port=3333"}

	resetFlagsAndEnv(t)

	// 2. Config File
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")
	configData := map[string]interface{}{"port": 1111}
	fileContent, _ := json.Marshal(configData)
	os.WriteFile(configFile, fileContent, 0644)
	t.Setenv("DUMMYBOX_CONFIG_FILE", configFile)

	// 3. Env Var
	t.Setenv("DUMMYBOX_PORT", "2222")

	cfg, err := New()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Flag should have the highest precedence
	if cfg.Port != 3333 {
		t.Errorf("Expected Port 3333 (from flag), got %d", cfg.Port)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		expectError bool
	}{
		{"valid", Config{Port: 8080, LogLevel: "info"}, false},
		{"invalid log level", Config{Port: 8080, LogLevel: "invalid"}, true},
		{"invalid port zero", Config{Port: 0, LogLevel: "info"}, true},
		{"invalid port negative", Config{Port: -1, LogLevel: "info"}, true},
		{"invalid port too high", Config{Port: 65536, LogLevel: "info"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.expectError {
				t.Errorf("Validate() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// resetFlagsAndEnv resets pflag and environment variables for a clean test run.
func resetFlagsAndEnv(t *testing.T) {
	t.Helper()
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	os.Clearenv()
}