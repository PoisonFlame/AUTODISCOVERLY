package handlers

import (
	"testing"

	"autodiscoverly/internal/mailconfig"
)

func TestDeriveDisplayName(t *testing.T) {
	cases := map[string]string{
		"jane.doe@example.com": "Jane Doe",
		"j_smith@example.com":  "J Smith",
		"support@example.com":  "Support",
		"first-last@ex.com":    "First Last",
		"UPPER@example.com":    "Upper",
	}
	for email, want := range cases {
		if got := deriveDisplayName(email); got != want {
			t.Errorf("deriveDisplayName(%q) = %q, want %q", email, got, want)
		}
	}
}

func TestDisplayNameFor_ConfigOverrideWins(t *testing.T) {
	resolved := mailconfig.ResolvedDomain{DisplayName: "Example Mail"}
	if got := displayNameFor(resolved, "jane.doe@example.com"); got != "Example Mail" {
		t.Errorf("displayNameFor() = %q, want configured override to win", got)
	}
}

func TestDisplayNameFor_DerivesWhenUnset(t *testing.T) {
	resolved := mailconfig.ResolvedDomain{}
	if got := displayNameFor(resolved, "jane.doe@example.com"); got != "Jane Doe" {
		t.Errorf("displayNameFor() = %q, want derived Jane Doe", got)
	}
}
