package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Listen != ":8787" {
		t.Errorf("listen = %q, want %q", cfg.Listen, ":8787")
	}
	if cfg.Providers.OpenAI.Upstream != "https://api.openai.com" {
		t.Errorf("openai upstream = %q", cfg.Providers.OpenAI.Upstream)
	}
	if !cfg.Providers.OpenAI.Enabled {
		t.Error("openai should be enabled by default")
	}
	if cfg.Providers.Anthropic.Upstream != "https://api.anthropic.com" {
		t.Errorf("anthropic upstream = %q", cfg.Providers.Anthropic.Upstream)
	}
	if !cfg.Providers.Anthropic.Enabled {
		t.Error("anthropic should be enabled by default")
	}
	if cfg.Storage.Driver != "sqlite" {
		t.Errorf("storage driver = %q", cfg.Storage.Driver)
	}
	if cfg.Storage.DSN != "data/agentledger.db" {
		t.Errorf("storage dsn = %q", cfg.Storage.DSN)
	}
	if cfg.Recording.BufferSize != 10000 {
		t.Errorf("buffer_size = %d", cfg.Recording.BufferSize)
	}
	if cfg.Recording.Workers != 4 {
		t.Errorf("workers = %d", cfg.Recording.Workers)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "test.yaml")
	content := []byte(`listen: ":9999"
providers:
  openai:
    upstream: "https://custom.openai.com"
    enabled: false
storage:
  dsn: "/tmp/test.db"
recording:
  buffer_size: 500
  workers: 2
`)
	if err := os.WriteFile(cfgFile, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Listen != ":9999" {
		t.Errorf("listen = %q, want %q", cfg.Listen, ":9999")
	}
	if cfg.Providers.OpenAI.Upstream != "https://custom.openai.com" {
		t.Errorf("openai upstream = %q", cfg.Providers.OpenAI.Upstream)
	}
	if cfg.Providers.OpenAI.Enabled {
		t.Error("openai should be disabled")
	}
	if cfg.Storage.DSN != "/tmp/test.db" {
		t.Errorf("dsn = %q", cfg.Storage.DSN)
	}
	if cfg.Recording.BufferSize != 500 {
		t.Errorf("buffer_size = %d", cfg.Recording.BufferSize)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("AGENTLEDGER_LISTEN", ":1234")
	t.Setenv("AGENTLEDGER_STORAGE_DSN", "/tmp/env.db")

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Listen != ":1234" {
		t.Errorf("listen = %q, want %q", cfg.Listen, ":1234")
	}
	if cfg.Storage.DSN != "/tmp/env.db" {
		t.Errorf("dsn = %q, want %q", cfg.Storage.DSN, "/tmp/env.db")
	}
}

func TestLoadMissingExplicitPath(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing explicit config path")
	}
}

func TestLoadAutoSearch(t *testing.T) {
	// Create a config in a temp dir and cd there
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "agentledger.yaml")
	if err := os.WriteFile(cfgFile, []byte(`listen: ":5555"`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Save and restore working dir
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":5555" {
		t.Errorf("listen = %q, want %q (auto-discovered)", cfg.Listen, ":5555")
	}
}
