package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"autodiscoverly/internal/config"
	"autodiscoverly/internal/mailconfig"
)

func testResolver() *mailconfig.Resolver {
	cfg := &config.Config{
		Server: config.Server{Listen: ":8080"},
		Defaults: config.Defaults{
			IMAP: config.ServerSettings{Hostname: "mail.mxrouting.net", Port: 993, Encryption: config.EncryptionSSL, UsernameFormat: config.UsernameFormatEmail},
			SMTP: config.ServerSettings{Hostname: "mail.mxrouting.net", Port: 465, Encryption: config.EncryptionSSL, UsernameFormat: config.UsernameFormatEmail},
		},
		Domains: map[string]config.DomainOverride{
			"example.com": {DisplayName: "Example Mail"},
		},
	}
	return mailconfig.New(cfg)
}

// TestRoutes is an end-to-end smoke test wiring the full router and hitting
// each endpoint the way a real mail client would.
func TestRoutes(t *testing.T) {
	handler := New(testResolver())
	ts := httptest.NewServer(handler)
	defer ts.Close()

	t.Run("health", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("pox autodiscover", func(t *testing.T) {
		body := `<Autodiscover xmlns="http://schemas.microsoft.com/exchange/autodiscover/outlook/requestschema/2006"><Request><EMailAddress>user@example.com</EMailAddress><AcceptableResponseSchema>http://schemas.microsoft.com/exchange/autodiscover/outlook/responseschema/2006a</AcceptableResponseSchema></Request></Autodiscover>`
		resp, err := http.Post(ts.URL+"/autodiscover/autodiscover.xml", "text/xml", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("autodiscover v2 json", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/autodiscover/autodiscover.json?Email=user@example.com&Protocol=IMAP")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("mozilla autoconfig", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/mail/config-v1.1.xml?emailaddress=user@example.com")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("mozilla autoconfig well-known", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/.well-known/autoconfig/mail/config-v1.1.xml", nil)
		req.Host = "example.com"
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
	})
}
