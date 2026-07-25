package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"autodiscoverly/internal/mailconfig"
)

// v2Response mirrors Microsoft's real Autodiscover v2 shape: a pointer to a
// single service endpoint URL for a requested "next-level protocol" (e.g.
// ActiveSync, EWS, REST), not a bag of IMAP/SMTP server settings -- v2 has
// no IMAP/SMTP protocol value at all (confirmed against Microsoft Learn/blog
// documentation of the real request/response shape; see AI_USAGE.md). The
// one case worth answering here is AutodiscoverV1, which points the client
// back at our own verified POX endpoint.
type v2Response struct {
	Protocol string `json:"Protocol"`
	Url      string `json:"Url"`
}

type v2ErrorResponse struct {
	ErrorCode string `json:"ErrorCode"`
	Message   string `json:"Message"`
}

// NewAutodiscoverV2Handler serves GET /autodiscover/autodiscover.json.
func NewAutodiscoverV2Handler(resolver *mailconfig.Resolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("Email")
		protocol := r.URL.Query().Get("Protocol")

		domain := domainFromEmail(email)
		if domain == "" {
			writeJSON(w, http.StatusBadRequest, v2ErrorResponse{
				ErrorCode: "InvalidRequest",
				Message:   "Email query parameter must be a valid email address",
			})
			return
		}

		if _, ok := resolver.Lookup(domain); !ok {
			writeJSON(w, http.StatusNotFound, v2ErrorResponse{
				ErrorCode: "InvalidUser",
				Message:   "Requested account could not be found",
			})
			return
		}

		if !strings.EqualFold(protocol, "AutodiscoverV1") {
			writeJSON(w, http.StatusNotFound, v2ErrorResponse{
				ErrorCode: "ProtocolNotSupported",
				Message:   "This server only resolves the AutodiscoverV1 (POX) endpoint via v2; it has no Exchange-native services to point to",
			})
			return
		}

		// r.Host reflects what the client connected to (a reverse proxy
		// preserves the original Host header when forwarding); Autodiscover
		// always runs over HTTPS from the client's perspective regardless
		// of whether TLS terminates here or upstream.
		writeJSON(w, http.StatusOK, v2Response{
			Protocol: "AutodiscoverV1",
			Url:      "https://" + r.Host + "/autodiscover/autodiscover.xml",
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
