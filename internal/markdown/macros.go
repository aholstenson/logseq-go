package markdown

import (
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var macroKind = ast.NewNodeKind("Macro")

// macro is the Logseq macro structure which is in the form of {{macro-name arg1, arg2}}.
// Three curly braces are also supported, such as {{{macro-name arg1, arg2}}}.
// Arguments are optional, but must be comma separated. Arguments can be quoted,
// in which case they can contain commas. When rendering in the Logseq editor
// quotes are not removed.
//
// Some examples of how Logseq parses and displays arguments:
//
// - {{poem red, blue}} - 2 arguments: `red`, `blue`
// - {{poem red,blue}} - 2 arguments: `red`, `blue`
// - {{poem red blue}} - 1 argument: `red blue`
// - {{poem "red", "blue"}} - 2 arguments: `red`, `blue`
// - {{poem "blue," , red}} - 2 arguments: `"blue,"`, `red`
// - {{poem "blue," red}} - invalid, quoted and no comma between arguments
// - {{poem red,}} - invalid, empty arguments
// - {{poem red,,}} - invalid, empty arguments
// - {{poem red,",blue"}} - 2 arguments: `red`, `",blue"`
// - {{poem "red\"",blue}} - 2 arguments: `"red\""`, `blue`
// - {{poem "r\\ed",blue}} - 2 arguments: `"r\\ed"`, `blue`
// - {{poem "r\ed",blue}} - 2 arguments: `"red"`, `blue`
// - {{poem "red\\",blue}} - 2 arguments: `"red\\"`, `blue`
type macro struct {
	ast.BaseInline
	Name      string
	Arguments []string
}

func (*macro) Kind() ast.NodeKind {
	return macroKind
}

func (n *macro) Dump(src []byte, level int) {
}

type macroParseState int

const (
	macroParseStateName macroParseState = iota
	macroParseStateArgumentStart
	macroParseStateArgumentNonQuoted
	macroParseStateArgumentQuote
	macroParseStateExpectComma
)

// macroParser parses Logseq macros which are in the form of {{macro-name arg1 arg2}}.
type macroParser struct {
}

func (t *macroParser) Trigger() []byte {
	return []byte{'{'}
}

func (t *macroParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()

	start := 2
	if len(line) < 2 || line[0] != '{' || line[1] != '{' {
		return nil
	}

	// Check if there is an extra {
	tripleCurly := false
	if len(line) > 2 && line[2] == '{' {
		start = 3
		tripleCurly = true
	}

	end := 0

	// Name is the text between the {{ and the first space or first opening parenthesis.
	name := ""
	// Arguments are anything after the name, we support quoted arguments and escaped quotes.
	arguments := []string{}

	// Scan until the closing }} while also dealing with escaped } characters.
	state := macroParseStateName
	for i := start; i < len(line)-1; i++ {
		if line[i] == '}' && line[i+1] == '}' && (!tripleCurly || (tripleCurly && line[i+2] == '}')) {
			// If the previous character was a \, then this is an escaped } and
			// we should continue scanning.
			if line[i-1] == '\\' {
				continue
			}

			end = i
			if tripleCurly {
				end += 1
			}

			// Add the name or last argument if there is one.
			switch state {
			case macroParseStateName:
				name = string(line[start:i])
			case macroParseStateArgumentStart, macroParseStateArgumentQuote, macroParseStateArgumentNonQuoted:
				value := strings.TrimSpace(string(line[start:i]))
				if len(value) != 0 {
					arguments = append(arguments, value)
				} else {
					return nil
				}
			}

			break
		}

		switch state {
		case macroParseStateName:
			// Names are always separated by a space, other types of whitespace
			// are not supported.
			if line[i] == ' ' {
				name = string(line[start:i])

				state = macroParseStateArgumentStart
				start = i + 1
			}
		case macroParseStateArgumentStart:
			// This is the start of an argument, which can be quoted or not.
			//
			// When a quote character is found at the start of an argument
			// Logseq will treat the argument as quoted, but will not do the
			// same if the quote is found in the middle of an argument.
			if line[i] == '"' {
				state = macroParseStateArgumentQuote
				start = i
			} else if line[i] == ',' {
				// Invalid, empty argument
				return nil
			} else if !unicode.IsSpace(rune(line[i])) {
				// This is the start of a non-quoted argument, switch the
				// state and set the start position.
				state = macroParseStateArgumentNonQuoted
				start = i
			}
		case macroParseStateArgumentQuote:
			if line[i] == '"' && line[i-1] != '\\' {
				// Quote is being ended
				value := unescapeString(line[start+1 : i])
				arguments = append(arguments, value)

				state = macroParseStateExpectComma
				start = i + 1
			}
		case macroParseStateArgumentNonQuoted:
			if line[i] == ',' {
				// End of the argument as a comma was found.
				value := strings.TrimSpace(string(line[start:i]))
				if len(value) != 0 {
					arguments = append(arguments, value)
				} else {
					// Invalid, empty argument
					return nil
				}

				state = macroParseStateArgumentStart
				start = i + 1
			}
		case macroParseStateExpectComma:
			if line[i] == ',' {
				state = macroParseStateArgumentStart
				start = i + 1
			} else if !unicode.IsSpace(rune(line[i])) {
				// Invalid, no comma between arguments
				return nil
			}
		}
	}

	// Didn't find the closing }}, so this isn't a macro.
	if end == 0 {
		return nil
	}

	// Make sure there is a name.
	if len(name) == 0 {
		return nil
	}

	block.Advance(end + 2)

	return &macro{
		Name:      name,
		Arguments: arguments,
	}
}
