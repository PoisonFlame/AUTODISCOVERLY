package handlers

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"strconv"

	"autodiscoverly/internal/mailconfig"
)

// SOAP support is best-effort: modern Outlook (including Outlook Mobile)
// overwhelmingly uses POX or the v2 JSON endpoint to configure plain
// IMAP/SMTP accounts, and only a minority of older clients try SOAP
// GetUserSettings first. We respond on the same URL so those clients still
// get usable settings instead of a hard failure.

const (
	soapEnvelopeNS     = "http://schemas.xmlsoap.org/soap/envelope/"
	soapAutodiscoverNS = "http://schemas.microsoft.com/exchange/2010/Autodiscover"
)

type soapUserSetting struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

type soapUserResponse struct {
	ErrorCode    string            `xml:"ErrorCode"`
	ErrorMessage string            `xml:"ErrorMessage"`
	UserSettings []soapUserSetting `xml:"UserSettings>UserSetting"`
}

type soapResponse struct {
	XMLName       xml.Name           `xml:"Response"`
	Xmlns         string             `xml:"xmlns,attr"`
	ErrorCode     string             `xml:"ErrorCode"`
	ErrorMessage  string             `xml:"ErrorMessage"`
	UserResponses []soapUserResponse `xml:"UserResponses>UserResponse"`
}

type soapResponseMessage struct {
	XMLName  xml.Name `xml:"GetUserSettingsResponseMessage"`
	Xmlns    string   `xml:"xmlns,attr"`
	Response soapResponse
}

type soapBody struct {
	XMLName  xml.Name `xml:"soap:Body"`
	Response soapResponseMessage
}

type soapEnvelope struct {
	XMLName xml.Name `xml:"soap:Envelope"`
	SoapNS  string   `xml:"xmlns:soap,attr"`
	Body    soapBody
}

func handleSOAP(w http.ResponseWriter, body []byte, resolver *mailconfig.Resolver) {
	email := extractSOAPMailbox(body)
	domain := domainFromEmail(email)

	if domain == "" {
		writeSOAPResponse(w, soapResponse{
			Xmlns:     soapAutodiscoverNS,
			ErrorCode: "InvalidRequest",
		})
		return
	}

	resolved, ok := resolver.Lookup(domain)
	if !ok {
		writeSOAPResponse(w, soapResponse{
			Xmlns:     soapAutodiscoverNS,
			ErrorCode: "NoError",
			UserResponses: []soapUserResponse{{
				ErrorCode: "InvalidUser",
			}},
		})
		return
	}

	settings := []soapUserSetting{
		{Name: "InternalImapServer", Value: resolved.IMAP.Hostname},
		{Name: "ExternalImapServer", Value: resolved.IMAP.Hostname},
		{Name: "InternalImapPort", Value: strconv.Itoa(resolved.IMAP.Port)},
		{Name: "ExternalImapPort", Value: strconv.Itoa(resolved.IMAP.Port)},
		{Name: "InternalImapSSL", Value: boolStr(resolved.IMAP.SSLEnabled())},
		{Name: "ExternalImapSSL", Value: boolStr(resolved.IMAP.SSLEnabled())},
		{Name: "InternalSmtpServer", Value: resolved.SMTP.Hostname},
		{Name: "ExternalSmtpServer", Value: resolved.SMTP.Hostname},
		{Name: "InternalSmtpPort", Value: strconv.Itoa(resolved.SMTP.Port)},
		{Name: "ExternalSmtpPort", Value: strconv.Itoa(resolved.SMTP.Port)},
		{Name: "InternalSmtpSSL", Value: boolStr(resolved.SMTP.SSLEnabled())},
		{Name: "ExternalSmtpSSL", Value: boolStr(resolved.SMTP.SSLEnabled())},
	}

	writeSOAPResponse(w, soapResponse{
		Xmlns:     soapAutodiscoverNS,
		ErrorCode: "NoError",
		UserResponses: []soapUserResponse{{
			ErrorCode:    "NoError",
			UserSettings: settings,
		}},
	})
}

func writeSOAPResponse(w http.ResponseWriter, resp soapResponse) {
	envelope := soapEnvelope{
		SoapNS: soapEnvelopeNS,
		Body: soapBody{
			Response: soapResponseMessage{
				Xmlns:    soapAutodiscoverNS,
				Response: resp,
			},
		},
	}
	writeXML(w, http.StatusOK, envelope)
}

// extractSOAPMailbox scans for a <Mailbox> or <EMailAddress> element
// anywhere in the body rather than binding to the full GetUserSettings
// request schema, since we only need the one field out of it.
func extractSOAPMailbox(body []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(body))
	for {
		tok, err := dec.Token()
		if err != nil {
			return ""
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local == "Mailbox" || se.Name.Local == "EMailAddress" {
			var value string
			if err := dec.DecodeElement(&value, &se); err == nil && value != "" {
				return value
			}
		}
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
