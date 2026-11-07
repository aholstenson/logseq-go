package markdown

import (
	"strings"
	"unicode"

	"github.com/aholstenson/logseq-go/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// macroCheckParser reads only paragraphs and macros. It is used to check if
// text starts with a macro, see startsMacro.
var macroCheckParser = parser.NewParser(
	parser.WithBlockParsers(util.Prioritized(parser.NewParagraphParser(), 1000)),
	parser.WithInlineParsers(util.Prioritized(&macroParser{}, 100)),
)

// startsMacro returns true if the text is read as a macro at its start, such
// as "{{embed [[a]]}} b". Text such as "{{ name }}" is not a macro.
func startsMacro(str string) bool {
	src := []byte(str)
	doc := macroCheckParser.Parse(text.NewReader(src))
	paragraph := doc.FirstChild()
	if paragraph == nil {
		return false
	}

	_, ok := paragraph.FirstChild().(*macro)
	return ok
}

var macroKind = ast.NewNodeKind("Macro")

// macro is the Logseq macro structure which is in the form of {{macro-name arg1, arg2}}.
// Three curly braces are also supported, such as {{{macro-name arg1, arg2}}}.
// Arguments are optional, but must be comma separated. Arguments can be quoted,
// in which case they can contain commas. A quoted argument keeps its quotes
// and backslashes, in the same way as Logseq.
//
// Some examples of how Logseq parses arguments:
//
// - {{poem red, blue}} - 2 arguments: `red`, `blue`
// - {{poem red,blue}} - 2 arguments: `red`, `blue`
// - {{poem red blue}} - 1 argument: `red blue`
// - {{poem "red", "blue"}} - 2 arguments: `"red"`, `"blue"`
// - {{poem "blue," , red}} - 2 arguments: `"blue,"`, `red`
// - {{poem "blue," red}} - invalid, quoted and no comma between arguments
// - {{poem red,}} - invalid, empty arguments
// - {{poem red,,}} - invalid, empty arguments
// - {{poem red,",blue"}} - 2 arguments: `red`, `",blue"`
// - {{poem "red\"",blue}} - 2 arguments: `"red\""`, `blue`
// - {{poem "r\ed",blue}} - 2 arguments: `"r\ed"`, `blue`
type macro struct {
	ast.BaseInline
	Name      string
	Arguments []string

	// RawArguments is the text after the name, without the spaces around it.
	// Logseq joins the arguments of some macros again, such as `query`, so a
	// comma in the text does not split it.
	RawArguments string

	// Source is the text of the macro, from `{{` to `}}`.
	Source string
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

// macroPrefixes are the characters after which Logseq can start a macro. They
// are the characters that end a run of plain text in the Markdown parser of
// Logseq (mldoc), as for block references, see blockRefPrefixes. After `_`, `^`
// and `#`, Logseq reads a subscript, a superscript or a tag first, so a macro
// does not start there.
const macroPrefixes = " \t\n\r\f\\[*~`=${"

// canStartMacro returns true if a macro can start at the current position.
// Logseq reads a run of plain text as one piece, so `x{{m}}` is text, see
// canStartAfterText.
func canStartMacro(parent ast.Node, block text.Reader) bool {
	return canStartAfterText(parent, block, macroPrefixes)
}

// macroCanFollow returns true if a macro can start directly after the
// Markdown text.
func macroCanFollow(text []byte) bool {
	return canFollowText(text, macroPrefixes)
}

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

	if !canStartMacro(parent, block) {
		// Logseq reads `x{{m [[q]]}}` as text with a page link.
		return nil
	}

	if !comesLater(block, pc, "}}", 2) {
		// Without a closing }} on the line, this isn't a macro.
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
	// argumentsStart is where the text of the arguments starts.
	argumentsStart := -1
	rawArguments := ""

	// Scan until the closing }} while also dealing with escaped } characters.
	state := macroParseStateName
	for i := start; i < len(line)-1; i++ {
		// A macro opened with three curly braces needs a third } to close, which
		// has to be on this line for the macro to end here.
		if line[i] == '}' && line[i+1] == '}' && (!tripleCurly || (i+2 < len(line) && line[i+2] == '}')) {
			// If the previous character was a \, then this is an escaped } and
			// we should continue scanning.
			if line[i-1] == '\\' {
				continue
			}

			end = i
			if tripleCurly {
				end += 1
			}

			if argumentsStart >= 0 {
				rawArguments = strings.TrimSpace(string(line[argumentsStart:i]))
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
				argumentsStart = start
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
				// Quote is being ended. The argument keeps its quotes.
				arguments = append(arguments, string(line[start:i+1]))

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
		Name:         name,
		Arguments:    arguments,
		RawArguments: rawArguments,
		Source:       string(line[:end+2]),
	}
}

// readsBackAsMacro returns true if the source of a macro is read back as the
// same node. This is false after the node was changed, and then the macro is
// written from the fields of the node.
func readsBackAsMacro(source string, node content.Node) bool {
	parsed := (&macroParser{}).Parse(nil, text.NewReader([]byte(source)), parser.NewContext())
	m, ok := parsed.(*macro)
	if !ok || m.Source != source {
		return false
	}

	switch n := node.(type) {
	case *content.Macro:
		other, ok := macroNodeOf(m).(*content.Macro)
		return ok && other.Name == n.Name && equalStrings(other.Arguments, n.Arguments)
	case *content.Query:
		other, ok := macroNodeOf(m).(*content.Query)
		return ok && other.Query == n.Query
	case *content.PageEmbed:
		other, ok := macroNodeOf(m).(*content.PageEmbed)
		return ok && other.To == n.To
	case *content.BlockEmbed:
		other, ok := macroNodeOf(m).(*content.BlockEmbed)
		return ok && other.ID == n.ID
	case *content.Cloze:
		other, ok := macroNodeOf(m).(*content.Cloze)
		return ok && other.Answer == n.Answer && other.Cue == n.Cue
	}

	return false
}

func equalStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// isQuotedMacroArgument returns true if the argument is one quoted argument,
// which the macro parser reads as it is. The first quote that is not escaped
// after the opening quote must be the last character.
func isQuotedMacroArgument(arg string) bool {
	if len(arg) < 2 || arg[0] != '"' {
		return false
	}

	for i := 1; i < len(arg); i++ {
		if arg[i] == '"' && arg[i-1] != '\\' {
			return i == len(arg)-1
		}
	}

	return false
}
