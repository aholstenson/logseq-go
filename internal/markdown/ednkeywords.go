package markdown

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

// validEDNKeyword checks if a property name passes the check that Logseq
// does with its EDN reader: Logseq reads the text `{:name nil}` as EDN, and
// the name is valid if that gives a map without errors.
//
// The reader reads more than a keyword from such a text. A comma and a space
// separate the forms of the map, so the name `b,c,d` gives the map
// `{:b c, d nil}`, which is valid, and the name `b,c` gives `{:b c nil}`,
// which is not, as a map needs an even number of forms. The name is the
// lower case form of the name that Logseq checks.
//
// This follows the EDN reader of ClojureScript, tools.reader, for the
// characters that can be in a name. The characters that Logseq does not
// allow before it reads the name, such as quotes and parentheses, are not
// read here. See validPropertyName.
func validEDNKeyword(name string) bool {
	r := &ednReader{text: []rune(":" + name + " nil")}
	forms, err := r.readForms(ednEnd)
	if err != nil || len(forms)%2 != 0 {
		return false
	}

	keys := make(map[string]struct{}, len(forms)/2)
	for i := 0; i < len(forms); i += 2 {
		if _, duplicate := keys[forms[i]]; duplicate {
			return false
		}

		keys[forms[i]] = struct{}{}
	}

	return true
}

// ednEnd is what ednReader returns at the end of the text.
const ednEnd rune = -1

var errInvalidEDN = errors.New("invalid EDN")

// ednReader reads EDN forms from a text. A form is returned as a string that
// is the same for forms that are equal, which is enough to find duplicate
// keys in a map.
type ednReader struct {
	text []rune
	pos  int
}

func (r *ednReader) peek() rune {
	if r.pos >= len(r.text) {
		return ednEnd
	}

	return r.text[r.pos]
}

func (r *ednReader) next() rune {
	c := r.peek()
	if c != ednEnd {
		r.pos++
	}

	return c
}

// readForms reads the forms up to the given closing character, which is
// ednEnd for the forms up to the end of the text.
func (r *ednReader) readForms(closing rune) ([]string, error) {
	forms := make([]string, 0)
	for {
		for ednWhitespace(r.peek()) {
			r.next()
		}

		c := r.peek()
		if c == closing {
			r.next()
			return forms, nil
		}

		if c == ednEnd {
			// The text ends before the closing character
			return nil, errInvalidEDN
		}

		form, keep, err := r.readForm()
		if err != nil {
			return nil, err
		}

		if keep {
			forms = append(forms, form)
		}
	}
}

// readKeptForm reads the next form that is not discarded with `#_`.
func (r *ednReader) readKeptForm() (string, error) {
	for {
		for ednWhitespace(r.peek()) {
			r.next()
		}

		if r.peek() == ednEnd {
			return "", errInvalidEDN
		}

		form, keep, err := r.readForm()
		if err != nil || keep {
			return form, err
		}
	}
}

// readForm reads the form that starts at the next character. keep is false
// for a form that is discarded with `#_`.
func (r *ednReader) readForm() (form string, keep bool, err error) {
	c := r.next()
	switch {
	case c == ':':
		form, err = r.readKeyword()
	case c == '[':
		var forms []string
		forms, err = r.readForms(']')
		form = "[" + strings.Join(forms, " ") + "]"
	case c == '\\':
		form, err = r.readCharacter()
	case c == '#':
		return r.readDispatch()
	case strings.ContainsRune(`;]"^(){}`, c):
		// A comment reads to the end of the line, so the map is not closed.
		// The other characters are not allowed or not matched.
		err = errInvalidEDN
	case isASCIIDigit(c) || ((c == '+' || c == '-') && isASCIIDigit(r.peek())):
		form, err = r.readNumber(c)
	default:
		form, err = r.readSymbol(c)
	}

	return form, err == nil, err
}

func (r *ednReader) readKeyword() (string, error) {
	c := r.next()
	if c == ednEnd || ednWhitespace(c) {
		return "", errInvalidEDN
	}

	token, err := r.readToken(c, false)
	if err != nil || token == "" || strings.Contains(token, "::") || token[0] == ':' || !parseEDNSymbol(token) {
		return "", errInvalidEDN
	}

	return ":" + token, nil
}

func (r *ednReader) readSymbol(initial rune) (string, error) {
	token, err := r.readToken(initial, false)
	if err != nil {
		return "", err
	}

	switch token {
	case "nil", "true", "false", "/":
		return token, nil
	}

	if !parseEDNSymbol(token) {
		return "", errInvalidEDN
	}

	return token, nil
}

// readToken reads the characters of a symbol, a keyword or a character,
// starting with the given character, up to a space or a character that ends a
// token. The token is empty if the given character ends it.
func (r *ednReader) readToken(initial rune, character bool) (string, error) {
	if !character && ednNotConstituent(initial) {
		return "", errInvalidEDN
	}

	if ednWhitespace(initial) || ednMacroTerminating(initial) {
		r.pos--
		return "", nil
	}

	var token strings.Builder
	token.WriteRune(initial)
	for {
		c := r.peek()
		if c == ednEnd || ednWhitespace(c) || ednMacroTerminating(c) {
			return token.String(), nil
		}

		if ednNotConstituent(c) {
			return "", errInvalidEDN
		}

		token.WriteRune(r.next())
	}
}

// readCharacter reads a character literal such as `\c` or `\newline`, after
// its backslash.
func (r *ednReader) readCharacter() (string, error) {
	c := r.next()
	if c == ednEnd {
		return "", errInvalidEDN
	}

	token := string(c)
	if !ednMacroTerminating(c) && !ednNotConstituent(c) && !ednWhitespace(c) {
		var err error
		if token, err = r.readToken(c, true); err != nil {
			return "", err
		}
	}

	// The reader counts the length of the token in UTF-16 code units, so a
	// character outside the Basic Multilingual Plane is not one character.
	switch {
	case len(utf16.Encode([]rune(token))) == 1:
		return `\` + token, nil
	case token == "newline", token == "space", token == "tab", token == "backspace", token == "formfeed", token == "return":
		return `\` + token, nil
	case strings.HasPrefix(token, "u"):
		code, err := strconv.ParseUint(token[1:], 16, 32)
		if len(token) != 5 || err != nil || (code > 0xd7ff && code < 0xe000) {
			return "", errInvalidEDN
		}

		return `\` + string(rune(code)), nil
	case strings.HasPrefix(token, "o"):
		code, err := strconv.ParseUint(token[1:], 8, 32)
		if len(token) > 4 || err != nil || code > 0o377 {
			return "", errInvalidEDN
		}

		return `\` + string(rune(code)), nil
	}

	return "", errInvalidEDN
}

// readDispatch reads a form that starts with `#`. Only a discarded form,
// `#_x`, and the symbolic values `##Inf`, `##-Inf` and `##NaN` can be read
// without a string or braces. A tagged form such as `#inst x` needs a reader
// that the EDN reader does not have.
func (r *ednReader) readDispatch() (string, bool, error) {
	switch r.next() {
	case '_':
		_, err := r.readKeptForm()
		return "", false, err
	case '#':
		form, err := r.readKeptForm()
		if err != nil || (form != "Inf" && form != "-Inf" && form != "NaN") {
			return "", false, errInvalidEDN
		}

		return "##" + form, true, nil
	}

	return "", false, errInvalidEDN
}

var (
	ednIntPattern   = regexp.MustCompile(`^([-+]?)(?:(0)|([1-9][0-9]*)|0[xX]([0-9A-Fa-f]+)|0([0-7]+)|([1-9][0-9]?)[rR]([0-9A-Za-z]+)|0[0-9]+)(N)?$`)
	ednFloatPattern = regexp.MustCompile(`^([-+]?[0-9]+(\.[0-9]*)?([eE][-+]?[0-9]+)?)(M)?$`)
	ednRatioPattern = regexp.MustCompile(`^([-+]?[0-9]+)/([0-9]+)$`)
)

// readNumber reads a number, which ends at a space or at any character that
// starts another form.
func (r *ednReader) readNumber(initial rune) (string, error) {
	var token strings.Builder
	token.WriteRune(initial)
	for {
		c := r.peek()
		if c == ednEnd || ednWhitespace(c) || ednMacro(c) {
			break
		}

		token.WriteRune(r.next())
	}

	s := token.String()
	if m := ednIntPattern.FindStringSubmatch(s); m != nil {
		// A number such as 08 matches the pattern but is not a number. A
		// number with a radix must start with a digit of that radix.
		switch {
		case m[2] != "", m[3] != "", m[4] != "", m[5] != "":
			return s, nil
		case m[7] != "":
			radix, _ := strconv.Atoi(m[6])
			if radix < 2 || radix > 36 {
				return "", errInvalidEDN
			}

			if _, err := strconv.ParseInt(m[7][:1], radix, 64); err != nil {
				return "", errInvalidEDN
			}

			return s, nil
		}

		return "", errInvalidEDN
	}

	if ednFloatPattern.MatchString(s) || ednRatioPattern.MatchString(s) {
		return s, nil
	}

	return "", errInvalidEDN
}

// parseEDNSymbol checks if a token is a valid name for a symbol or a keyword:
// it can have a namespace in front of a slash, as in `ns/name`, and the name
// does not start with a digit.
func parseEDNSymbol(token string) bool {
	if token == "" || strings.HasSuffix(token, ":") || strings.HasPrefix(token, "::") {
		return false
	}

	slash := strings.IndexByte(token, '/')
	if slash <= 0 {
		return token == "/" || slash == -1
	}

	ns, name := token[:slash], token[slash+1:]
	if name == "" || isASCIIDigit(rune(name[0])) || strings.HasSuffix(ns, ":") {
		return false
	}

	return name == "/" || !strings.Contains(name, "/")
}

// ednWhitespace checks if the EDN reader reads a character as a space. It
// reads a comma as a space, and the characters that `\s` matches in
// JavaScript.
func ednWhitespace(c rune) bool {
	switch c {
	case ',', ' ', '\t', '\n', '\v', '\f', '\r',
		'\u00a0', '\u1680', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000', '\ufeff':
		return true
	}

	return c >= '\u2000' && c <= '\u200a'
}

// ednMacro checks if a character starts a form that is not a symbol or a
// number.
func ednMacro(c rune) bool {
	return strings.ContainsRune(`":;^()[]{}\#`, c)
}

// ednMacroTerminating checks if a character ends a symbol, a keyword or a
// character.
func ednMacroTerminating(c rune) bool {
	return c != '#' && c != ':' && ednMacro(c)
}

// ednNotConstituent checks if a character can not be a part of a symbol or a
// keyword.
func ednNotConstituent(c rune) bool {
	return c == '@' || c == '`' || c == '~'
}

func isASCIIDigit(c rune) bool {
	return c >= '0' && c <= '9'
}
