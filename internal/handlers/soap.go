package handlers

import (
	"bytes"
	"encoding/xml"
	"net/http"

	"autodiscoverly/internal/mailconfig"
)

// SOAP is a secondary path: modern Outlook (including Outlook Mobile)
// overwhelmingly uses POX to configure plain IMAP/SMTP accounts, and only a
// minority of older clients try SOAP GetUserSettings first. We respond on
// the same URL so those clients still get usable settings instead of a hard
// failure. The wire format here (namespace, UserSetting xsi:type
// discriminator, ProtocolConnection element names) is verified against
// Microsoft's published GetUserSettings SOAP reference and the [MS-OXWSADISC]
// full XML schema, not guessed -- see AI_USAGE.md for sources.

const (
	soapEnvelopeNS     = "http://schemas.xmlsoap.org/soap/envelope/"
	soapAutodiscoverNS = "http://schemas.microsoft.com/exchange/2010/Autodiscover"
	xsiNS              = "http://www.w3.org/2001/XMLSchema-instance"
)

type protocolConnection struct {
	Hostname         string `xml:"Hostname"`
	Port             int    `xml:"Port"`
	EncryptionMethod string `xml:"EncryptionMethod"`
}

// soapUserSetting only models the one xsi:type this server ever emits
// (ProtocolConnectionCollectionSetting, used for *Imap4Connections and
// *SmtpConnections) -- the base schema also defines StringSetting and other
// variants for settings we don't implement (mailbox server DN, EWS URLs,
// etc.), which don't apply to a plain IMAP/SMTP provider.
type soapUserSetting struct {
	XSIType             string               `xml:"i:type,attr"`
	Name                string               `xml:"Name"`
	ProtocolConnections []protocolConnection `xml:"ProtocolConnections>ProtocolConnection"`
}

type soapUserResponse struct {
	ErrorCode    string            `xml:"ErrorCode"`
	ErrorMessage string            `xml:"ErrorMessage"`
	UserSettings []soapUserSetting `xml:"UserSettings>UserSetting"`
}

type soapResponse struct {
	XMLName       xml.Name           `xml:"Response"`
	Xmlns         string             `xml:"xmlns,attr"`
	XmlnsI        string             `xml:"xmlns:i,attr"`
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
		protocolConnectionSetting("InternalImap4Connections", resolved.IMAP),
		protocolConnectionSetting("ExternalImap4Connections", resolved.IMAP),
		protocolConnectionSetting("InternalSmtpConnections", resolved.SMTP),
		protocolConnectionSetting("ExternalSmtpConnections", resolved.SMTP),
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

func protocolConnectionSetting(name string, server mailconfig.ResolvedServer) soapUserSetting {
	return soapUserSetting{
		XSIType: "ProtocolConnectionCollectionSetting",
		Name:    name,
		ProtocolConnections: []protocolConnection{{
			Hostname:         server.Hostname,
			Port:             server.Port,
			EncryptionMethod: encryptionMethodString(server),
		}},
	}
}

// EncryptionMethod is documented as free-form text (no fixed enum), so
// these values follow the same SSL/STARTTLS/none vocabulary the rest of the
// config uses rather than a Microsoft-specified list.
func encryptionMethodString(server mailconfig.ResolvedServer) string {
	switch {
	case server.SSLEnabled():
		return "SSL"
	case server.Encryption == "STARTTLS":
		return "TLS"
	default:
		return "None"
	}
}

func writeSOAPResponse(w http.ResponseWriter, resp soapResponse) {
	resp.XmlnsI = xsiNS
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
