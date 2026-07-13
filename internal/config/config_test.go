package config_test

import (
	"path/filepath"
	"testing"

	"github.com/temic/go-music/internal/config"
)

func TestNormalizeFillsDataPath(t *testing.T) {
	cfg := config.Normalize(config.Config{
		MusicPath: `D:\Music`,
		Host:      "0.0.0.0",
		Port:      8080,
		Token:     "secret",
	})

	if cfg.DataPath != "data" {
		t.Fatalf("data_path = %q, want data", cfg.DataPath)
	}
}

func TestResolveDataPathAbsolute(t *testing.T) {
	got := config.ResolveDataPath(`D:\Go_music\data`)
	if got != `D:\Go_music\data` {
		t.Fatalf("resolved = %q", got)
	}
}

func TestResolveDataPathRelativeUsesExecutableDir(t *testing.T) {
	got := config.ResolveDataPath("data")
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
	if filepath.Base(got) != "data" {
		t.Fatalf("base = %q, want data", filepath.Base(got))
	}
}
