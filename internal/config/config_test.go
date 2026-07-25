package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}

func TestLoad_Valid(t *testing.T) {
	path := writeTempConfig(t, `
server:
  listen: ":9090"
defaults:
  imap:
    hostname: mail.mxrouting.net
    port: 993
    encryption: SSL
    username_format: email
  smtp:
    hostname: mail.mxrouting.net
    port: 465
    encryption: SSL
    username_format: email
domains:
  example.com:
    display_name: "Example Mail"
  another.org:
    display_name: "Another Org"
    imap:
      port: 143
      encryption: STARTTLS
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Server.Listen != ":9090" {
		t.Errorf("Server.Listen = %q, want %q", cfg.Server.Listen, ":9090")
	}
	if len(cfg.Domains) != 2 {
		t.Fatalf("len(Domains) = %d, want 2", len(cfg.Domains))
	}
	if cfg.Defaults.IMAP.Hostname != "mail.mxrouting.net" {
		t.Errorf("Defaults.IMAP.Hostname = %q, want mail.mxrouting.net", cfg.Defaults.IMAP.Hostname)
	}

	another := cfg.Domains["another.org"]
	if another.IMAP == nil || another.IMAP.Port != 143 {
		t.Errorf("another.org IMAP override not parsed correctly: %+v", another.IMAP)
	}
}

func TestLoad_DefaultListenAddr(t *testing.T) {
	path := writeTempConfig(t, `
domains:
  example.com: {}
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Server.Listen != ":8080" {
		t.Errorf("Server.Listen = %q, want default :8080", cfg.Server.Listen)
	}
}

func TestLoad_NoDomains(t *testing.T) {
	path := writeTempConfig(t, `
server:
  listen: ":8080"
domains: {}
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() with no domains should return an error")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("Load() with a missing file should return an error")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTempConfig(t, "not: valid: yaml: [")

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() with invalid YAML should return an error")
	}
}
