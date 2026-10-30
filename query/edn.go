package query

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

type ednKind int

const (
	ednList ednKind = iota
	ednVector
	ednMap
	ednSet
	ednString
	ednSymbol
	ednKeyword
	ednNumber
	ednRegex
	ednTagged
	ednChar
)

// ednNode is a value read from EDN. Lists and vectors are kept apart, as
// Datalog gives them different meanings.
type ednNode struct {
	kind ednKind

	// text is the value of a string or regex, the name of a symbol, the name
	// of a keyword without the colon, the text of a number or character, or
	// the name of a tag.
	text string

	// items are the values in a list, vector, set or tagged value. The keys
	// and values of a map are in turns.
	items []*ednNode

	// pos and end are the byte offsets where the value starts and ends.
	pos int
	end int
}

func (n *ednNode) isSymbol(name string) bool {
	return n.kind == ednSymbol && n.text == name
}

// head returns the name of the symbol that a list starts with.
func (n *ednNode) head() string {
	if n.kind != ednList || len(n.items) == 0 || n.items[0].kind != ednSymbol {
		return ""
	}

	return n.items[0].text
}

// ednReader reads the EDN that Logseq writes queries in. It reads the parts of
// Clojure that are found in queries as well, such as `#"regex"` and `#(...)`.
type ednReader struct {
	input string
	pos   int
}

func (r *ednReader) skipSpace() error {
	for r.pos < len(r.input) {
		c, size := utf8.DecodeRuneInString(r.input[r.pos:])
		switch {
		case isSpace(c):
			r.pos += size
		case c == ';':
			for r.pos < len(r.input) && r.input[r.pos] != '\n' {
				r.pos++
			}
		case strings.HasPrefix(r.input[r.pos:], "#_"):
			// The value after #_ is left out.
			r.pos += 2
			if _, err := r.read(); err != nil {
				return err
			}
		default:
			return nil
		}
	}

	return nil
}

func (r *ednReader) read() (*ednNode, error) {
	if err := r.skipSpace(); err != nil {
		return nil, err
	}

	if r.pos >= len(r.input) {
		return nil, errorAt(r.pos, "unexpected end of the query")
	}

	start := r.pos
	node, err := r.readValue()
	if err != nil {
		return nil, err
	}

	node.pos = start
	node.end = r.pos
	return node, nil
}

func (r *ednReader) readValue() (*ednNode, error) {
	start := r.pos
	c := r.input[r.pos]

	switch c {
	case '(':
		r.pos++
		return r.readCollection(ednList, ')', start)
	case '[':
		r.pos++
		return r.readCollection(ednVector, ']', start)
	case '{':
		r.pos++
		node, err := r.readCollection(ednMap, '}', start)
		if err != nil {
			return nil, err
		}

		if len(node.items)%2 != 0 {
			return nil, errorAt(start, "a map needs a value for each key")
		}
		return node, nil
	case ')', ']', '}':
		return nil, errorAt(r.pos, "unexpected `%c`", c)
	case '"':
		value, err := r.readString(true)
		if err != nil {
			return nil, err
		}
		return &ednNode{kind: ednString, text: value}, nil
	case '\\':
		r.pos++
		return &ednNode{kind: ednChar, text: r.readToken()}, nil
	case '\'':
		// A quoted value is read as the value itself.
		r.pos++
		return r.read()
	case '#':
		r.pos++
		if r.pos >= len(r.input) {
			return nil, errorAt(start, "unexpected end of the query")
		}

		switch r.input[r.pos] {
		case '{':
			r.pos++
			return r.readCollection(ednSet, '}', start)
		case '(':
			// An anonymous function such as #(get % :block/priority).
			r.pos++
			node, err := r.readCollection(ednList, ')', start)
			if err != nil {
				return nil, err
			}
			return &ednNode{kind: ednTagged, text: "fn", items: []*ednNode{node}}, nil
		case '"':
			value, err := r.readString(false)
			if err != nil {
				return nil, err
			}
			return &ednNode{kind: ednRegex, text: value}, nil
		}

		tag := r.readToken()
		if tag == "" {
			return nil, errorAt(start, "expected a tag after `#`")
		}

		value, err := r.read()
		if err != nil {
			return nil, err
		}
		return &ednNode{kind: ednTagged, text: tag, items: []*ednNode{value}}, nil
	}

	token := r.readToken()
	switch {
	case strings.HasPrefix(token, ":"):
		return &ednNode{kind: ednKeyword, text: token[1:]}, nil
	case ednNumberPattern.MatchString(token):
		return &ednNode{kind: ednNumber, text: token}, nil
	}

	return &ednNode{kind: ednSymbol, text: token}, nil
}

var ednNumberPattern = regexp.MustCompile(`^[+-]?\d`)

func (r *ednReader) readCollection(kind ednKind, end byte, start int) (*ednNode, error) {
	node := &ednNode{kind: kind}
	for {
		if err := r.skipSpace(); err != nil {
			return nil, err
		}

		if r.pos >= len(r.input) {
			return nil, errorAt(start, "missing `%c`", end)
		}

		if r.input[r.pos] == end {
			r.pos++
			return node, nil
		}

		item, err := r.read()
		if err != nil {
			return nil, err
		}
		node.items = append(node.items, item)
	}
}

// readString reads a string in quotes. Escapes are turned into the characters
// they stand for if unescape is set, and kept as they are for regexes.
func (r *ednReader) readString(unescape bool) (string, error) {
	start := r.pos
	r.pos++

	var b strings.Builder
	for r.pos < len(r.input) {
		c := r.input[r.pos]
		switch c {
		case '"':
			r.pos++
			return b.String(), nil
		case '\\':
			r.pos++
			if r.pos >= len(r.input) {
				return "", errorAt(start, "missing `\"`")
			}

			e := r.input[r.pos]
			if !unescape {
				b.WriteByte('\\')
				b.WriteByte(e)
				break
			}

			switch e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			default:
				b.WriteByte(e)
			}
		default:
			b.WriteByte(c)
		}
		r.pos++
	}

	return "", errorAt(start, "missing `\"`")
}

// readToken reads a symbol, keyword, number or character up to the next
// delimiter.
func (r *ednReader) readToken() string {
	start := r.pos
	for r.pos < len(r.input) {
		c, size := utf8.DecodeRuneInString(r.input[r.pos:])
		if isSpace(c) || strings.ContainsRune("()[]{}\";", c) {
			break
		}
		r.pos += size
	}

	return r.input[start:r.pos]
}
