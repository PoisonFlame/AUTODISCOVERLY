// Package mailconfig resolves a mail domain to its effective IMAP/SMTP
// settings, merging per-domain overrides on top of the configured defaults.
package mailconfig

import (
	"fmt"
	"strings"

	"autodiscoverly/internal/config"
)

// ResolvedServer is the effective (post-merge, post-default) settings for
// reaching an IMAP or SMTP server.
type ResolvedServer struct {
	Hostname       string
	Port           int
	Encryption     config.Encryption
	UsernameFormat config.UsernameFormat
}

// Username returns the login name to advertise to a mail client for the
// given email address, based on this server's UsernameFormat.
func (s ResolvedServer) Username(email string) string {
	if s.UsernameFormat == config.UsernameFormatLocalPart {
		if at := strings.IndexByte(email, '@'); at >= 0 {
			return email[:at]
		}
	}
	return email
}

// SSLEnabled reports whether the connection uses implicit TLS (as opposed to
// STARTTLS or no encryption).
func (s ResolvedServer) SSLEnabled() bool {
	return s.Encryption == config.EncryptionSSL
}

// ResolvedDomain is the effective settings for a single mail domain.
type ResolvedDomain struct {
	Domain      string
	DisplayName string
	IMAP        ResolvedServer
	SMTP        ResolvedServer
}

// Resolver looks up the effective mail settings for a domain.
type Resolver struct {
	defaults config.Defaults
	domains  map[string]config.DomainOverride
}

// New builds a Resolver from a loaded Config. Domain keys are matched
// case-insensitively.
func New(cfg *config.Config) *Resolver {
	domains := make(map[string]config.DomainOverride, len(cfg.Domains))
	for domain, override := range cfg.Domains {
		domains[strings.ToLower(domain)] = override
	}
	return &Resolver{defaults: cfg.Defaults, domains: domains}
}

// Lookup returns the effective settings for domain, or ok=false if the
// domain isn't configured.
func (r *Resolver) Lookup(domain string) (ResolvedDomain, bool) {
	override, ok := r.domains[strings.ToLower(domain)]
	if !ok {
		return ResolvedDomain{}, false
	}

	return ResolvedDomain{
		Domain:      strings.ToLower(domain),
		DisplayName: override.DisplayName,
		IMAP:        mergeServer(r.defaults.IMAP, override.IMAP),
		SMTP:        mergeServer(r.defaults.SMTP, override.SMTP),
	}, true
}

// ValidateAll resolves every configured domain and returns an error
// describing the first one with incomplete effective settings (missing
// hostname/port or an unrecognized encryption/username format). Intended to
// be called once at startup so misconfiguration fails fast.
func (r *Resolver) ValidateAll() error {
	for domain := range r.domains {
		resolved, _ := r.Lookup(domain)
		if err := validateServer("imap", resolved.IMAP); err != nil {
			return fmt.Errorf("domain %q: %w", domain, err)
		}
		if err := validateServer("smtp", resolved.SMTP); err != nil {
			return fmt.Errorf("domain %q: %w", domain, err)
		}
	}
	return nil
}

func validateServer(protocol string, s ResolvedServer) error {
	if s.Hostname == "" {
		return fmt.Errorf("%s hostname is not set (no default and no override)", protocol)
	}
	if s.Port <= 0 {
		return fmt.Errorf("%s port is not set (no default and no override)", protocol)
	}
	switch s.Encryption {
	case config.EncryptionSSL, config.EncryptionSTARTTLS, config.EncryptionNone:
	default:
		return fmt.Errorf("%s encryption %q is not one of SSL, STARTTLS, none", protocol, s.Encryption)
	}
	switch s.UsernameFormat {
	case config.UsernameFormatEmail, config.UsernameFormatLocalPart:
	default:
		return fmt.Errorf("%s username_format %q is not one of email, local-part", protocol, s.UsernameFormat)
	}
	return nil
}

func mergeServer(def config.ServerSettings, override *config.ServerSettings) ResolvedServer {
	result := ResolvedServer{
		Hostname:       def.Hostname,
		Port:           def.Port,
		Encryption:     def.Encryption,
		UsernameFormat: def.UsernameFormat,
	}
	if override != nil {
		if override.Hostname != "" {
			result.Hostname = override.Hostname
		}
		if override.Port != 0 {
			result.Port = override.Port
		}
		if override.Encryption != "" {
			result.Encryption = override.Encryption
		}
		if override.UsernameFormat != "" {
			result.UsernameFormat = override.UsernameFormat
		}
	}
	if result.Encryption == "" {
		result.Encryption = config.EncryptionSSL
	}
	if result.UsernameFormat == "" {
		result.UsernameFormat = config.UsernameFormatEmail
	}
	return result
}
