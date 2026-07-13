package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	MusicPath string `yaml:"music_path"`
	DataPath  string `yaml:"data_path"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Token     string `yaml:"token"`
	Metadata  MetadataConfig `yaml:"metadata"`
}

// MetadataConfig controls background metadata enrichment.
type MetadataConfig struct {
	Enabled   bool   `yaml:"enabled"`
	UserAgent string `yaml:"user_agent"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Save writes the config to a YAML file.
func Save(path string, cfg Config) error {
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// Default returns a starter configuration.
func Default() Config {
	return Config{
		MusicPath: `D:\Music`,
		DataPath:  `data`,
		Host:      "0.0.0.0",
		Port:      8080,
		Token:     "change-me-to-a-secure-token",
		Metadata: MetadataConfig{
			Enabled:   true,
			UserAgent: "GoMusic/1.0 (https://github.com/temic/go-music)",
		},
	}
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *Config) applyDefaults() {
	if c.DataPath == "" {
		c.DataPath = "data"
	}
	if c.Metadata.UserAgent == "" {
		c.Metadata.UserAgent = "GoMusic/1.0 (https://github.com/temic/go-music)"
	}
}

// Normalize returns cfg with default values applied.
func Normalize(cfg Config) Config {
	cfg.applyDefaults()
	return cfg
}

// ResolveDataPath returns an absolute path for the metadata storage directory.
func ResolveDataPath(dataPath string) string {
	dataPath = strings.TrimSpace(dataPath)
	if dataPath == "" {
		dataPath = "data"
	}
	if filepath.IsAbs(dataPath) {
		return filepath.Clean(dataPath)
	}

	if exe, err := os.Executable(); err == nil {
		return filepath.Clean(filepath.Join(filepath.Dir(exe), dataPath))
	}

	return filepath.Clean(filepath.Join(".", dataPath))
}

func (c Config) validate() error {
	if c.MusicPath == "" {
		return fmt.Errorf("music_path is required")
	}
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if c.Token == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}
