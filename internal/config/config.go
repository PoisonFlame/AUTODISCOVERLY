// Package config loads and validates the YAML configuration that maps mail
// domains to their IMAP/SMTP server settings.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Encryption is the transport security used to connect to a mail server.
type Encryption string

const (
	EncryptionSSL      Encryption = "SSL"
	EncryptionSTARTTLS Encryption = "STARTTLS"
	EncryptionNone     Encryption = "none"
)

// UsernameFormat controls what login name is advertised to mail clients.
type UsernameFormat string

const (
	UsernameFormatEmail     UsernameFormat = "email"
	UsernameFormatLocalPart UsernameFormat = "local-part"
)

// ServerSettings describes how to reach a single IMAP or SMTP server.
// Any zero-value field is inherited from Defaults when merged for a domain.
type ServerSettings struct {
	Hostname       string         `yaml:"hostname"`
	Port           int            `yaml:"port"`
	Encryption     Encryption     `yaml:"encryption"`
	UsernameFormat UsernameFormat `yaml:"username_format"`
}

// Defaults holds the IMAP/SMTP settings shared by domains that don't
// override them.
type Defaults struct {
	IMAP ServerSettings `yaml:"imap"`
	SMTP ServerSettings `yaml:"smtp"`
}

// DomainOverride holds the per-domain settings. Nil IMAP/SMTP means "use
// Defaults entirely"; a non-nil ServerSettings is merged field-by-field over
// Defaults.
type DomainOverride struct {
	DisplayName string          `yaml:"display_name"`
	IMAP        *ServerSettings `yaml:"imap"`
	SMTP        *ServerSettings `yaml:"smtp"`
}

// Server holds process-level HTTP server settings.
type Server struct {
	Listen string `yaml:"listen"`
}

// Config is the top-level structure of config.yaml.
type Config struct {
	Server   Server                    `yaml:"server"`
	Defaults Defaults                  `yaml:"defaults"`
	Domains  map[string]DomainOverride `yaml:"domains"`
}

// Load reads and parses the YAML config file at path, applying defaults and
// running basic structural validation. It does not resolve/merge per-domain
// settings; see the mailconfig package for that.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8080"
	}

	if len(cfg.Domains) == 0 {
		return nil, fmt.Errorf("config file %q defines no domains", path)
	}

	return &cfg, nil
}
