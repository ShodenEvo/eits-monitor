package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Queue struct {
	Enabled        bool `json:"enabled"`
	MaximumRecords int  `json:"maximum_records"`
}

type Logging struct {
	Level         string `json:"level"`
	MaximumSizeMB int64  `json:"maximum_size_mb"`
	MaximumFiles  int    `json:"maximum_files"`
}

type Config struct {
	ServerURL                 string  `json:"server_url"`
	EnrollmentToken           string  `json:"enrollment_token,omitempty"`
	DeviceName                string  `json:"device_name"`
	CollectionIntervalSeconds int     `json:"collection_interval_seconds"`
	RequestTimeoutSeconds     int     `json:"request_timeout_seconds"`
	AllowInsecureHTTP         bool    `json:"allow_insecure_http"`
	SkipTLSVerify             bool    `json:"skip_tls_verify"`
	StateDirectory            string  `json:"state_directory"`
	LogDirectory              string  `json:"log_directory"`
	Queue                     Queue   `json:"queue"`
	Logging                   Logging `json:"logging"`
}

func DefaultPath() string {
	if runtime.GOOS == "windows" {
		base := os.Getenv("ProgramData")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "EITS", "Agent", "config.json")
	}
	return "/etc/eits-agent/config.json"
}

func Defaults() Config {
	state, logs := "/var/lib/eits-agent", "/var/log/eits-agent"
	if runtime.GOOS == "windows" {
		base := os.Getenv("ProgramData")
		if base == "" {
			base = `C:\ProgramData`
		}
		state = filepath.Join(base, "EITS", "Agent")
		logs = filepath.Join(state, "logs")
	}
	return Config{
		CollectionIntervalSeconds: 30,
		RequestTimeoutSeconds:     15,
		StateDirectory:            state,
		LogDirectory:              logs,
		Queue:                     Queue{Enabled: true, MaximumRecords: 2880},
		Logging:                   Logging{Level: "info", MaximumSizeMB: 10, MaximumFiles: 5},
	}
}

func Load(path string) (Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	cfg.ServerURL = strings.TrimRight(strings.TrimSpace(cfg.ServerURL), "/")
	if cfg.ServerURL == "" {
		return cfg, errors.New("server_url is required")
	}
	if strings.HasPrefix(strings.ToLower(cfg.ServerURL), "http://") && !cfg.AllowInsecureHTTP {
		return cfg, errors.New("HTTP server URL rejected; set allow_insecure_http=true only for trusted development networks")
	}
	if cfg.CollectionIntervalSeconds < 10 {
		cfg.CollectionIntervalSeconds = 10
	}
	if cfg.RequestTimeoutSeconds < 3 {
		cfg.RequestTimeoutSeconds = 3
	}
	if cfg.Queue.MaximumRecords < 1 {
		cfg.Queue.MaximumRecords = 2880
	}
	if cfg.Logging.MaximumSizeMB < 1 {
		cfg.Logging.MaximumSizeMB = 10
	}
	if cfg.Logging.MaximumFiles < 1 {
		cfg.Logging.MaximumFiles = 5
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
