package handlers

import (
	"strings"
	"unicode"

	"autodiscoverly/internal/mailconfig"
)

// displayNameFor returns the display name to advertise for email. A
// domain's configured DisplayName (if set) always wins, since that's an
// intentional branding choice; otherwise a per-mailbox name is derived from
// the local part of the address (e.g. "jane.doe@example.com" -> "Jane Doe"),
// since we have no server-side source of each user's real name.
func displayNameFor(resolved mailconfig.ResolvedDomain, email string) string {
	if resolved.DisplayName != "" {
		return resolved.DisplayName
	}
	return deriveDisplayName(email)
}

func deriveDisplayName(email string) string {
	local := email
	if at := strings.IndexByte(email, '@'); at >= 0 {
		local = email[:at]
	}

	words := strings.FieldsFunc(local, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == '+'
	})
	if len(words) == 0 {
		return local
	}

	for i, w := range words {
		words[i] = titleCaseWord(w)
	}
	return strings.Join(words, " ")
}

func titleCaseWord(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	r[0] = unicode.ToUpper(r[0])
	for i := 1; i < len(r); i++ {
		r[i] = unicode.ToLower(r[i])
	}
	return string(r)
}
