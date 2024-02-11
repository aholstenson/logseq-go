package markdown

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"github.com/aholstenson/logseq-go/content"
)

var urlRegexp = regexp.MustCompile(`^(?:http|https|ftp)://[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-z]+(?::\d+)?(?:[/#?][-a-zA-Z0-9@:%_+.~#$!?&/=\(\);,'">\^{}\[\]` + "`" + `]*)?`)

type EscapeFunc func(rune, rune) bool

func EscapeNone(rune, rune) bool {
	return false
}

func EscapePotentialMarkdown(prev rune, r rune) bool {
	if r == '*' || r == '_' || r == '[' || r == ']' {
		return true
	}

	if prev == '~' && r == '~' {
		return true
	}

	return false
}

func EscapeLinkURL(prev rune, r rune) bool {
	return r == '(' || r == ')'
}

func EscapeLinkTitle(prev rune, r rune) bool {
	return r == '"' || r == '\'' || r == ')'
}

func EscapeWikiLink(prev rune, r rune) bool {
	return r == ']'
}

func EscapeBlockRef(prev rune, r rune) bool {
	return r == ')'
}

func EscapeMacroQuotedArgument(prev rune, r rune) bool {
	return r == '"'
}

func EscapeString(str string, f EscapeFunc) string {
	out := strings.Builder{}
	p := rune(0)
	for _, r := range str {
		if f(p, r) {
			out.WriteRune('\\')
		}

		out.WriteRune(r)
		p = r
	}

	return out.String()
}

// Output is used to write Markdown to an output buffer. It will help keep
// track of list indentation and when to add newlines.
type Output struct {
	out *writer
}

// NewWriter creates a new Markdown writer.
func NewWriter(out io.Writer) *Output {
	return &Output{
		out: newWriter(out),
	}
}

func AsString(n content.Node) (string, error) {
	out := strings.Builder{}
	w := NewWriter(&out)
	if err := w.Write(n); err != nil {
		return "", err
	}

	return out.String(), nil
}

func Write(n content.Node, out io.Writer) error {
	w := NewWriter(out)
	return w.Write(n)
}

func (w *Output) Write(n content.Node) error {
	switch node := n.(type) {
	case *content.RawHTML:
		return w.writeRaw(node.HTML)
	case *content.Text:
		return w.writeText(node)
	case *content.Emphasis:
		return w.writeEmphasis(node)
	case *content.Strong:
		return w.writeStrong(node)
	case *content.Strikethrough:
		return w.writeStrikethrough(node)
	case *content.CodeSpan:
		return w.writeCodeSpan(node)
	case *content.Link:
		return w.writeLink(node)
	case *content.AutoLink:
		return w.writeAutoLink(node)
	case *content.PageLink:
		return w.writePageLink(node)
	case *content.Hashtag:
		return w.writeHashtag(node)
	case *content.BlockRef:
		return w.writeBlockRef(node)
	case *content.Image:
		return w.writeImage(node)
	case *content.Macro:
		return w.writeMacro(node, node.Name, node.Arguments)
	case *content.Query:
		return w.writeMacro(node, "query", []string{node.Query})
	case *content.PageEmbed:
		return w.writeMacro(node, "embed", []string{"[[" + EscapeString(node.To, EscapeWikiLink) + "]]"})
	case *content.BlockEmbed:
		return w.writeMacro(node, "embed", []string{"((" + EscapeString(node.ID, EscapeBlockRef) + "))"})
	case *content.Cloze:
		return w.writeCloze(node)
	case *content.Heading:
		return w.writeHeading(node)
	case *content.RawHTMLBlock:
		return w.writeRawHTMLBlock(node)
	case *content.Paragraph:
		return w.writeParagraph(node)
	case *content.List:
		return w.writeList(node)
	case *content.Blockquote:
		return w.writeBlockquote(node)
	case *content.CodeBlock:
		return w.writeCodeBlock(node)
	case *content.ThematicBreak:
		return w.writeThematicBreak(node)
	case *content.Block:
		return w.writeBlock(node)
	case *content.Properties:
		return w.writeProperties(node)
	case *content.AdvancedCommand:
		return w.writeAdvancedCommand(node)
	case *content.QueryCommand:
		return w.writeBeginEnd(node, "QUERY", node.Query)
	case *content.TaskMarker:
		return w.writeTaskMarker(node)
	case *content.Logbook:
		return w.writeLogbook(node)
	default:
		return fmt.Errorf("unsupported node: %T", node)
	}
}

func (w *Output) writeRaw(s string) error {
	return w.out.WriteString(s)
}

func (w *Output) write(s string, escapeFunc EscapeFunc) error {
	out := strings.Builder{}
	p := rune(0)
	for _, r := range s {
		if escapeFunc(p, r) {
			out.WriteRune('\\')
		}

		out.WriteRune(r)
		p = r
	}

	return w.writeRaw(out.String())
}

func (w *Output) startBlock(node content.BlockNode, marker string) error {
	return w.startBlockWithAutomaticBehavior(node, marker, true)
}

func (w *Output) startBlockWithAutomaticBehavior(node content.BlockNode, marker string, doubleNewLineForAutomatic bool) error {
	if w.out.HasWrittenAtCurrentIndent() {
		var prefix string
		if pl, ok := node.(content.PreviousLineAware); ok {
			switch pl.PreviousLineType() {
			case content.PreviousLineTypeBlank:
				prefix = "\n\n"
			case content.PreviousLineTypeNonBlank:
				prefix = "\n"
			case content.PreviousLineTypeAutomatic:
				if doubleNewLineForAutomatic {
					prefix = "\n\n"
				} else {
					prefix = "\n"
				}
			default:
				return fmt.Errorf("unknown previous line type: %d", pl.PreviousLineType())
			}
		} else {
			if doubleNewLineForAutomatic {
				prefix = "\n\n"
			} else {
				prefix = "\n"
			}
		}

		err := w.out.WriteString(prefix)
		if err != nil {
			return err
		}
	}

	w.out.PushIndentation(marker)
	return nil
}

func (w *Output) endBlock() {
	w.out.PopIndentation()
}

func (w *Output) writeChildren(node content.HasChildren) error {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		err := w.Write(child)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Output) writeText(node *content.Text) error {
	err := w.write(node.Value, EscapePotentialMarkdown)
	if err != nil {
		return err
	}

	if node.SoftLineBreak {
		err = w.writeRaw("\n")
		if err != nil {
			return err
		}
	} else if node.HardLineBreak {
		err = w.writeRaw("\\\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Output) writeEmphasis(node *content.Emphasis) error {
	if _, ok := node.PreviousSibling().(*content.Emphasis); ok {
		// Writing two emphasis nodes next to each other is not valid Markdown,
		// so we add a space between them as a compromise.
		err := w.writeRaw(" ")
		if err != nil {
			return err
		}
	}

	err := w.writeRaw("*")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("*")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeStrong(node *content.Strong) error {
	if _, ok := node.PreviousSibling().(*content.Strong); ok {
		// Writing two strong nodes next to each other is not valid Markdown,
		// so we add a space between them as a compromise.
		err := w.writeRaw(" ")
		if err != nil {
			return err
		}
	}

	err := w.writeRaw("**")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("**")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeStrikethrough(node *content.Strikethrough) error {
	err := w.writeRaw("~~")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("~~")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeCodeSpan(node *content.CodeSpan) error {
	// First find the longest sequence of backticks in the value so can use
	// the correct marker.
	longestSequence := 0
	for i := 0; i < len(node.Value); i++ {
		if node.Value[i] != '`' {
			continue
		}

		if longestSequence == 0 {
			longestSequence = 1
		} else if node.Value[i-1] == '`' {
			longestSequence++
		}
	}
	marker := strings.Repeat("`", longestSequence+1)

	err := w.writeRaw(marker)
	if err != nil {
		return err
	}

	err = w.writeRaw(node.Value)
	if err != nil {
		return err
	}

	err = w.writeRaw(marker)
	if err != nil {
		return err
	}
	return nil
}

func (w *Output) writeLink(node *content.Link) error {
	err := w.writeRaw("[")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("](")
	if err != nil {
		return err
	}

	err = w.write(node.URL, EscapeLinkURL)
	if err != nil {
		return err
	}

	if node.Title != "" {
		err = w.writeRaw(" '")
		if err != nil {
			return err
		}

		err = w.write(node.Title, EscapeLinkTitle)
		if err != nil {
			return err
		}

		err = w.writeRaw("'")
		if err != nil {
			return err
		}
	}

	err = w.writeRaw(")")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeAutoLink(node *content.AutoLink) error {
	if urlRegexp.Match([]byte(node.URL)) {
		// No need for brackets, Logseq will automatically linkify the URL.
		return w.writeRaw(node.URL)
	}

	err := w.writeRaw("<")
	if err != nil {
		return err
	}

	err = w.writeRaw(node.URL)
	if err != nil {
		return err
	}

	err = w.writeRaw(">")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writePageLink(node *content.PageLink) error {
	err := w.writeRaw("[[")
	if err != nil {
		return err
	}

	err = w.write(node.To, EscapeWikiLink)
	if err != nil {
		return err
	}

	err = w.writeRaw("]]")
	if err != nil {
		return err
	}

	return nil
}

// writeHashtag writes *content.PageLink as `#to` or `#[[to]]`. The extended
// syntax is used if the target contains whitespace.
func (w *Output) writeHashtag(node *content.Hashtag) error {
	err := w.writeRaw("#")
	if err != nil {
		return err
	}

	writeExtended := false
	for _, r := range node.To {
		if unicode.IsSpace(r) {
			writeExtended = true
			break
		}
	}

	if writeExtended {
		err = w.writeRaw("[[")
		if err != nil {
			return err
		}
	}

	err = w.write(node.To, EscapeWikiLink)
	if err != nil {
		return err
	}

	if writeExtended {
		err = w.writeRaw("]]")
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Output) writeBlockRef(node *content.BlockRef) error {
	err := w.writeRaw("((")
	if err != nil {
		return err
	}

	err = w.write(node.ID, EscapeWikiLink)
	if err != nil {
		return err
	}

	err = w.writeRaw("))")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeImage(node *content.Image) error {
	err := w.writeRaw("![")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("](")
	if err != nil {
		return err
	}

	err = w.write(node.URL, EscapeLinkURL)
	if err != nil {
		return err
	}

	if node.Title != "" {
		err = w.writeRaw(" '")
		if err != nil {
			return err
		}

		err = w.write(node.Title, EscapeLinkTitle)
		if err != nil {
			return err
		}

		err = w.writeRaw("'")
		if err != nil {
			return err
		}
	}

	err = w.writeRaw(")")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeCloze(node *content.Cloze) error {
	cue := strings.TrimSpace(node.Cue)
	answer := strings.TrimSpace(node.Answer)
	if cue != "" {
		return w.writeMacro(node, "cloze", []string{answer + " \\ " + cue})
	} else {
		return w.writeMacro(node, "cloze", []string{answer})
	}
}

func (w *Output) writeMacro(node content.Node, name string, arguments []string) error {
	err := w.writeRaw("{{")
	if err != nil {
		return err
	}

	// Validate the macro name, it can not contain whitespace.
	for _, r := range name {
		if unicode.IsSpace(r) {
			return fmt.Errorf("macro name can not contain whitespace")
		}
	}

	err = w.writeRaw(name)
	if err != nil {
		return err
	}

	if arguments != nil {
		for i, arg := range arguments {
			if i == 0 {
				err = w.writeRaw(" ")
				if err != nil {
					return err
				}
			} else {
				err = w.writeRaw(", ")
				if err != nil {
					return err
				}
			}

			// Check if the argument contains a comma, if so we need to quote
			// the argument.
			quoted := false
			for _, r := range arg {
				if r == ',' {
					quoted = true
					break
				}
			}

			if quoted {
				err = w.writeRaw("\"")
				if err != nil {
					return err
				}

				err = w.write(arg, EscapeMacroQuotedArgument)
				if err != nil {
					return err
				}

				err = w.writeRaw("\"")
				if err != nil {
					return err
				}
			} else {
				err = w.write(arg, EscapeNone)
				if err != nil {
					return err
				}
			}
		}
	}

	err = w.writeRaw("}}")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeRawHTMLBlock(node *content.RawHTMLBlock) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw(node.HTML)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeHeading(node *content.Heading) error {
	err := w.startBlock(node, strings.Repeat("#", node.Level)+" ")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeParagraph(node *content.Paragraph) error {
	doubleNewLine := true
	if _, previousProperties := node.PreviousSibling().(*content.Properties); previousProperties {
		doubleNewLine = false
	}

	err := w.startBlockWithAutomaticBehavior(node, "", doubleNewLine)
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeList(node *content.List) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	i := 0
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*content.ListItem); !ok {
			return fmt.Errorf("unsupported list child: %T", child)
		}

		i++
		var marker string
		if node.Type == content.ListTypeOrdered {
			marker = fmt.Sprintf("%d", i) + string(node.Marker)
		} else {
			marker = string(node.Marker)
		}

		err := w.out.WriteString(marker + " ")
		if err != nil {
			return err
		}

		w.out.PushIndentation(strings.Repeat(" ", len(marker)+1))

		err = w.writeChildren(child.(content.HasChildren))
		if err != nil {
			return err
		}

		if child.NextSibling() != nil {
			err = w.writeRaw("\n")
			if err != nil {
				return err
			}
		}

		w.out.PopIndentation()
	}

	w.endBlock()
	return nil
}

func (w *Output) writeBlockquote(node *content.Blockquote) error {
	err := w.startBlock(node, "> ")
	if err != nil {
		return err
	}

	if !w.out.lastWasLineBreak {
		// This is a hack to make sure that the indicator is written in lists
		// if the blockquote is the first item in a list item.
		_, err = w.out.output.Write([]byte{'>', ' '})
		if err != nil {
			return err
		}
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeCodeBlock(node *content.CodeBlock) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw("```")
	if err != nil {
		return err
	}

	if node.Language != "" {
		err = w.writeRaw(node.Language)
		if err != nil {
			return err
		}
	}

	err = w.writeRaw("\n")
	if err != nil {
		return err
	}

	err = w.writeRaw(node.Code)
	if err != nil {
		return err
	}

	// If the code does not end with a blank line, we add a newline
	if !strings.HasSuffix(node.Code, "\n") {
		err = w.writeRaw("\n")
		if err != nil {
			return err
		}
	}

	err = w.writeRaw("```")
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeThematicBreak(node *content.ThematicBreak) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw("---")
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeBlock(node *content.Block) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	// Write the content first
	for _, child := range node.Content() {
		err := w.Write(child)
		if err != nil {
			return err
		}
	}

	w.endBlock()

	hasParentBlock := false
	if _, ok := node.Parent().(*content.Block); ok {
		hasParentBlock = true
	}

	previousIndent := ""
	if hasParentBlock && w.out.IndentationLevel() > 0 {
		// As Logseq uses tabs for indentation of blocks we pop the current
		// indentation which is the two spaces to align content with "- " of
		// the list item. This allows the indentation to be only tabs for
		// blocks
		previousIndent = w.out.PopIndentation()
	}

	// Output the sub blocks
	blocks := node.Blocks()
	if len(blocks) > 0 {
		if w.out.HasWrittenAtCurrentIndent() {
			err := w.out.WriteString("\n")
			if err != nil {
				return err
			}
		}

		if hasParentBlock {
			w.out.PushIndentation("\t")
		} else {
			w.out.PushIndentation("")
		}

		i := 0
		for _, child := range blocks {
			i++
			err := w.out.WriteString("- ")
			if err != nil {
				return err
			}

			w.out.PushIndentation("  ")

			err = w.Write(child)
			if err != nil {
				return err
			}

			if child.NextSibling() != nil {
				err = w.writeRaw("\n")
				if err != nil {
					return err
				}
			}

			w.out.PopIndentation()
		}

		w.out.PopIndentation()
	}

	if hasParentBlock {
		// Push the previous indentation back on the stack
		w.out.PushIndentation(previousIndent)
	}

	return nil
}

func (w *Output) writeProperties(node *content.Properties) error {
	err := w.startBlockWithAutomaticBehavior(node, "", false)
	if err != nil {
		return err
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*content.Property); !ok {
			return fmt.Errorf("unsupported properties child: %T", child)
		}

		property := child.(*content.Property)
		err := w.writeRaw(property.Name)
		if err != nil {
			return err
		}

		err = w.writeRaw(":: ")
		if err != nil {
			return err
		}

		err = w.writeChildren(property)
		if err != nil {
			return err
		}

		if child.NextSibling() != nil {
			err = w.writeRaw("\n")
			if err != nil {
				return err
			}
		}
	}

	w.endBlock()
	return nil
}

func (w *Output) writeAdvancedCommand(node *content.AdvancedCommand) error {
	return w.writeBeginEnd(node, node.Type, node.Value)
}

func (w *Output) writeBeginEnd(node content.BlockNode, variant string, value string) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw("#+BEGIN_" + variant + "\n")
	if err != nil {
		return err
	}

	err = w.writeRaw(value)
	if err != nil {
		return err
	}

	if !w.out.lastWasLineBreak {
		err = w.writeRaw("\n")
		if err != nil {
			return err
		}
	}

	err = w.writeRaw("#+END_" + variant)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeTaskMarker(node *content.TaskMarker) error {
	var err error
	switch node.Status {
	case content.TaskStatusNone:
		return nil
	case content.TaskStatusTodo:
		err = w.writeRaw("TODO")
	case content.TaskStatusDoing:
		err = w.writeRaw("DOING")
	case content.TaskStatusDone:
		err = w.writeRaw("DONE")
	case content.TaskStatusLater:
		err = w.writeRaw("LATER")
	case content.TaskStatusNow:
		err = w.writeRaw("NOW")
	case content.TaskStatusCancelled:
		err = w.writeRaw("CANCELLED")
	case content.TaskStatusCanceled:
		err = w.writeRaw("CANCELED")
	case content.TaskStatusInProgress:
		err = w.writeRaw("IN-PROGRESS")
	case content.TaskStatusWait:
		err = w.writeRaw("WAIT")
	case content.TaskStatusWaiting:
		err = w.writeRaw("WAITING")
	default:
		return fmt.Errorf("unsupported task status: %d", node.Status)
	}

	if err != nil {
		return err
	}

	if node.NextSibling() != nil {
		err = w.writeRaw(" ")
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Output) writeLogbook(node *content.Logbook) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw(":LOGBOOK:\n")
	if err != nil {
		return err
	}

	for _, entry := range node.Children() {
		switch e := entry.(type) {
		case *content.LogbookEntryRaw:
			err = w.writeRaw(e.Value)
			if err != nil {
				return err
			}

			if !strings.HasSuffix(e.Value, "\n") {
				err = w.writeRaw("\n")
				if err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unsupported logbook entry: %T", entry)
		}
	}

	err = w.writeRaw(":END:")
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}
