package handlers

import (
	"encoding/xml"
	"net/http"
	"strings"

	"autodiscoverly/internal/mailconfig"
)

type autoconfigServer struct {
	Type           string `xml:"type,attr"`
	Hostname       string `xml:"hostname"`
	Port           int    `xml:"port"`
	SocketType     string `xml:"socketType"`
	Username       string `xml:"username"`
	Authentication string `xml:"authentication"`
}

type autoconfigProvider struct {
	ID             string           `xml:"id,attr"`
	Domain         string           `xml:"domain"`
	DisplayName    string           `xml:"displayName,omitempty"`
	IncomingServer autoconfigServer `xml:"incomingServer"`
	OutgoingServer autoconfigServer `xml:"outgoingServer"`
}

type autoconfigClientConfig struct {
	XMLName  xml.Name           `xml:"clientConfig"`
	Version  string             `xml:"version,attr"`
	Provider autoconfigProvider `xml:"emailProvider"`
}

// NewAutoconfigHandler serves the Mozilla Autoconfig format used by
// Thunderbird and other ISPDB-aware clients, both at
// GET /mail/config-v1.1.xml (on autoconfig.<domain>) and at
// GET /.well-known/autoconfig/mail/config-v1.1.xml (on <domain> itself).
func NewAutoconfigHandler(resolver *mailconfig.Resolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := domainFromAutoconfigRequest(r)
		if domain == "" {
			http.Error(w, "unable to determine domain from request", http.StatusBadRequest)
			return
		}

		resolved, ok := resolver.Lookup(domain)
		if !ok {
			http.NotFound(w, r)
			return
		}

		email := r.URL.Query().Get("emailaddress")
		displayName := resolved.DisplayName
		if email == "" {
			email = "%EMAILADDRESS%"
		} else if displayName == "" {
			// Only derive from the local part when we have a real address;
			// deriving from the %EMAILADDRESS% placeholder would be garbage.
			displayName = deriveDisplayName(email)
		}

		cfg := autoconfigClientConfig{
			Version: "1.1",
			Provider: autoconfigProvider{
				ID:          resolved.Domain,
				Domain:      resolved.Domain,
				DisplayName: displayName,
				IncomingServer: autoconfigServer{
					Type:           "imap",
					Hostname:       resolved.IMAP.Hostname,
					Port:           resolved.IMAP.Port,
					SocketType:     socketType(resolved.IMAP),
					Username:       autoconfigUsername(resolved.IMAP, email),
					Authentication: "password-cleartext",
				},
				OutgoingServer: autoconfigServer{
					Type:           "smtp",
					Hostname:       resolved.SMTP.Hostname,
					Port:           resolved.SMTP.Port,
					SocketType:     socketType(resolved.SMTP),
					Username:       autoconfigUsername(resolved.SMTP, email),
					Authentication: "password-cleartext",
				},
			},
		}

		writeXML(w, http.StatusOK, cfg)
	}
}

// autoconfigUsername mirrors ResolvedServer.Username but preserves the
// %EMAILADDRESS%/%EMAILLOCALPART% placeholders Thunderbird expects when the
// client didn't send a concrete address (e.g. .well-known probing).
func autoconfigUsername(server mailconfig.ResolvedServer, email string) string {
	if email == "%EMAILADDRESS%" {
		return "%EMAILADDRESS%"
	}
	return server.Username(email)
}

func socketType(server mailconfig.ResolvedServer) string {
	switch {
	case server.SSLEnabled():
		return "SSL"
	case server.Encryption == "STARTTLS":
		return "STARTTLS"
	default:
		return "plain"
	}
}

func domainFromAutoconfigRequest(r *http.Request) string {
	if email := r.URL.Query().Get("emailaddress"); email != "" {
		if domain := domainFromEmail(email); domain != "" {
			return domain
		}
	}

	host := r.Host
	if idx := strings.IndexByte(host, ':'); idx >= 0 {
		host = host[:idx]
	}
	host = strings.ToLower(host)
	return strings.TrimPrefix(host, "autoconfig.")
}
