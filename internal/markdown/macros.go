package markdown

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var macroKind = ast.NewNodeKind("Macro")

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
	lookingForName := true
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
			if lookingForName {
				name = string(line[start:i])
			} else {
				value := strings.TrimSpace(string(line[start:i]))
				if len(value) != 0 {
					arguments = append(arguments, value)
				}
			}
			break
		}

		if lookingForName {
			if line[i] == ' ' {
				lookingForName = false
				name = string(line[start:i])
				start = i + 1
			}
		} else {
			if line[i] == '"' {
				// Scan until the closing " while also dealing with escape
				// sequences, such as \" and \\.
				for j := i + 1; j < len(line)-1; j++ {
					if line[j] == '"' && line[j-1] != '\\' {
						// Add the argument.
						value := unescapeString(line[i+1 : j])
						arguments = append(arguments, value)

						// Skip the closing ".
						i = j
						start = j + 1
						break
					}
				}
			} else if line[i] == ' ' {
				// Check if the argument is empty, if so, skip it.
				value := strings.TrimSpace(string(line[start:i]))
				if len(value) != 0 {
					arguments = append(arguments, value)
					start = i + 1
				}
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
