// Package handlers implements the HTTP endpoints for Outlook Autodiscover
// (POX, SOAP, v2 JSON), Mozilla Autoconfig, and the health check.
package handlers

import (
	"bytes"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"strings"

	"autodiscoverly/internal/mailconfig"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB is generous for these small XML/SOAP requests.

const (
	poxResponseSchemaNS = "http://schemas.microsoft.com/exchange/autodiscover/responseschema/2006"
	poxOutlookSchemaNS  = "http://schemas.microsoft.com/exchange/autodiscover/outlook/responseschema/2006a"
)

type poxRequest struct {
	XMLName xml.Name `xml:"Autodiscover"`
	Request struct {
		EMailAddress             string `xml:"EMailAddress"`
		AcceptableResponseSchema string `xml:"AcceptableResponseSchema"`
	} `xml:"Request"`
}

type poxProtocol struct {
	Type           string `xml:"Type"`
	Server         string `xml:"Server"`
	Port           int    `xml:"Port"`
	DomainRequired string `xml:"DomainRequired"`
	LoginName      string `xml:"LoginName,omitempty"`
	SPA            string `xml:"SPA"`
	SSL            string `xml:"SSL"`
	AuthRequired   string `xml:"AuthRequired"`
}

type poxAccount struct {
	AccountType string        `xml:"AccountType"`
	Action      string        `xml:"Action"`
	Protocol    []poxProtocol `xml:"Protocol"`
}

type poxUser struct {
	DisplayName string `xml:"DisplayName,omitempty"`
}

type poxError struct {
	ErrorCode int    `xml:"ErrorCode"`
	Message   string `xml:"Message"`
}

type poxResponseBody struct {
	XMLName xml.Name    `xml:"Response"`
	Xmlns   string      `xml:"xmlns,attr"`
	User    *poxUser    `xml:"User,omitempty"`
	Account *poxAccount `xml:"Account,omitempty"`
	Error   *poxError   `xml:"Error,omitempty"`
}

type poxEnvelope struct {
	XMLName  xml.Name `xml:"Autodiscover"`
	Xmlns    string   `xml:"xmlns,attr"`
	Response poxResponseBody
}

// NewAutodiscoverXMLHandler serves POST /autodiscover/autodiscover.xml. Both
// Outlook's legacy POX protocol and its SOAP GetUserSettings variant are
// posted to this same URL, so the handler sniffs the body's root XML element
// to decide which one it's looking at.
func NewAutodiscoverXMLHandler(resolver *mailconfig.Resolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes))
		if err != nil {
			http.Error(w, "unable to read request body", http.StatusBadRequest)
			return
		}

		if rootElementLocalName(body) == "Envelope" {
			handleSOAP(w, body, resolver)
			return
		}
		handlePOX(w, body, resolver)
	}
}

func handlePOX(w http.ResponseWriter, body []byte, resolver *mailconfig.Resolver) {
	var req poxRequest
	if err := xml.Unmarshal(body, &req); err != nil || req.Request.EMailAddress == "" {
		writePOXError(w, 600, "Invalid request")
		return
	}

	email := req.Request.EMailAddress
	domain := domainFromEmail(email)
	if domain == "" {
		writePOXError(w, 600, "Invalid request")
		return
	}

	if strings.Contains(req.Request.AcceptableResponseSchema, "mobilesync") {
		// We don't speak Exchange ActiveSync; tell the client so it falls
		// back to plain IMAP setup instead of hanging on a sync attempt.
		writePOXError(w, 600, "ActiveSync is not supported by this server; use IMAP")
		return
	}

	resolved, ok := resolver.Lookup(domain)
	if !ok {
		writePOXError(w, 500, "Requested account could not be found")
		return
	}

	account := poxAccount{
		AccountType: "email",
		Action:      "settings",
		Protocol: []poxProtocol{
			poxProtocolFor("IMAP", resolved.IMAP, email),
			poxProtocolFor("SMTP", resolved.SMTP, email),
		},
	}

	envelope := poxEnvelope{
		Xmlns: poxResponseSchemaNS,
		Response: poxResponseBody{
			Xmlns:   poxOutlookSchemaNS,
			User:    &poxUser{DisplayName: resolved.DisplayName},
			Account: &account,
		},
	}

	writeXML(w, http.StatusOK, envelope)
}

func poxProtocolFor(protocolType string, server mailconfig.ResolvedServer, email string) poxProtocol {
	ssl := "off"
	if server.SSLEnabled() {
		ssl = "on"
	}
	return poxProtocol{
		Type:           protocolType,
		Server:         server.Hostname,
		Port:           server.Port,
		DomainRequired: "off",
		LoginName:      server.Username(email),
		SPA:            "off",
		SSL:            ssl,
		AuthRequired:   "on",
	}
}

func writePOXError(w http.ResponseWriter, code int, message string) {
	envelope := poxEnvelope{
		Xmlns: poxResponseSchemaNS,
		Response: poxResponseBody{
			Xmlns: poxOutlookSchemaNS,
			Error: &poxError{ErrorCode: code, Message: message},
		},
	}
	// Autodiscover reports logical errors as HTTP 200 with an <Error>
	// element in the body; Outlook does not treat a non-200 status as a
	// recoverable Autodiscover response.
	writeXML(w, http.StatusOK, envelope)
}

func writeXML(w http.ResponseWriter, status int, v any) {
	out, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("marshaling XML response: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(xml.Header))
	w.Write(out)
}

func domainFromEmail(email string) string {
	at := strings.LastIndexByte(email, '@')
	if at < 0 || at == len(email)-1 {
		return ""
	}
	return strings.ToLower(email[at+1:])
}

func rootElementLocalName(body []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(body))
	for {
		tok, err := dec.Token()
		if err != nil {
			return ""
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local
		}
	}
}
