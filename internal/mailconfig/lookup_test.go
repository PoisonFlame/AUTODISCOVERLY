package mailconfig

import (
	"testing"

	"autodiscoverly/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		Server: config.Server{Listen: ":8080"},
		Defaults: config.Defaults{
			IMAP: config.ServerSettings{
				Hostname:       "mail.mxrouting.net",
				Port:           993,
				Encryption:     config.EncryptionSSL,
				UsernameFormat: config.UsernameFormatEmail,
			},
			SMTP: config.ServerSettings{
				Hostname:       "mail.mxrouting.net",
				Port:           465,
				Encryption:     config.EncryptionSSL,
				UsernameFormat: config.UsernameFormatEmail,
			},
		},
		Domains: map[string]config.DomainOverride{
			"Example.com": {DisplayName: "Example Mail"},
			"another.org": {
				DisplayName: "Another Org",
				IMAP: &config.ServerSettings{
					Port:       143,
					Encryption: config.EncryptionSTARTTLS,
				},
			},
		},
	}
}

func TestLookup_UsesDefaults(t *testing.T) {
	r := New(testConfig())

	resolved, ok := r.Lookup("example.com")
	if !ok {
		t.Fatal("Lookup(example.com) should be found")
	}
	if resolved.IMAP.Hostname != "mail.mxrouting.net" || resolved.IMAP.Port != 993 {
		t.Errorf("unexpected IMAP settings: %+v", resolved.IMAP)
	}
	if resolved.SMTP.Hostname != "mail.mxrouting.net" || resolved.SMTP.Port != 465 {
		t.Errorf("unexpected SMTP settings: %+v", resolved.SMTP)
	}
}

func TestLookup_CaseInsensitive(t *testing.T) {
	r := New(testConfig())

	if _, ok := r.Lookup("EXAMPLE.COM"); !ok {
		t.Fatal("Lookup should be case-insensitive")
	}
}

func TestLookup_PerFieldOverride(t *testing.T) {
	r := New(testConfig())

	resolved, ok := r.Lookup("another.org")
	if !ok {
		t.Fatal("Lookup(another.org) should be found")
	}
	// Hostname inherited from defaults, port/encryption overridden.
	if resolved.IMAP.Hostname != "mail.mxrouting.net" {
		t.Errorf("IMAP.Hostname = %q, want inherited default", resolved.IMAP.Hostname)
	}
	if resolved.IMAP.Port != 143 {
		t.Errorf("IMAP.Port = %d, want 143", resolved.IMAP.Port)
	}
	if resolved.IMAP.Encryption != config.EncryptionSTARTTLS {
		t.Errorf("IMAP.Encryption = %q, want STARTTLS", resolved.IMAP.Encryption)
	}
	// SMTP untouched by override, still defaults.
	if resolved.SMTP.Port != 465 {
		t.Errorf("SMTP.Port = %d, want default 465", resolved.SMTP.Port)
	}
}

func TestLookup_UnknownDomain(t *testing.T) {
	r := New(testConfig())

	if _, ok := r.Lookup("nope.example"); ok {
		t.Fatal("Lookup(nope.example) should not be found")
	}
}

func TestUsername(t *testing.T) {
	emailFmt := ResolvedServer{UsernameFormat: config.UsernameFormatEmail}
	localFmt := ResolvedServer{UsernameFormat: config.UsernameFormatLocalPart}

	if got := emailFmt.Username("user@example.com"); got != "user@example.com" {
		t.Errorf("email format Username() = %q, want full address", got)
	}
	if got := localFmt.Username("user@example.com"); got != "user" {
		t.Errorf("local-part format Username() = %q, want local part", got)
	}
}

func TestValidateAll_CatchesMissingHostname(t *testing.T) {
	cfg := testConfig()
	cfg.Defaults.IMAP.Hostname = ""
	r := New(cfg)

	if err := r.ValidateAll(); err == nil {
		t.Fatal("ValidateAll() should error when a resolved domain has no IMAP hostname")
	}
}

func TestValidateAll_OK(t *testing.T) {
	r := New(testConfig())
	if err := r.ValidateAll(); err != nil {
		t.Fatalf("ValidateAll() returned unexpected error: %v", err)
	}
}
