package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	DBPath  string `yaml:"db_path"`
	LogPath string `yaml:"log_path"`
	PidPath string `yaml:"pid_path"`
	Cleanup struct {
		RetainDoneDays int `yaml:"retain_done_days"`
	} `yaml:"cleanup"`
	DefaultTimeout string `yaml:"default_timeout"`
}

func Default() *Config {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".taskdeck")
	return &Config{
		Port:    10086,
		DBPath:  filepath.Join(base, "taskdeck.db"),
		LogPath: filepath.Join(base, "taskdeck.log"),
		PidPath: filepath.Join(base, "taskdeck.pid"),
		Cleanup: struct {
			RetainDoneDays int `yaml:"retain_done_days"`
		}{RetainDoneDays: 30},
		DefaultTimeout: "30m",
	}
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".taskdeck"), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func Load() (*Config, error) {
	cfg := Default()

	cfgPath, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}
