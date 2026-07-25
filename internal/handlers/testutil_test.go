package handlers

import (
	"autodiscoverly/internal/config"
	"autodiscoverly/internal/mailconfig"
)

func newTestResolver() *mailconfig.Resolver {
	cfg := &config.Config{
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
			"example.com": {DisplayName: "Example Mail"},
			"noname.example": {},
			"starttls.example": {
				DisplayName: "STARTTLS Example",
				IMAP: &config.ServerSettings{
					Port:       143,
					Encryption: config.EncryptionSTARTTLS,
				},
				SMTP: &config.ServerSettings{
					Port:       587,
					Encryption: config.EncryptionSTARTTLS,
				},
			},
		},
	}
	return mailconfig.New(cfg)
}
