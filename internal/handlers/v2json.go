package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"autodiscoverly/internal/mailconfig"
)

// v2Response mirrors the shape returned for Protocol=IMAP/POP3/SMTP queries
// against Autodiscover v2. Note: Microsoft's v2 JSON endpoint is primarily
// documented for locating Exchange-native service endpoints (EWS/ActiveSync/
// REST), not plain IMAP server settings, so this response shape is a
// best-effort implementation modeled on the same fields POX exposes. Verify
// against a real Outlook Mobile client before depending on it exclusively —
// POX (handled by NewAutodiscoverXMLHandler) is the well-established
// fallback either way.
type v2Response struct {
	Protocol string `json:"Protocol"`
	Server   string `json:"Server,omitempty"`
	Port     int    `json:"Port,omitempty"`
	SSL      bool   `json:"SSL"`
	Username string `json:"Username,omitempty"`
}

type v2ErrorResponse struct {
	ErrorCode string `json:"ErrorCode"`
	Message   string `json:"Message"`
}

var v2SupportedProtocols = map[string]func(mailconfig.ResolvedDomain) mailconfig.ResolvedServer{
	"IMAP": func(d mailconfig.ResolvedDomain) mailconfig.ResolvedServer { return d.IMAP },
	"SMTP": func(d mailconfig.ResolvedDomain) mailconfig.ResolvedServer { return d.SMTP },
}

// NewAutodiscoverV2Handler serves GET /autodiscover/autodiscover.json.
func NewAutodiscoverV2Handler(resolver *mailconfig.Resolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("Email")
		protocol := strings.ToUpper(r.URL.Query().Get("Protocol"))
		if protocol == "" {
			protocol = "IMAP"
		}

		domain := domainFromEmail(email)
		if domain == "" {
			writeJSON(w, http.StatusBadRequest, v2ErrorResponse{
				ErrorCode: "InvalidRequest",
				Message:   "Email query parameter must be a valid email address",
			})
			return
		}

		resolved, ok := resolver.Lookup(domain)
		if !ok {
			writeJSON(w, http.StatusNotFound, v2ErrorResponse{
				ErrorCode: "InvalidUser",
				Message:   "Requested account could not be found",
			})
			return
		}

		serverFor, ok := v2SupportedProtocols[protocol]
		if !ok {
			writeJSON(w, http.StatusNotFound, v2ErrorResponse{
				ErrorCode: "ProtocolNotSupported",
				Message:   "This server only supports IMAP and SMTP autodiscovery",
			})
			return
		}
		server := serverFor(resolved)

		writeJSON(w, http.StatusOK, v2Response{
			Protocol: protocol,
			Server:   server.Hostname,
			Port:     server.Port,
			SSL:      server.SSLEnabled(),
			Username: server.Username(email),
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
