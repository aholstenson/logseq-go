package utils

import "strings"

// dateFormatToken maps a token of the date format Logseq uses in its config to
// the equivalent in a Go layout.
type dateFormatToken struct {
	// token is what the format uses.
	token string
	// layout is what Go uses for the same thing.
	layout string
}

// dateFormatTokens are the tokens that can appear in a date format, in the
// order they are matched. Tokens sharing a prefix are ordered longest first so
// the longest one wins, keeping `EEEE` from being read as `EEE` and a leftover
// `E`.
var dateFormatTokens = []dateFormatToken{
	{"yyyy", "2006"},
	{"yy", "06"},
	{"MMMM", "January"},
	{"MMM", "Jan"},
	{"MM", "01"},
	{"M", "1"},
	// `do` is an ordinal day, which Go does not have, so it uses the plain day.
	{"do", "2"},
	{"dd", "02"},
	{"d", "2"},
	{"EEEE", "Monday"},
	{"EEE", "Mon"},
	{"HH", "15"},
	{"H", "15"},
	{"hh", "03"},
	{"h", "3"},
	{"mm", "04"},
	{"m", "4"},
	{"ss", "05"},
	{"s", "5"},
}

// ConvertDateFormat converts a date format as used by Logseq into a layout that
// can be used with Go time formatting and parsing. Anything that is not a token
// is kept as it is.
func ConvertDateFormat(dateFormat string) string {
	var goDateFormat strings.Builder

	for i := 0; i < len(dateFormat); {
		// Tokens are replaced where they are found instead of one token at a
		// time over the whole format, so a replacement can not be picked up as
		// a token of its own.
		matched := false
		for _, t := range dateFormatTokens {
			if strings.HasPrefix(dateFormat[i:], t.token) {
				goDateFormat.WriteString(t.layout)
				i += len(t.token)
				matched = true
				break
			}
		}

		if !matched {
			goDateFormat.WriteByte(dateFormat[i])
			i++
		}
	}

	return goDateFormat.String()
}
