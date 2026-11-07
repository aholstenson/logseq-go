package markdown

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/textmerge"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var urlRegexp = regexp.MustCompile(`^(?:http|https|ftp)://[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-z]+(?::\d+)?(?:[/#?][-a-zA-Z0-9@:%_+.~#$!?&/=\(\);,'">\^{}\[\]` + "`" + `]*)?`)

// EscapeFunc escapes a string so that reading it back gives the same value.
type EscapeFunc func(string) string

func EscapeNone(str string) string {
	return str
}

// EscapePotentialMarkdown escapes characters in text that would otherwise be
// read back as formatting.
//
// Square brackets are only escaped where they would be read back as a link, a
// page link or a footnote. Logseq reads `\[` up
// to a later `\]` as a display math formula, so `[` is never escaped. Where a
// bracket must be escaped, the escape goes on the `]` or on the `^` of a
// footnote instead.
//
// The markers of emphasis, strikethrough and highlights are only escaped
// where they could open or close formatting, so that text such as
// `snake_case` or `2 * 3` is written as it is.
//
// The `#` of a tag, the `((` of a block reference, the `{{` of a macro, the
// backticks of code and the `<` of HTML are escaped where they would be read
// back as that syntax. A backslash is written as `\\` where it would otherwise
// escape the character after it, so `C:\Users` is written as it is.
func EscapePotentialMarkdown(str string) string {
	return escapeText(str, unknownRune, unknownRune)
}

// unknownRune stands for a character next to a text that is not known.
const unknownRune = rune(-1)

// lineEdgeRune stands for the start or the end of the line or of the block
// that a text is in, such as the end of a paragraph.
const lineEdgeRune = rune(-2)

// escapeText is EscapePotentialMarkdown for a text with known characters
// directly before and after it. Either can be unknownRune or lineEdgeRune.
func escapeText(str string, before rune, after rune) string {
	return escapeTextIn(str, "", before, after, false, true, nil, blockRefCanFollowRune(before))
}

// blockRefCanFollowRune returns true if a block reference can start directly
// after the character when it is read back, see canStartBlockRef.
func blockRefCanFollowRune(r rune) bool {
	return r == lineEdgeRune || (r >= 0 && strings.ContainsRune(blockRefPrefixes, r))
}

// escapeTextIn is escapeText for a text that can be inside formatting, such
// as emphasis. The text of formatting has no tags when it is read back, so a
// `#` in it is not escaped. If headings is false, the text is on lines that
// cannot start a heading when read back, so a `#` at the start of a line is
// not escaped for that. If outside is not nil, it tells if a marker can be
// in the nodes before or after the text, see markerOutside. refAtStart is
// whether a block reference can start at the start of the text when it is
// read back, see blockRefTextEscapes. If source is not empty, it is the text
// as it is in the file, and its backslashes are kept, see sourceEscapes.
func escapeTextIn(str string, source string, before rune, after rune, inFormatting bool, headings bool, outside markerOutside, refAtStart bool) string {
	escapes := map[int]bool{}
	sourceEscapes(str, source, escapes)
	mathEscapes(str, escapes)
	bracketEscapes(str, escapes)
	delimiterEscapes(str, before, after, outside, escapes)
	markupEscapes(str, before, after, !inFormatting, headings, escapes)
	backslashEscapes(str, after, escapes)
	blockRefTextEscapes(str, refAtStart, escapes)

	// The bytes of the text are copied, not the decoded runes, so that bytes
	// that are not valid UTF-8 are not changed.
	out := strings.Builder{}
	for i := 0; i < len(str); {
		_, size := utf8.DecodeRuneInString(str[i:])
		if escapes[i] {
			out.WriteByte('\\')
		}

		out.WriteString(str[i : i+size])
		i += size
	}

	return out.String()
}

// sourceEscapes finds the characters of the text that are escaped with a
// backslash in the source of the text, which is the text as it is in the
// file, see content.Text.TextSource. Escaping them again keeps the text of
// the file as it is. A backslash before ASCII punctuation always reads as
// that character, so this does not change how the text is read. Nothing is
// found if the source does not read as the text, which is the case when the
// text was changed after it was read.
func sourceEscapes(str string, source string, escapes map[int]bool) {
	if source == "" {
		return
	}

	found := map[int]bool{}
	value := make([]byte, 0, len(str))
	for i := 0; i < len(source); i++ {
		if source[i] == '\\' && i+1 < len(source) && util.IsPunct(source[i+1]) {
			found[len(value)] = true
			i++
		}

		value = append(value, source[i])
	}

	if string(value) != str {
		return
	}

	for i := range found {
		escapes[i] = true
	}
}

// asCharacter gets the character that a rune next to a text acts as for the
// rules of the reader. The edge of a line acts as a line break.
func asCharacter(r rune) rune {
	if r == lineEdgeRune {
		return '\n'
	}

	return r
}

// isASCIIPunctuation returns true for the characters that a backslash
// escapes, which are the ASCII punctuation characters.
func isASCIIPunctuation(r rune) bool {
	return r < utf8.RuneSelf && util.IsPunct(byte(r))
}

// backslashEscapes finds the backslashes in the text that are written as
// `\\`. A backslash before ASCII punctuation must be, as it would otherwise
// escape that character. A backslash before a space is also written as `\\`.
// `\ ` and `\\ ` read the same, and this keeps the text `a \\ b` as it is. A
// backslash at the end of the text is written as `\\` when a line break,
// ASCII punctuation or something not known comes after it, as a backslash
// before a line break is a hard line break.
func backslashEscapes(str string, after rune, escapes map[int]bool) {
	for i := 0; i < len(str); i++ {
		if str[i] != '\\' {
			continue
		}

		if i+1 < len(str) {
			next, _ := utf8.DecodeRuneInString(str[i+1:])
			escapes[i] = escapes[i] || isASCIIPunctuation(next) || unicode.IsSpace(next)
			continue
		}

		escapes[i] = escapes[i] || after == unknownRune || after == '\n' || isASCIIPunctuation(after)
	}
}

// markupEscapes finds the characters in the text that would start a tag, a
// heading, a block reference, a macro, code or HTML when read back. Tags are
// only looked for if tags is true, and headings only if headings is true.
func markupEscapes(str string, before rune, after rune, tags bool, headings bool, escapes map[int]bool) {
	if !strings.ContainsAny(str, "#({`<") {
		return
	}

	// Where the closing markers are last found. A marker that opens before
	// one of them can be closed. Finding them once keeps a long text with
	// many opening markers fast to write.
	lastMacroClose := strings.LastIndex(str, "}}")
	lastPageLinkClose := strings.LastIndex(str, "]]")
	lastAngleClose := strings.LastIndexByte(str, '>')

	// backticksEscaped is whether all backticks from a position on have been
	// escaped, which is at most once.
	backticksEscaped := false

	for i := 0; i < len(str); i++ {
		switch str[i] {
		case '#':
			previous := before
			if i > 0 {
				previous, _ = utf8.DecodeLastRuneInString(str[:i])
			}

			if (tags && startsTag(str, i, previous, lastPageLinkClose)) || (headings && startsHeading(str, i, previous, after)) {
				escapes[i] = true
			}
		case '(':
			if blockRefLength([]byte(str[i:])) > 0 && blockRefCanStartAt(str, i, escapes) {
				escapes[i] = true
			}
		case '{':
			if strings.HasPrefix(str[i:], "{{") && lastMacroClose >= i+2 && macroCanStartAt(str, i, escapes) {
				escapes[i] = true
			}
		case '`':
			end := i
			for end < len(str) && str[end] == '`' {
				end++
			}

			if !backticksEscaped && backtickRunAt(str, end, end-i) >= 0 {
				// A backslash does not escape a backtick inside code, so an
				// escaped backtick can still close code that a backtick
				// that is not escaped opens. All of them are escaped.
				for k := i; k < len(str); k++ {
					if str[k] == '`' {
						escapes[k] = true
					}
				}

				backticksEscaped = true
			}

			i = end - 1
		case '<':
			if lastAngleClose > i && startsHTMLOrAutoLink(str[i+1:]) {
				escapes[i] = true
			}
		}
	}
}

// blockRefTextEscapes removes the escapes from the text that is read back as
// a block reference to an id that is not a uuid, such as `((see [[x]]))`. The
// reader keeps such text as it is, see blockRefParser, so a backslash in it
// would be read back as a part of the text. Nothing in such text is read as
// markup, so it needs no escapes. refAtStart is whether such text can start
// at the start of the text, which is not the case directly after other text
// that does not end in one of blockRefPrefixes.
func blockRefTextEscapes(str string, refAtStart bool, escapes map[int]bool) {
	if !strings.Contains(str, "((") {
		return
	}

	for i := 0; i+1 < len(str); i++ {
		if str[i] != '(' || str[i+1] != '(' || escapes[i] || (i == 0 && !refAtStart) || !blockRefCanStartAt(str, i, escapes) {
			continue
		}

		length := blockRefTextLength([]byte(str[i:]))
		if length == 0 {
			continue
		}

		for k := i; k < i+length; k++ {
			delete(escapes, k)
		}

		i += length - 1
	}
}

// blockRefCanStartAt returns true if a block reference can start at the
// position in the text when it is read back, using the rules of
// canStartBlockRef. At the start of the text, the node before can be markup,
// after which a block reference can start. escapes are the characters that
// are already escaped, as an escaped character is its own element.
func blockRefCanStartAt(str string, i int, escapes map[int]bool) bool {
	return canStartAt(str, i, escapes, blockRefPrefixes)
}

// macroCanStartAt returns true if a macro can start at the position in the
// text when it is read back, using the rules of canStartMacro. See
// blockRefCanStartAt.
func macroCanStartAt(str string, i int, escapes map[int]bool) bool {
	return canStartAt(str, i, escapes, macroPrefixes)
}

// canStartAt returns true if an element that Logseq does not read inside a
// run of plain text can start at the position in the text when it is read
// back, see canStartAfterText. At the start of the text, the node before can
// be markup, after which the element can start.
func canStartAt(str string, i int, escapes map[int]bool, prefixes string) bool {
	if i == 0 || escapes[i-1] || strings.IndexByte(prefixes, str[i-1]) >= 0 {
		return true
	}

	start := 0
	if i > maxTimestampLength {
		start = i - maxTimestampLength
	}

	return endsWithTimestamp([]byte(str[start:i]))
}

// startsTag returns true if the `#` at the position would be read back as the
// start of a tag, using the rules of the tag parser. lastPageLinkClose is
// where the last `]]` of the text starts, or -1 if it has none.
func startsTag(str string, i int, previous rune, lastPageLinkClose int) bool {
	if previous != unknownRune && !unicode.IsSpace(asCharacter(previous)) &&
		!strings.ContainsRune(tagPrefixes, previous) {
		return false
	}

	if previous == '[' && i > 0 && isPriorityText(str[i-1:]) {
		// The priority parser reads `[#A]` and `[#word]` as a whole, so the
		// tag parser never sees the `#`.
		return false
	}

	rest := str[i+1:]
	if strings.HasPrefix(rest, "#") {
		return false
	}

	if (previous == lineEdgeRune || previous == '\n') && isDirective([]byte(str[i:])) {
		// A directive at the start of a line is not a tag, see directives.go.
		return false
	}

	if strings.HasPrefix(rest, "[[") {
		// `#[[]]` has no title, so it is not a tag, see pageLinkClose.
		return !strings.HasPrefix(rest, "[[]]") && lastPageLinkClose >= i+3
	}

	return tagNameLength([]byte(rest)) > 0
}

// isPriorityText returns true if the text starts with `[#` and a closing `]`
// that has no space or `[` before it, which the priority parser reads as a
// priority or claims as text.
func isPriorityText(str string) bool {
	if len(str) < 4 || !strings.HasPrefix(str, "[#") {
		return false
	}

	for i := 2; i < len(str); i++ {
		if str[i] == ']' {
			return i > 2
		}

		if str[i] == '[' || util.IsSpace(str[i]) {
			return false
		}
	}

	return false
}

// startsHeading returns true if the `#` at the position starts a line and
// would be read back as the marker of a heading. Logseq reads any number of
// `#` marks as a heading, see atxHeadingParser. after is the character after
// the text. If the marks end the text, they start a heading only if the line
// ends or a space comes after them, so the `#` of "##x" is not escaped when
// the tag "#x" follows it.
func startsHeading(str string, i int, previous rune, after rune) bool {
	if previous != unknownRune && previous != lineEdgeRune && previous != '\n' {
		return false
	}

	end := i
	for end < len(str) && str[end] == '#' {
		end++
	}

	if end == len(str) {
		return after == unknownRune || after == lineEdgeRune || after == '\n' || after == ' ' || after == '\t'
	}

	return str[end] == ' ' || str[end] == '\t'
}

// backtickRunAt finds the first run of exactly the given number of
// backticks from the position, which would close code that a run of the same
// length opens. Returns -1 if there is no such run.
func backtickRunAt(str string, from int, length int) int {
	for i := from; i < len(str); {
		if str[i] != '`' {
			i++
			continue
		}

		end := i
		for end < len(str) && str[end] == '`' {
			end++
		}

		if end-i == length {
			return i
		}

		i = end
	}

	return -1
}

// startsHTMLOrAutoLink returns true if the text after a `<` could make the
// `<` the start of an HTML tag or an autolink such as `<https://a.com>`.
//
// The caller checks that a `>` comes later in the text.
func startsHTMLOrAutoLink(rest string) bool {
	if rest == "" {
		return false
	}

	c := rest[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '/' || c == '!' || c == '?'
}

// delimiterEscapes finds the positions in the text that need a backslash so
// that the runs of `*`, `_`, `~`, `^` and `=` in it are read back as text.
// Every `*` and `_` of a run is escaped. For `~~`, `^^` and `==` the first
// marker is kept, as a single `~`, `^` or `=` is not a marker.
//
// A run is escaped if it can open formatting and something can close it, or
// if it can close formatting and something can open it. A run that only opens
// is kept as it is when no later run of the text can close it and nothing
// after the text in its paragraph can, such as the first `__` of `__a__b`.
// The same applies to a run that only closes, and what comes before it.
//
// Nothing can come after the text when it ends the paragraph, or when outside
// tells that the marker is not in the nodes after it. Outside can be nil.
//
// A marker that is in escapes already, such as one that is escaped in the
// file, see sourceEscapes, is text. It does not open or close formatting.
func delimiterEscapes(str string, before rune, after rune, outside markerOutside, escapes map[int]bool) {
	if !strings.ContainsAny(str, "*_~^=") {
		return
	}

	type run struct {
		start    int
		end      int
		min      int
		canOpen  bool
		canClose bool
	}

	runs := []run{}

	// Where the first run that can open and the last run that can close
	// start, for each marker character.
	firstOpener := map[byte]int{}
	lastCloser := map[byte]int{}

	for i := 0; i < len(str); {
		processor, min := delimiterProcessorFor(str[i])
		if processor == nil || escapes[i] {
			// A marker that is escaped already is read back as text.
			i++
			continue
		}

		end := i
		for end < len(str) && str[end] == str[i] && !escapes[end] {
			end++
		}

		if end-i >= min {
			canOpen, canClose := delimiterSides(str, i, end, min, processor, before, after)
			if canOpen || canClose {
				runs = append(runs, run{start: i, end: end, min: min, canOpen: canOpen, canClose: canClose})
			}

			if _, ok := firstOpener[str[i]]; canOpen && !ok {
				firstOpener[str[i]] = i
			}

			if canClose {
				lastCloser[str[i]] = i
			}
		}

		i = end
	}

	// canBeAfter and canBeBefore tell if the marker can be in what comes
	// after or before the text.
	canBeAfter := func(c byte) bool {
		return after != lineEdgeRune && (outside == nil || outside(c, true))
	}
	canBeBefore := func(c byte) bool {
		return before != lineEdgeRune && (outside == nil || outside(c, false))
	}

	for _, r := range runs {
		c := str[r.start]
		escape := false
		if r.canOpen {
			closer, ok := lastCloser[c]
			escape = (ok && closer > r.start) || canBeAfter(c)
		}

		if r.canClose && !escape {
			opener, ok := firstOpener[c]
			escape = (ok && opener < r.start) || canBeBefore(c)
		}

		if !escape {
			continue
		}

		for k := r.start; k < r.end; k++ {
			if r.min == 1 || k > r.start {
				escapes[k] = true
			}
		}
	}
}

// markerOutside tells if the marker character can be in the nodes after a
// text, if after is true, or in the nodes before it. It is used to find if a
// run of markers in the text can pair with a run outside it.
type markerOutside func(c byte, after bool) bool

// markerOutsideOf gets the markerOutside of a text in a paragraph. The marker
// can not be on a side if no node on that side has the marker character: in
// the value of a text, or in the Markdown that another node is written as.
// Returns nil for a text that is not in a paragraph.
func (w *Output) markerOutsideOf(node *content.Text) markerOutside {
	if _, ok := node.Parent().(*content.Paragraph); !ok {
		return nil
	}

	return func(c byte, after bool) bool {
		sibling := node.PreviousSibling()
		if after {
			sibling = node.NextSibling()
		}

		for sibling != nil {
			value := ""
			if text, ok := sibling.(*content.Text); ok {
				value = text.Value
			} else if value = w.writeIsolated(sibling); value == "" {
				// The node cannot be written, so what it holds is not known.
				return true
			}

			if strings.IndexByte(value, c) >= 0 {
				return true
			}

			if after {
				sibling = sibling.NextSibling()
			} else {
				sibling = sibling.PreviousSibling()
			}
		}

		return false
	}
}

// delimiterProcessorFor gets the processor and the shortest run of markers
// for the formatting that the character is a marker of.
func delimiterProcessorFor(c byte) (parser.DelimiterProcessor, int) {
	switch c {
	case '*', '_':
		return emphasisDelimiterProcessorFor(c), 1
	case '~':
		return defaultStrikethroughDelimiterProcessor, 2
	case '^':
		return caretHighlightDelimiterProcessor, 2
	case '=':
		return equalSignsHighlightDelimiterProcessor, 2
	}

	return nil, 0
}

// delimiterSides finds if the run of markers between start and end can open
// and if it can close formatting when it is read back. A run at the start or
// the end of the text uses the given character before or after the text. When
// that character is not known, each kind of character that changes the
// result is tried there.
func delimiterSides(str string, start int, end int, min int, processor parser.DelimiterProcessor, before rune, after rune) (canOpen bool, canClose bool) {
	befores := []rune{' ', 'a', '.'}
	if start > 0 {
		previous, _ := utf8.DecodeLastRuneInString(str[:start])
		befores = []rune{previous}
	} else if before != unknownRune {
		befores = []rune{asCharacter(before)}
	}

	afters := []string{""}
	if end == len(str) {
		if after == lineEdgeRune {
			afters = []string{""}
		} else if after != unknownRune {
			afters = []string{string(after)}
		} else {
			afters = []string{"", "a", "."}
		}
	}

	for _, before := range befores {
		for _, after := range afters {
			node := scanDelimiter([]byte(str[start:]+after), before, min, processor)
			if node != nil {
				canOpen = canOpen || node.CanOpen
				canClose = canClose || node.CanClose
			}
		}
	}

	return canOpen, canClose
}

// bracketEscapes finds the positions in the text that need a backslash so
// that the square brackets in it are read back as text.
func bracketEscapes(str string, escapes map[int]bool) {
	if !strings.ContainsAny(str, "[]") {
		return
	}

	ends := &pageLinkEnds{str: str, escapes: escapes}
	for i := 0; i < len(str); i++ {
		switch str[i] {
		case '[':
			if i+1 >= len(str) {
				continue
			}

			if str[i+1] == '[' && (i == 0 || str[i-1] != '!') && !strings.HasPrefix(str[i+2:], "]]") {
				// A page link. After `![` the reader starts an image, so
				// `![[Page]]` is text and is not changed. `[[]]` has no
				// title, so it is not a page link, see pageLinkClose.
				if end := ends.find(i + 2); end >= 0 {
					escapes[end+1] = true
				}
			} else if str[i+1] == '^' && footnoteLabelEnd([]byte(str[i:])) >= 0 {
				// A footnote reference or the start of a footnote definition.
				escapes[i+1] = true
			}
		case ']':
			if escapes[i] || i+1 >= len(str) {
				continue
			}

			// Logseq has no reference links or link reference definitions,
			// so only a `]` that ends the text of an inline link is escaped.
			if str[i+1] == '(' && closesLink(str[i+1:]) {
				escapes[i] = true
			}
		}
	}
}

// pageLinkEnds finds the `]]` that end page links in a text. It remembers the
// last `]]` and line break it found, so that a long line with many `[[` is
// read about once, and not once for each `[[`.
type pageLinkEnds struct {
	str     string
	escapes map[int]bool

	closers    nextMatch
	lineBreaks nextMatch
}

// find finds the `]]` that ends a page link that starts before the given
// position, in the same way as the page link parser. Returns the position of
// the first `]`, or -1 if the page link does not end on the same line.
func (e *pageLinkEnds) find(from int) int {
	search := func(from int) int {
		for j := from; j+1 < len(e.str); j++ {
			if e.closes(j) {
				return j
			}
		}

		return -1
	}

	end := e.closers.find(from, search)
	if end >= 0 && !e.closes(end) {
		// The `]]` has been escaped since it was found.
		e.closers = nextMatch{}
		end = e.closers.find(from, search)
	}

	if end < 0 {
		return -1
	}

	lineBreak := e.lineBreaks.find(from, func(from int) int {
		if i := strings.IndexByte(e.str[from:], '\n'); i >= 0 {
			return from + i
		}

		return -1
	})

	if lineBreak >= 0 && lineBreak < end {
		return -1
	}

	return end
}

// closes checks if a `]]` that is not escaped starts at the position.
func (e *pageLinkEnds) closes(j int) bool {
	return e.str[j] == ']' && e.str[j+1] == ']' && !e.escapes[j] && !e.escapes[j+1]
}

// closesLink returns true if a `]` that is directly followed by rest would end
// the text of a link. The reader decides this, so that a `]` before something
// that is not a valid destination, such as `(Page name)`, is not escaped.
//
// Only the part of rest that a link can take up is read, see linkCloseBound,
// so that a long text with many `](` is written in a time that grows with its
// length.
func closesLink(rest string) bool {
	end := linkCloseBound(rest)
	if end < 0 {
		return false
	}

	src := []byte("[x]" + rest[:end])

	options := defaultParseOptions()
	context := parser.NewContext()
	context.Set(parseOptionsKey, &options)
	doc := markdownParser.Parse(text.NewReader(src), parser.WithContext(context))

	found := false
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if link, ok := n.(*ast.Link); ok {
			label, isText := link.FirstChild().(*ast.Text)
			found = isText && label.Segment.Start == 1
			return ast.WalkStop, nil
		}

		return ast.WalkContinue, nil
	})

	return found
}

// linkCloseBound finds where the destination and title of a link that starts
// with the `(` at the start of rest would end, after the closing `)`. It
// returns -1 if no link can start there. The rules are looser than those of
// the reader, so that the reader can decide from the part of rest before the
// end.
func linkCloseBound(rest string) int {
	if !strings.HasPrefix(rest, "(") {
		return -1
	}

	if length := pageRefLength([]byte(rest[1:])); length > 0 && 1+length < len(rest) && rest[1+length] == ')' {
		// A label for a page, as in `[label]([[Page]])`, see
		// labelledRefLinkParser.
		return 1 + length + 1
	}

	i := skipLinkSpace(rest, 1)
	if i < len(rest) && rest[i] == ')' {
		return i + 1
	}

	// The destination.
	if i < len(rest) && rest[i] == '<' {
		for i++; i < len(rest) && rest[i] != '>'; i++ {
			if rest[i] == '\\' && i+1 < len(rest) && util.IsPunct(rest[i+1]) {
				i++
			}
		}

		if i >= len(rest) {
			return -1
		}

		i++
	} else {
		depth := 0
	destination:
		for ; i < len(rest); i++ {
			switch c := rest[i]; {
			case c == '\\' && i+1 < len(rest) && util.IsPunct(rest[i+1]):
				// A backslash escapes only ASCII punctuation.
				i++
			case c == '(':
				depth++
			case c == ')':
				if depth == 0 {
					return i + 1
				}

				depth--
			case c <= ' ':
				break destination
			}
		}
	}

	// The title, if there is one, and then the closing `)`.
	i = skipLinkSpace(rest, i)
	if i >= len(rest) {
		return -1
	}

	switch rest[i] {
	case ')':
		return i + 1
	case '"', '\'', '(':
		closer := rest[i]
		if closer == '(' {
			closer = ')'
		}

		for i++; i < len(rest) && rest[i] != closer; i++ {
			if rest[i] == '\\' && i+1 < len(rest) && util.IsPunct(rest[i+1]) {
				i++
			}
		}

		i = skipLinkSpace(rest, i+1)
		if i < len(rest) && rest[i] == ')' {
			return i + 1
		}
	}

	return -1
}

// skipLinkSpace skips the spaces, tabs and line breaks from the position.
func skipLinkSpace(str string, i int) int {
	for i < len(str) && (str[i] == ' ' || str[i] == '\t' || str[i] == '\n' || str[i] == '\r') {
		i++
	}

	return i
}

// mathEscapes finds the dollar signs of the parts of a text that would be
// read back as a LaTeX formula, so that text such as `$x$` stays text. The
// dollar signs within a formula are escaped as well, as leaving one of them
// alone would let it pair up with something later in the text.
func mathEscapes(str string, escapes map[int]bool) {
	if !strings.Contains(str, "$") {
		return
	}

	src := []byte(str)
	scanner := newMathScanner(src)
	for i := 0; i < len(src); {
		if src[i] != '$' {
			i++
			continue
		}

		_, length, _, ok := scanner.at(i, scanner.lineEnd(i))
		if !ok {
			i++
			continue
		}

		for k := i; k < i+length; k++ {
			if src[k] == '$' {
				escapes[k] = true
			}
		}

		i += length
	}
}

func EscapeLinkURL(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == '(' || r == ')'
	})
}

func EscapeLinkTitle(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == '"' || r == '\'' || r == ')'
	})
}

// EscapeRefTitle escapes the title of a page link or block reference with a
// label, which is written in double quotes. Logseq keeps a backslash in the
// title, so only the quote that would end the title is escaped.
func EscapeRefTitle(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == '"'
	})
}

func EscapeWikiLink(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == ']'
	})
}

// EscapePageLinkTitle escapes the title of a page link that is written as
// `[[title]]`. A `]` that is followed by another `]`, or that is the last
// character, could end the link, so it is escaped. Other brackets are kept, so
// that `[[file:./a.md][A]]` is written as it is.
//
// Page links can be nested, so the `]]` of a page link in the title, such as
// in `a [[b]]`, is kept when every `[[` in the title has its `]]`. The reader
// then counts the brackets and ends the link at the right `]]`.
func EscapePageLinkTitle(str string) string {
	nested := nestedLinkCloses(str)

	out := strings.Builder{}
	for i := 0; i < len(str); i++ {
		if nested[i] {
			// The two characters of a `]]` that closes a nested link.
			out.WriteString(str[i : i+2])
			i++
			continue
		}

		if str[i] == ']' && (i+1 == len(str) || str[i+1] == ']') {
			out.WriteByte('\\')
		}

		out.WriteByte(str[i])
	}

	return out.String()
}

// pageLinkTargetReadsBack checks if the target of a page link as it is in the
// file, see content.PageLink.ToSource, is read back as the target. This is not
// the case when the target was changed after it was read, as when a page is
// renamed.
func pageLinkTargetReadsBack(source string, to string) bool {
	if strings.ContainsAny(source, "\r\n") || unescapePunctuation([]byte(source)) != to {
		return false
	}

	link := "[[" + source + "]]"
	return pageLinkClose([]byte(link)) == len(link)-2
}

// nestedLinkCloses finds the positions of the `]]` in a page title that close
// a `[[` in the same title. Returns nil if a `[[` in the title is not closed,
// as the reader then ends the link at the first `]]`.
func nestedLinkCloses(str string) map[int]bool {
	if !strings.Contains(str, "[[") {
		return nil
	}

	closes := map[int]bool{}
	open := 0
	for i := 0; i+1 < len(str); i++ {
		if str[i] == '[' && str[i+1] == '[' {
			open++
			i++
		} else if str[i] == ']' && str[i+1] == ']' && open > 0 {
			closes[i] = true
			open--
			i++
		}
	}

	if open > 0 {
		return nil
	}

	return closes
}

// EscapeFootnoteLabel escapes the label of a footnote, which is read up to the
// first closing bracket in it.
func EscapeFootnoteLabel(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == ']' || r == '['
	})
}

func EscapeBlockRef(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == ')'
	})
}

func EscapeMacroQuotedArgument(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == '"'
	})
}

// escapeRunes puts a backslash in front of every rune the given function picks
// out, passing it the rune before it for the cases that need that context.
func escapeRunes(str string, f func(prev rune, r rune) bool) string {
	// The bytes of the text are copied, not the decoded runes, so that bytes
	// that are not valid UTF-8 are not changed.
	out := strings.Builder{}
	p := rune(0)
	for i := 0; i < len(str); {
		r, size := utf8.DecodeRuneInString(str[i:])
		if f(p, r) {
			out.WriteByte('\\')
		}

		out.WriteString(str[i : i+size])
		p = r
		i += size
	}

	return out.String()
}

// Option changes how Markdown is written for the parts of the syntax that a
// graph configures the shape of.
type Option func(*outputOptions)

// outputOptions are the settings that the writer takes from the graph. The
// zero value is not usable, use defaultOutputOptions to get the defaults of
// Logseq.
type outputOptions struct {
	// logbookWithSeconds is whether the times in logbook entries are written
	// with seconds.
	logbookWithSeconds bool
}

// defaultOutputOptions are what Logseq does for a graph that does not
// configure anything else.
func defaultOutputOptions() outputOptions {
	return outputOptions{
		logbookWithSeconds: true,
	}
}

// WithLogbookSeconds sets whether the times in logbook entries are written
// with seconds, which is `:with-second-support?` of `:logbook/settings`.
func WithLogbookSeconds(withSeconds bool) Option {
	return func(o *outputOptions) {
		o.logbookWithSeconds = withSeconds
	}
}

// Output is used to write Markdown to an output buffer. It will help keep
// track of list indentation and when to add newlines.
type Output struct {
	out  *writer
	opts outputOptions

	// isolated is set when a node is written on its own, to find the
	// characters around another node. Text is then escaped without looking
	// at the nodes around it, which also keeps two text nodes from writing
	// each other without end.
	isolated bool

	// quoteLevels are the indentation levels of the `>` markers of the quotes
	// that are being written, from the outermost to the innermost.
	quoteLevels []int

	// footnoteLevels are the indentation levels of the lines of the footnote
	// definitions that are being written, from the outermost to the
	// innermost.
	footnoteLevels []int

	// blockTextLevels are the indentation levels of the text of the blocks
	// that are being written, after their bullets, from the outermost to the
	// innermost.
	blockTextLevels []int
}

// NewWriter creates a new Markdown writer.
func NewWriter(out io.Writer, opts ...Option) *Output {
	options := defaultOutputOptions()
	for _, opt := range opts {
		opt(&options)
	}

	return &Output{
		out:  newWriter(out),
		opts: options,
	}
}

func AsString(n content.Node, opts ...Option) (string, error) {
	out := strings.Builder{}
	w := NewWriter(&out, opts...)
	if err := w.Write(n); err != nil {
		return "", err
	}

	return out.String(), nil
}

func Write(n content.Node, out io.Writer, opts ...Option) error {
	w := NewWriter(out, opts...)
	return w.Write(n)
}

func (w *Output) Write(n content.Node) error {
	if block, ok := n.(*content.Block); ok && block.LineEnding() != "" && w.out.lineEnding == "" && !w.isolated {
		// The block was read from a text with other line breaks, so all of
		// its lines are written with them.
		w.out.lineEnding = block.LineEnding()
		defer func() {
			w.out.lineEnding = ""
		}()
	}

	if aware, ok := n.(content.MacroSourceAware); ok {
		if source := aware.MacroSource(); source != "" && readsBackAsMacro(source, n) {
			// The macro is not changed, so its text is written as it is.
			return w.writeRaw(source)
		}
	}

	switch node := n.(type) {
	case *content.RawHTML:
		return w.writeRaw(node.HTML)
	case *content.Text:
		return w.writeText(node)
	case *content.RawText:
		// Raw text is already Markdown, so it is written as it is.
		return w.writeRaw(node.Value)
	case *content.Emphasis:
		return w.writeEmphasis(node)
	case *content.Strong:
		return w.writeStrong(node)
	case *content.Strikethrough:
		return w.writeStrikethrough(node)
	case *content.Highlight:
		return w.writeHighlight(node)
	case *content.FootnoteRef:
		return w.writeFootnoteRef(node)
	case *content.FootnoteDefinition:
		return w.writeFootnoteDefinition(node)
	case *content.Math:
		return w.writeMath(node)
	case *content.MathBlock:
		return w.writeMathBlock(node)
	case *content.CodeSpan:
		return w.writeCodeSpan(node)
	case *content.Link:
		return w.writeLink(node)
	case *content.AutoLink:
		return w.writeAutoLink(node)
	case *content.PageLink:
		return w.writePageLink(node)
	case *content.PageRefText:
		return w.writePageRefText(node)
	case *content.Hashtag:
		return w.writeHashtag(node)
	case *content.BlockRef:
		return w.writeBlockRef(node)
	case *content.Image:
		return w.writeImage(node)
	case *content.Macro:
		return w.writeMacro(node, node.Name, node.Arguments)
	case *content.Query:
		return w.writeMacroText("query", node.Query)
	case *content.PageEmbed:
		return w.writeMacroText("embed", "[["+EscapePageLinkTitle(node.To)+"]]")
	case *content.BlockEmbed:
		return w.writeMacroText("embed", "(("+EscapeBlockRef(node.ID)+"))")
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
	case *content.Table:
		return w.writeTable(node)
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
	case *content.FrontMatter:
		return w.writeFrontMatter(node)
	case *content.AdvancedCommand:
		return w.writeAdvancedCommand(node)
	case *content.QueryCommand:
		variant := "QUERY"
		if node.LowerCase {
			variant = "query"
		}
		return w.writeBeginEnd(node, beginEndLines{variant, node.Parameters, node.LowerCase}, node.Query)
	case *content.TaskMarker:
		return w.writeTaskMarker(node)
	case *content.TaskPriority:
		return w.writeTaskPriority(node)
	case *content.TaskDate:
		return w.writeTaskDate(node)
	case *content.Logbook:
		return w.writeLogbook(node)
	case *content.Drawer:
		return w.writeDrawer(node)
	default:
		return fmt.Errorf("unsupported node: %T", node)
	}
}

func (w *Output) writeRaw(s string) error {
	return w.out.WriteString(s)
}

func (w *Output) write(s string, escapeFunc EscapeFunc) error {
	return w.writeRaw(escapeFunc(s))
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

// writePageRefText writes the title of a page that a property value names
// without markup. The title is written the same way text is.
func (w *Output) writePageRefText(node *content.PageRefText) error {
	before, after := unknownRune, unknownRune
	if !w.isolated {
		before, after = w.textSurroundings(node, false)
	}

	if r := after; r != unknownRune && r != lineEdgeRune && isValueSeparator(r) {
		// The value is split on the separator before the titles are read, so
		// the separator acts as the end of the title.
		after = lineEdgeRune
	}

	return w.writeRaw(escapeText(node.To, before, after))
}

func (w *Output) writeText(node *content.Text) error {
	before, after := unknownRune, unknownRune
	var outside markerOutside
	if !w.isolated {
		before, after = w.textSurroundings(node, node.SoftLineBreak || node.HardLineBreak)
		outside = w.markerOutsideOf(node)
	}

	// A block reference can start at the start of the text, unless text that
	// it would continue comes before it.
	refAtStart := true
	if prev, ok := node.PreviousSibling().(*content.Text); ok && !prev.SoftLineBreak && !prev.HardLineBreak {
		refAtStart = blockRefCanFollowRune(before)
	}

	value := escapeTextIn(node.Value, node.TextSource(), before, after, isInFormatting(node), !isInIndentedParagraph(node), outside, refAtStart)
	if strings.HasSuffix(value, "]") && !endsWithEscape([]byte(value)) && (after == '(' || after == '[') {
		// The next node starts with `(` or `[`, which together with the
		// closing bracket would be read back as a link.
		value = value[:len(value)-1] + `\]`
	}

	if ref, ok := node.NextSibling().(*content.BlockRef); ok && !ref.HasLabel() &&
		!node.SoftLineBreak && !node.HardLineBreak && value != "" && !blockRefCanFollow([]byte(value)) && isASCIIPunctuation(rune(value[len(value)-1])) {
		// A block reference directly after text is read back as text, but
		// it is read after an escaped character, as in `\)((id))`.
		value = value[:len(value)-1] + `\` + value[len(value)-1:]
	}

	if _, ok := node.NextSibling().(content.MacroSourceAware); ok &&
		!node.SoftLineBreak && !node.HardLineBreak && value != "" && !macroCanFollow([]byte(value)) && isASCIIPunctuation(rune(value[len(value)-1])) {
		// The same goes for a macro, as in `\){{m}}`.
		value = value[:len(value)-1] + `\` + value[len(value)-1:]
	}

	if image, ok := node.PreviousSibling().(*content.Image); ok && image.Metadata == "" &&
		imageMetadataLength([]byte(value)) > 0 {
		// The `{` directly after an image without metadata would be read
		// back as the metadata of the image.
		value = `\` + value
	}

	err := w.writeRaw(value)
	if err != nil {
		return err
	}

	if node.SoftLineBreak {
		lineEnd := "\n"
		if !strings.HasSuffix(node.TrailingSpace, "  ") {
			// Two spaces at the end of the line would be read back as a
			// hard line break, so only fewer spaces are kept.
			lineEnd = node.TrailingSpace + lineEnd
		}

		err = w.writeRaw(lineEnd)
		if err != nil {
			return err
		}
	} else if node.HardLineBreak {
		// Logseq shows a backslash at the end of a line as text, so a break
		// that was written with spaces is written with the same spaces.
		lineEnd := "\\\n"
		if node.TrailingSpace != "" {
			lineEnd = node.TrailingSpace + "\n"
		}

		err = w.writeRaw(lineEnd)
		if err != nil {
			return err
		}
	}

	if node.LazyLine && (node.SoftLineBreak || node.HardLineBreak) {
		// The next line is a part of the quote or the footnote, but was
		// written without the `>` markers of the quotes or the indentation of
		// the footnote. The outermost of them is left out.
		level := -1
		for _, levels := range [][]int{w.quoteLevels, w.footnoteLevels} {
			if len(levels) > 0 && (level < 0 || levels[0] < level) {
				level = levels[0]
			}
		}

		if level >= 0 {
			w.out.SkipIndentationOnNextLine(level)
		}
	} else if node.LazyBlockLine && (node.SoftLineBreak || node.HardLineBreak) && len(w.blockTextLevels) > 0 {
		// The next line is a part of the block, but is indented less than
		// the text of the block.
		w.out.SkipIndentationOnNextLine(w.blockTextLevels[len(w.blockTextLevels)-1])
	}

	if (node.SoftLineBreak || node.HardLineBreak) && node.NextLineIndentation != "" {
		err = w.writeRaw(node.NextLineIndentation)
		if err != nil {
			return err
		}
	}

	return nil
}

// isInIndentedParagraph returns true if the node is in a paragraph that is
// indented four or more columns. The lines of such a paragraph are read back
// as text, so they cannot start a heading.
func isInIndentedParagraph(node content.Node) bool {
	for p := node.Parent(); p != nil; p = p.Parent() {
		if paragraph, ok := p.(*content.Paragraph); ok {
			width, _ := util.IndentWidth([]byte(paragraph.Indentation), 0)
			return width >= 4
		}
	}

	return false
}

// isInFormatting returns true if the node is inside emphasis, strong text,
// strikethrough or a highlight.
func isInFormatting(node content.Node) bool {
	for p := node.Parent(); p != nil; p = p.Parent() {
		switch p.(type) {
		case *content.Emphasis, *content.Strong, *content.Strikethrough, *content.Highlight:
			return true
		}
	}

	return false
}

// textSurroundings finds the characters that are written directly before
// and after a text node, as these decide if the characters in the text can be
// read back as markup. Returns unknownRune for a side that is not known.
func (w *Output) textSurroundings(node content.Node, lineBreak bool) (rune, rune) {
	before, after := edgeRunesOf(node.Parent())

	if prev := node.PreviousSibling(); prev != nil {
		before = unknownRune
		if text, ok := prev.(*content.Text); ok && (text.SoftLineBreak || text.HardLineBreak) {
			before = '\n'
		} else if written := w.writeIsolated(prev); written != "" {
			before, _ = utf8.DecodeLastRuneInString(written)
		}
	}

	if lineBreak {
		after = '\n'
	} else if next := node.NextSibling(); next != nil {
		after = unknownRune
		if written := w.writeIsolated(next); written != "" {
			after, _ = utf8.DecodeRuneInString(written)
		}
	}

	return before, after
}

// edgeRunesOf gets characters that are like the ones written directly before
// the first child and after the last child of the node. Returns unknownRune
// if the node is not one of the known kinds.
func edgeRunesOf(node content.HasChildren) (rune, rune) {
	switch node.(type) {
	case *content.Paragraph:
		return lineEdgeRune, lineEdgeRune
	case *content.Heading, *content.Property:
		// After the `# ` of a heading or the `:: ` of a property, and at the
		// end of the line.
		return ' ', lineEdgeRune
	case *content.Emphasis, *content.Strong, *content.Strikethrough, *content.Highlight,
		*content.Link, *content.PageLink, *content.BlockRef:
		// The markers around formatting and the brackets around a label.
		return '.', '.'
	}

	return unknownRune, unknownRune
}

// writeIsolated writes a node on its own, to find the characters at its
// start and end. Returns an empty string if the node cannot be written.
func (w *Output) writeIsolated(node content.Node) string {
	out := strings.Builder{}
	isolated := &Output{out: newWriter(&out), opts: w.opts, isolated: true}
	if err := isolated.Write(node); err != nil {
		return ""
	}

	return out.String()
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

	marker := string(emphasisMarker(node, node.Underscore))
	err := w.writeRaw(marker)
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw(marker)
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

	marker := strings.Repeat(string(emphasisMarker(node, node.Underscore)), 2)
	err := w.writeRaw(marker)
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw(marker)
	if err != nil {
		return err
	}

	return nil
}

// emphasisMarker gets the marker that emphasis or strong text is written
// with. If underscore is true, `_` is used where it is read back as the same
// formatting: a `_` that opens must not come directly after a letter or a
// digit, and a `_` that closes must not come directly before one. Elsewhere
// `*` is used.
//
// Only the text next to the node is looked at. The other nodes that can be
// next to it, such as links and code, start and end with punctuation.
func emphasisMarker(node content.Node, underscore bool) byte {
	if !underscore {
		return '*'
	}

	if text, ok := node.PreviousSibling().(*content.Text); ok && !text.SoftLineBreak && !text.HardLineBreak && text.Value != "" {
		r, _ := utf8.DecodeLastRuneInString(text.Value)
		if !util.IsSpaceRune(r) && !util.IsPunctRune(r) {
			return '*'
		}
	}

	if text, ok := node.NextSibling().(*content.Text); ok && text.Value != "" {
		r, _ := utf8.DecodeRuneInString(text.Value)
		if !util.IsSpaceRune(r) && !util.IsPunctRune(r) {
			return '*'
		}
	}

	return '_'
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

func (w *Output) writeHighlight(node *content.Highlight) error {
	marker := "^^"
	if node.EqualSigns {
		marker = "=="
	}

	err := w.writeRaw(marker)
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw(marker)
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeFootnoteRef(node *content.FootnoteRef) error {
	return w.writeRaw("[^" + EscapeFootnoteLabel(node.Label) + "]")
}

func (w *Output) writeFootnoteDefinition(node *content.FootnoteDefinition) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	// The spaces after the label are kept as they are in the file.
	space := node.LabelSpace
	if space == "" && node.FirstChild() != nil {
		space = " "
	}

	err = w.writeRaw("[^" + EscapeFootnoteLabel(node.Label) + "]:" + space)
	if err != nil {
		return err
	}

	// The content of a footnote carries on over the lines that are indented
	// under the definition.
	w.out.PushIndentation("    ")
	w.footnoteLevels = append(w.footnoteLevels, w.out.IndentationLevel()-1)

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	w.footnoteLevels = w.footnoteLevels[:len(w.footnoteLevels)-1]
	w.out.PopIndentation()

	// A lazy line is only a line of the footnote, so no line after the
	// footnote leaves out indentation.
	w.out.SkipIndentationOnNextLine(-1)

	w.endBlock()
	return nil
}

func (w *Output) writeMath(node *content.Math) error {
	opening, closing := "$", "$"
	switch {
	case node.Backslashes && node.Displayed:
		opening, closing = `\[`, `\]`
	case node.Backslashes:
		opening, closing = `\(`, `\)`
	case node.Displayed:
		opening, closing = "$$", "$$"
	}

	err := w.writeRaw(opening)
	if err != nil {
		return err
	}

	err = w.writeRaw(node.Value)
	if err != nil {
		return err
	}

	return w.writeRaw(closing)
}

func (w *Output) writeMathBlock(node *content.MathBlock) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	opening, closing := "$$", "$$"
	if node.Environment != "" {
		opening = `\begin{` + node.Environment + `}`
		closing = `\end{` + node.Environment + `}`
	}

	err = w.writeRaw(opening + "\n")
	if err != nil {
		return err
	}

	err = w.writeRaw(node.Value)
	if err != nil {
		return err
	}

	// The line that ends the block needs a line of its own.
	if !strings.HasSuffix(node.Value, "\n") {
		err = w.writeRaw("\n")
		if err != nil {
			return err
		}
	}

	err = w.writeRaw(closing)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeCodeSpan(node *content.CodeSpan) error {
	// A code span read from a file is written with the backticks and spaces
	// it had, if they still give the same code.
	if source := node.CodeSpanSource(); source != "" && codeSpanReadsBack(source, node.Value) {
		return w.writeRaw(source)
	}

	// The marker has to be longer than the longest run of backticks in the
	// value, or it would end the code span early.
	longestRun := 0
	currentRun := 0
	for i := 0; i < len(node.Value); i++ {
		if node.Value[i] == '`' {
			currentRun++
			if currentRun > longestRun {
				longestRun = currentRun
			}
		} else {
			currentRun = 0
		}
	}
	marker := strings.Repeat("`", longestRun+1)

	// A code span that starts or ends with a backtick needs to be padded with
	// spaces to keep the backtick apart from the marker. The same padding is
	// needed for a value surrounded by spaces, as a reader strips one space
	// from each end when both are present.
	value := node.Value
	if needsCodeSpanPadding(value) {
		value = " " + value + " "
	}

	err := w.writeRaw(marker)
	if err != nil {
		return err
	}

	err = w.writeRaw(value)
	if err != nil {
		return err
	}

	err = w.writeRaw(marker)
	if err != nil {
		return err
	}
	return nil
}

// needsCodeSpanPadding checks if a code span value has to be written with a
// space at each end to survive being read back.
func needsCodeSpanPadding(value string) bool {
	if strings.HasPrefix(value, "`") || strings.HasSuffix(value, "`") {
		return true
	}

	// A value of only spaces is kept as is by a reader, and padding it would
	// make it longer instead.
	if strings.Trim(value, " ") == "" {
		return false
	}

	return strings.HasPrefix(value, " ") && strings.HasSuffix(value, " ")
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

	err = w.writeDestination(node, node.URL, node.Title)
	if err != nil {
		return err
	}

	err = w.writeRaw(")")
	if err != nil {
		return err
	}

	return nil
}

// writeDestination writes the text between the parentheses of a link or an
// image: the URL and the title. If the node was read from a file and its text
// reads back as the same URL and title, the text is written as it is.
func (w *Output) writeDestination(node content.DestinationSourceAware, url string, title string) error {
	if source := node.DestinationSource(); source != "" && destinationReadsBack(source, url, title) {
		return w.writeRaw(source)
	}

	var err error
	if strings.ContainsFunc(url, unicode.IsSpace) {
		// A URL with spaces is only read as one destination in angle
		// brackets.
		err = w.writeRaw("<" + escapeRunes(url, func(prev rune, r rune) bool {
			return r == '<' || r == '>'
		}) + ">")
	} else {
		err = w.write(url, EscapeLinkURL)
	}
	if err != nil {
		return err
	}

	if title != "" {
		err = w.writeRaw(" '")
		if err != nil {
			return err
		}

		err = w.write(title, EscapeLinkTitle)
		if err != nil {
			return err
		}

		err = w.writeRaw("'")
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Output) writeAutoLink(node *content.AutoLink) error {
	if !node.Bracketed && urlRegexp.Match([]byte(node.URL)) {
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

// writePageLink writes *content.PageLink as `[[to]]`, or as `[label]([[to]])`
// if the link has a label.
func (w *Output) writePageLink(node *content.PageLink) error {
	if node.HasLabel() {
		err := w.writeLabelStart(node)
		if err != nil {
			return err
		}
	}

	err := w.writeRaw("[[")
	if err != nil {
		return err
	}

	if source := node.ToSource(); source != "" && pageLinkTargetReadsBack(source, node.To) {
		err = w.writeRaw(source)
	} else {
		err = w.write(node.To, EscapePageLinkTitle)
	}
	if err != nil {
		return err
	}

	err = w.writeRaw("]]")
	if err != nil {
		return err
	}

	if node.HasLabel() {
		return w.writeRefTitleEnd(node, node.Title)
	}

	return nil
}

// writeLabelStart writes the label of a page link or block reference as
// `[label](`. The target and the closing `)` are written after it.
func (w *Output) writeLabelStart(node content.HasChildren) error {
	err := w.writeRaw("[")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	return w.writeRaw("](")
}

// writeHashtag writes *content.Hashtag as `#to` or `#[[to]]`. The form with
// brackets is used if the tag has it, or if `#to` would not be read back as a
// tag with the same title.
func (w *Output) writeHashtag(node *content.Hashtag) error {
	if !node.Bracketed && isBareTagName(node.To) {
		return w.writeRaw("#" + node.To)
	}

	err := w.writeRaw("#[[")
	if err != nil {
		return err
	}

	err = w.write(node.To, EscapePageLinkTitle)
	if err != nil {
		return err
	}

	return w.writeRaw("]]")
}

// isBareTagName returns true if `#name` is read back as a tag with the whole
// name. A space or a tag delimiter such as `,` ends the name early, and a
// bracket can start a link, so such names need the form `#[[name]]`.
func isBareTagName(name string) bool {
	return name != "" &&
		tagNameLength([]byte(name)) == len(name) &&
		!strings.ContainsAny(name, "[]")
}

// writeBlockRef writes *content.BlockRef as `((id))`, or as `[label](((id)))`
// if the reference has a label.
func (w *Output) writeBlockRef(node *content.BlockRef) error {
	if node.HasLabel() {
		err := w.writeLabelStart(node)
		if err != nil {
			return err
		}
	}

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

	if node.HasLabel() {
		return w.writeRefTitleEnd(node, node.Title)
	}

	return nil
}

// writeRefTitleEnd writes the title of a page link or block reference with a
// label, and the `)` that ends it. If the node was read from a file and the
// text after its `[[Page]]` or `((id))` reads back as the same title, the
// text is written as it is. Otherwise a title is written in double quotes,
// which is the only form that Logseq reads, see refLinkTitle.
func (w *Output) writeRefTitleEnd(node content.DestinationSourceAware, title string) error {
	if source := node.DestinationSource(); source != "" {
		rest := refTitleSource(source)
		if read, ok := refLinkTitle(rest); ok && read == title && !strings.ContainsAny(rest, "\r\n") {
			return w.writeRaw(rest + ")")
		}
	}

	if title != "" {
		err := w.writeRaw(" \"")
		if err != nil {
			return err
		}

		err = w.write(title, EscapeRefTitle)
		if err != nil {
			return err
		}

		err = w.writeRaw("\"")
		if err != nil {
			return err
		}
	}

	return w.writeRaw(")")
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

	err = w.writeDestination(node, node.URL, node.Title)
	if err != nil {
		return err
	}

	err = w.writeRaw(")")
	if err != nil {
		return err
	}

	if node.Metadata != "" {
		return w.writeRaw(node.Metadata)
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

// writeMacroText writes a macro with the text after the name as it is.
// Logseq joins the arguments of the `query` and `embed` macros again, so a
// comma in the text needs no quotes.
func (w *Output) writeMacroText(name string, text string) error {
	return w.writeRaw("{{" + name + " " + text + "}}")
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

			if isQuotedMacroArgument(arg) {
				// The argument has its quotes, as it was read from a file.
				err = w.writeRaw(arg)
				if err != nil {
					return err
				}

				continue
			}

			// An argument with a comma needs quotes, so that it is not split.
			if strings.ContainsRune(arg, ',') {
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

func (w *Output) writeDrawer(node *content.Drawer) error {
	// As for properties, a drawer made with the API follows the line before
	// it without a blank line.
	err := w.startBlockWithAutomaticBehavior(node, "", false)
	if err != nil {
		return err
	}

	err = w.writeRaw(":" + node.Name + ":\n" + node.Content)
	if err != nil {
		return err
	}

	if node.Content != "" && !strings.HasSuffix(node.Content, "\n") {
		err = w.writeRaw("\n")
		if err != nil {
			return err
		}
	}

	endLine := ":END:"
	if node.EndLine != "" && isDrawerEnd([]byte(node.EndLine)) && !strings.ContainsAny(node.EndLine, "\r\n") {
		endLine = node.EndLine
	}

	err = w.writeRaw(endLine)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeRawHTMLBlock(node *content.RawHTMLBlock) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	// The HTML of a block read from a file ends with the line break of its
	// last line. The line break before the next node is written by that node.
	err = w.writeRaw(strings.TrimSuffix(node.HTML, "\n"))
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeHeading(node *content.Heading) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	// The marks are written as text and not as indentation, as indentation
	// is only written after a line break. A heading block starts on the line
	// of its bullet, as in `- ## Section`.
	marks := node.Indentation + strings.Repeat("#", node.Level)
	if node.MarksSpace != "" {
		marks += node.MarksSpace
	} else if node.FirstChild() != nil {
		marks += " "
	}

	err = w.writeRaw(marks)
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	if node.FirstChild() != nil && node.TrailingSpace != "" {
		err = w.writeRaw(node.TrailingSpace)
		if err != nil {
			return err
		}
	}

	w.endBlock()
	return nil
}

func (w *Output) writeParagraph(node *content.Paragraph) error {
	doubleNewLine := true
	if _, previousProperties := node.PreviousSibling().(*content.Properties); previousProperties {
		doubleNewLine = false
	}

	// The indentation is written before each line of the paragraph.
	err := w.startBlockWithAutomaticBehavior(node, node.Indentation, doubleNewLine)
	if err != nil {
		return err
	}

	if node.FirstLineIndentation != "" && node.FirstChild() != nil {
		err = w.writeRaw(node.FirstLineIndentation)
		if err != nil {
			return err
		}
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	if node.TrailingSpace != "" && node.FirstChild() != nil {
		err = w.writeRaw(node.TrailingSpace)
		if err != nil {
			return err
		}
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
		item := child.(*content.ListItem)
		if i > 1 && item.PreviousLineType() == content.PreviousLineTypeBlank {
			// Keep the blank line between the item and the item before it.
			err := w.out.WriteString("\n")
			if err != nil {
				return err
			}
		}

		var marker string
		if node.Type == content.ListTypeOrdered {
			// The number is kept as it is in the file.
			number := item.Number
			if number == "" {
				number = fmt.Sprintf("%d", i)
			}
			marker = number + string(node.Marker)
		} else {
			marker = string(node.Marker)
		}

		space := item.MarkerSpace
		if space == "" {
			space = " "
		}

		err := w.out.WriteString(marker + space)
		if err != nil {
			return err
		}

		// The lines after the first line of the item are indented as far
		// as its text starts.
		width := len(marker)
		for i := 0; i < len(space); i++ {
			width = nextColumn(width, space[i])
		}

		w.out.PushIndentation(strings.Repeat(" ", width))

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

// writeTable writes a table in the GitHub Flavored Markdown syntax. A table
// that is not changed is written with its text from the file. Other tables
// get cells that are padded so that the pipes of every row line up in a text
// editor.
func (w *Output) writeTable(node *content.Table) error {
	if source := node.SourceText(); source != "" && readsBackAsTable(source, node) {
		err := w.startBlock(node, "")
		if err != nil {
			return err
		}

		err = w.writeRaw(source)
		if err != nil {
			return err
		}

		w.endBlock()
		return nil
	}

	rows, err := w.tableRows(node)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return fmt.Errorf("table has no rows")
	}

	// The header and the delimiter row set the columns of the table. A row
	// can have more cells than that, and they are kept after the columns.
	columns := len(rows[0])
	if len(node.Alignments) > columns {
		// A column that the table sets the alignment of, but that no row has a
		// cell for, is still written out.
		columns = len(node.Alignments)
	}

	widths := tableColumnWidths(rows)
	for len(widths) < columns {
		widths = append(widths, 3)
	}

	err = w.startBlock(node, "")
	if err != nil {
		return err
	}

	// The header names the columns and is followed by the row that sets how
	// the text of each column lines up.
	err = w.writeRaw(tableRow(rows[0], widths[:columns]))
	if err != nil {
		return err
	}

	err = w.writeRaw("\n" + tableDelimiterRow(node, widths[:columns]))
	if err != nil {
		return err
	}

	for _, row := range rows[1:] {
		rowColumns := columns
		if len(row) > rowColumns {
			rowColumns = len(row)
		}

		err = w.writeRaw("\n" + tableRow(row, widths[:rowColumns]))
		if err != nil {
			return err
		}
	}

	w.endBlock()
	return nil
}

// readsBackAsTable returns true if the text of a table is read back as a
// table with the same rows and alignments. This is false after the table was
// changed, and then the table is written from its rows.
func readsBackAsTable(source string, node *content.Table) bool {
	block, err := ParseString(source)
	if err != nil {
		return false
	}

	parsed, ok := block.FirstChild().(*content.Table)
	if !ok || parsed.NextSibling() != nil {
		return false
	}

	if len(parsed.Alignments) != len(node.Alignments) {
		return false
	}

	for i := range parsed.Alignments {
		if parsed.Alignments[i] != node.Alignments[i] {
			return false
		}
	}

	a, b := parsed.FirstChild(), node.FirstChild()
	for ; a != nil && b != nil; a, b = a.NextSibling(), b.NextSibling() {
		if content.Debug(a) != content.Debug(b) {
			return false
		}
	}

	return a == nil && b == nil
}

// tableRows writes out the cells of every row, so that the width of each
// column is known before any of it is written.
func (w *Output) tableRows(node *content.Table) ([][]string, error) {
	rows := make([][]string, 0)
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		row, ok := child.(*content.TableRow)
		if !ok {
			return nil, fmt.Errorf("unsupported table child: %T", child)
		}

		cells := make([]string, 0)
		for cellNode := row.FirstChild(); cellNode != nil; cellNode = cellNode.NextSibling() {
			cell, ok := cellNode.(*content.TableCell)
			if !ok {
				return nil, fmt.Errorf("unsupported table row child: %T", cellNode)
			}

			value, err := w.tableCell(cell)
			if err != nil {
				return nil, err
			}

			cells = append(cells, value)
		}

		rows = append(rows, cells)
	}

	return rows, nil
}

// tableCell writes the content of a cell to the text that goes between the
// pipes of a row. A row is a single line, so line breaks become spaces, and
// the pipes in the content are escaped to keep them out of the shape of the
// table.
func (w *Output) tableCell(node *content.TableCell) (string, error) {
	out := strings.Builder{}
	cellWriter := &Output{
		out:  newWriter(&out),
		opts: w.opts,
	}

	err := cellWriter.writeChildren(node)
	if err != nil {
		return "", err
	}

	value := out.String()
	value = strings.ReplaceAll(value, "\\\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.ReplaceAll(value, "|", "\\|"), nil
}

// tableColumnWidths measures how wide each column has to be for the content of
// every cell in it to fit.
func tableColumnWidths(rows [][]string) []int {
	widths := make([]int, 0)
	for _, cells := range rows {
		for i, cell := range cells {
			// Three is the shortest a column can be, as that is what the row
			// that sets the alignment of a centered column needs.
			for i >= len(widths) {
				widths = append(widths, 3)
			}

			if width := utf8.RuneCountInString(cell); width > widths[i] {
				widths[i] = width
			}
		}
	}

	return widths
}

// tableRow writes one row of a table, padding every cell to the width of its
// column. Rows that are missing cells get empty ones, as a row that is shorter
// than the header is read back with the columns it does not have left empty.
func tableRow(cells []string, widths []int) string {
	row := strings.Builder{}
	for i, width := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}

		row.WriteString("| ")
		row.WriteString(cell)
		row.WriteString(strings.Repeat(" ", width-utf8.RuneCountInString(cell)))
		row.WriteString(" ")
	}

	row.WriteString("|")
	return row.String()
}

// tableDelimiterRow writes the row below the header, which sets how the text
// of each column lines up.
func tableDelimiterRow(node *content.Table, widths []int) string {
	row := strings.Builder{}
	for i, width := range widths {
		row.WriteString("| ")

		switch node.AlignmentOf(i) {
		case content.TableAlignmentLeft:
			row.WriteString(":" + strings.Repeat("-", width-1))
		case content.TableAlignmentRight:
			row.WriteString(strings.Repeat("-", width-1) + ":")
		case content.TableAlignmentCenter:
			row.WriteString(":" + strings.Repeat("-", width-2) + ":")
		default:
			row.WriteString(strings.Repeat("-", width))
		}

		row.WriteString(" ")
	}

	row.WriteString("|")
	return row.String()
}

func (w *Output) writeBlockquote(node *content.Blockquote) error {
	err := w.startBlock(node, "> ")
	if err != nil {
		return err
	}

	w.quoteLevels = append(w.quoteLevels, w.out.IndentationLevel()-1)
	defer func() {
		w.quoteLevels = w.quoteLevels[:len(w.quoteLevels)-1]

		// A lazy line is only a line of the quote, so no line after the
		// quote leaves out indentation.
		w.out.SkipIndentationOnNextLine(-1)
	}()

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

	fence := codeBlockFence(node.Code, node.Fence)
	err = w.writeRaw(fence)
	if err != nil {
		return err
	}

	err = w.writeRaw(codeBlockInfo(node))
	if err != nil {
		return err
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

	err = w.writeRaw(fence)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

// codeBlockFence gets the fence to write around a code block. It is the
// fence that the code block has, or ``` when it has none or the fence is not
// three or more "`" or "~". When a line of the code starts with as many or
// more of the fence character, that line would close the code block when the
// page is read again. Then the fence is one character longer than the
// longest such run, the same as CommonMark requires.
func codeBlockFence(code string, fence string) string {
	if !isCodeFence(fence) {
		fence = "```"
	}

	char := fence[:1]
	length := len(fence)
	for _, line := range strings.Split(code, "\n") {
		line = strings.TrimLeft(line, " \t")
		run := len(line) - len(strings.TrimLeft(line, char))
		if run >= length {
			length = run + 1
		}
	}

	return strings.Repeat(char, length)
}

// isCodeFence checks if text is three or more "`" or three or more "~".
func isCodeFence(fence string) bool {
	if len(fence) < 3 || (fence[0] != '`' && fence[0] != '~') {
		return false
	}

	return strings.Trim(fence, fence[:1]) == ""
}

// codeBlockInfo gets the text to write after the opening fence of a code
// block. This is the text as it is in the file when it still gives the same
// language and options, else the language and the options with one space
// between them. The options are only written when there is a language.
func codeBlockInfo(node *content.CodeBlock) string {
	if source := node.InfoSource(); source != "" && !strings.ContainsAny(source, "\r\n") {
		language, options := splitCodeInfo(source)
		if language == node.Language && options == node.Options {
			return source
		}
	}

	if node.Language == "" || node.Options == "" {
		return node.Language
	}

	return node.Language + " " + node.Options
}

func (w *Output) writeThematicBreak(node *content.ThematicBreak) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	marker := "---"
	if isThematicBreak([]byte(node.Marker)) {
		marker = node.Marker
	}

	err = w.writeRaw(marker)
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
	if _, ok := node.Parent().(*content.Block); ok && !writesWithoutBullet(node) {
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
		hasContent := w.out.HasWrittenAtCurrentIndent()
		if hasContent {
			err := w.out.WriteString("\n")
			if err != nil {
				return err
			}
		}

		if hasParentBlock || writesWithoutBullet(node) {
			// The children of the pre-block are indented with a tab so that
			// they are read back as its children and not as top-level blocks.
			w.out.PushIndentation("\t")
		} else {
			w.out.PushIndentation("")
		}

		i := 0
		for _, child := range blocks {
			i++

			if child.PreviousLineType() == content.PreviousLineTypeBlank && (i > 1 || hasContent) && writesWithoutBullet(child) {
				// Keep the blank line before a heading block.
				err := w.out.WriteString("\n")
				if err != nil {
					return err
				}
			}

			if writesWithoutBullet(child) {
				// A pre-block or a heading block is written as is, with no
				// marker to indent under. It gets an indentation level of its
				// own, so that what was written before it does not count as
				// content of the block.
				//
				// A heading block of a parent block is written at the column
				// of the text of the parent, where it was read. The tab that
				// indents the sub blocks is wider than the text of the parent,
				// and a heading at a deeper column reads as text after a few
				// saves.
				replaceIndent := hasParentBlock && previousIndent != ""
				if replaceIndent {
					w.out.PopIndentation()
					w.out.PushIndentation(previousIndent)
				}

				w.out.PushIndentation("")
				err := w.Write(child)
				w.out.PopIndentation()

				if replaceIndent {
					w.out.PopIndentation()
					w.out.PushIndentation("\t")
				}
				if err != nil {
					return err
				}

				if child.NextSibling() != nil {
					err = w.writeRaw("\n")
					if err != nil {
						return err
					}
				}

				continue
			}

			if child.PreviousLineType() == content.PreviousLineTypeBlank && (i > 1 || hasContent) {
				// Keep the blank line between the block and what is written
				// before it.
				err := w.out.WriteString("\n")
				if err != nil {
					return err
				}
			}

			err := w.out.WriteString(bulletMarker(child))
			if err != nil {
				return err
			}

			w.out.PushIndentation("  ")
			w.blockTextLevels = append(w.blockTextLevels, w.out.IndentationLevel()-1)

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

			w.blockTextLevels = w.blockTextLevels[:len(w.blockTextLevels)-1]
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

// bulletMarker gets the text that starts the line of the bullet of a block.
// It is `- `, or the dash and the spaces after it as they are in the file, see
// content.Block.BulletSpace, and the content of the block follows it on the
// same line. If the
// line of the bullet has no text, the `-` is written with the spaces that
// followed it, and the content starts on the next line, see
// content.Block.WithEmptyFirstLine.
func bulletMarker(block *content.Block) string {
	if !block.HasEmptyFirstLine() {
		if space := block.BulletSpace(); space != "" && len(block.Content()) > 0 {
			return "-" + space
		}

		return "- "
	}

	marker := "-" + block.EmptyFirstLineSpace()
	if len(block.Content()) > 0 {
		marker += "\n"
	}

	return marker
}

// writesWithoutBullet checks if a block is written without a bullet. This is
// the case in the positions where the bullet-less form parses back into the
// same structure: a pre-block as the first block of a page, a heading block
// as a top-level block that starts with a heading, and a heading block that
// starts with a heading in a block that is written with a bullet and has
// content of its own.
func writesWithoutBullet(node *content.Block) bool {
	parent, ok := node.Parent().(*content.Block)
	if !ok {
		return false
	}

	if parent.Parent() != nil {
		// A heading line in a block starts a child block only after the
		// first line of the block, see convertOutline.
		if !node.IsHeadingBlock() || writesWithoutBullet(parent) {
			return false
		}

		_, ok := node.FirstChild().(*content.Heading)
		return ok
	}

	if node.IsPreBlock() {
		return node.PreviousSibling() == nil
	}

	if node.IsHeadingBlock() {
		_, ok := node.FirstChild().(*content.Heading)
		return ok
	}

	return false
}

func (w *Output) writeProperties(node *content.Properties) error {
	if node.Drawer {
		return w.writePropertyDrawer(node)
	}

	err := w.startBlockWithAutomaticBehavior(node, "", false)
	if err != nil {
		return err
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*content.Property); !ok {
			return fmt.Errorf("unsupported properties child: %T", child)
		}

		property := child.(*content.Property)
		err := w.writePropertyName(property)
		if err != nil {
			return err
		}

		err = w.writeChildren(property)
		if err != nil {
			return err
		}

		err = w.writeRaw(property.TrailingSpace)
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

// writePropertyName writes the name of a property and the separator before
// its value, as `name:: ` or, for a directive, as `#+name: `. A property with
// an empty value can be written as `name::`, see
// content.Property.NoSpaceAfterSeparator.
//
// If the property has a value, the spaces and tabs that were written before
// the value are written again, see content.Property.SpaceBeforeValue.
func (w *Output) writePropertyName(property *content.Property) error {
	space := ""
	if property.FirstChild() != nil {
		space = property.SpaceBeforeValue
	}

	if !property.Directive {
		if property.FirstChild() == nil && property.NoSpaceAfterSeparator {
			return w.writeRaw(property.Name + "::")
		}

		return w.writeRaw(property.Name + ":: " + space)
	}

	if property.FirstChild() == nil {
		return w.writeRaw("#+" + property.Name + ":")
	}

	return w.writeRaw("#+" + property.Name + ": " + space)
}

// writePropertyDrawer writes properties in a `:PROPERTIES:` drawer, with one
// `:name: value` line for each property.
func (w *Output) writePropertyDrawer(node *content.Properties) error {
	err := w.startBlockWithAutomaticBehavior(node, "", false)
	if err != nil {
		return err
	}

	opening, closing := ":PROPERTIES:", ":END:"
	if node.DrawerLowerCase {
		opening, closing = ":properties:", ":end:"
	}

	err = w.writeRaw(opening + "\n")
	if err != nil {
		return err
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		property, ok := child.(*content.Property)
		if !ok {
			return fmt.Errorf("unsupported properties child: %T", child)
		}

		err = w.writeRaw(":" + property.Name + ":")
		if err != nil {
			return err
		}

		if property.FirstChild() != nil {
			err = w.writeRaw(" ")
			if err != nil {
				return err
			}

			err = w.writeChildren(property)
			if err != nil {
				return err
			}
		}

		err = w.writeRaw("\n")
		if err != nil {
			return err
		}
	}

	err = w.writeRaw(closing)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

// writeFrontMatter writes YAML front matter. The text is written as it was
// read, unless the properties in it were changed. Then the front matter is
// written from the properties, with one `name: value` line for each property.
func (w *Output) writeFrontMatter(node *content.FrontMatter) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	changed, err := frontMatterPropertiesChanged(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("---\n")
	if err != nil {
		return err
	}

	if !changed {
		err = w.writeRaw(node.Raw)
		if err != nil {
			return err
		}
	} else {
		for child := node.FindProperties().FirstChild(); child != nil; child = child.NextSibling() {
			property, ok := child.(*content.Property)
			if !ok {
				return fmt.Errorf("unsupported properties child: %T", child)
			}

			err = w.writeRaw(property.Name + ": ")
			if err != nil {
				return err
			}

			err = w.writeChildren(property)
			if err != nil {
				return err
			}

			err = w.writeRaw("\n")
			if err != nil {
				return err
			}
		}
	}

	err = w.writeRaw("---")
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

// frontMatterPropertiesChanged checks if the properties of front matter are
// different from the properties in its text. Front matter without properties
// can not be changed, as it holds YAML that Logseq does not read.
func frontMatterPropertiesChanged(node *content.FrontMatter) (bool, error) {
	properties := node.FindProperties()
	if properties == nil {
		return false, nil
	}

	defaults := defaultParseOptions()
	original, err := readFrontMatterProperties(node.Raw, &defaults)
	if err != nil {
		return false, err
	}

	if original == nil {
		return true, nil
	}

	current, err := AsString(properties)
	if err != nil {
		return false, err
	}

	read, err := AsString(original)
	if err != nil {
		return false, err
	}

	return current != read, nil
}

func (w *Output) writeAdvancedCommand(node *content.AdvancedCommand) error {
	if node.FirstChild() == nil {
		return w.writeBeginEnd(node, beginEndLinesOf(node), node.Value)
	}

	body, err := w.commandBody(node)
	if err != nil {
		return err
	}

	return w.writeBeginEnd(node, beginEndLinesOf(node), body)
}

// commandBody returns the text to write between the BEGIN and END lines of a
// command with children. The text that was read is used while the children
// give the same text as it, so that a command that was not changed is written
// as it was read.
//
// When the children were changed, only the lines that changed are taken from
// the children, and the other lines are kept as they were read. The writer
// does not always give back the exact text it read, such as `- b` that is
// written as `* b`, and those lines did not change. If the result does not
// read back as the children, the text is written from the children.
func (w *Output) commandBody(node *content.AdvancedCommand) (string, error) {
	current, err := w.writeNodes(node)
	if err != nil {
		return "", err
	}

	read, err := parseCommandBody(node.Value, nil)
	if err != nil {
		return "", err
	}

	original, err := w.writeNodes(content.NewAdvancedCommand(node.Type, node.Value, read...))
	if err != nil {
		return "", err
	}

	if current == original {
		return node.Value, nil
	}

	changed := current + "\n"
	merged, ok := textmerge.Merge(original+"\n", node.Value, changed)
	if !ok {
		return changed, nil
	}

	check, err := parseCommandBody(merged, nil)
	if err != nil {
		return changed, nil
	}

	checked, err := w.writeNodes(content.NewAdvancedCommand(node.Type, merged, check...))
	if err != nil || checked != current {
		return changed, nil
	}

	return merged, nil
}

// writeNodes writes the children of a node on their own, with the settings of
// this writer.
func (w *Output) writeNodes(node content.HasChildren) (string, error) {
	var out strings.Builder
	nested := &Output{
		out:  newWriter(&out),
		opts: w.opts,
	}

	err := nested.writeChildren(node)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

// beginEndLines is how the BEGIN and END lines of a command are written.
type beginEndLines struct {
	variant    string
	parameters string
	lowerCase  bool
}

func beginEndLinesOf(node *content.AdvancedCommand) beginEndLines {
	return beginEndLines{node.Type, node.Parameters, node.LowerCase}
}

// begin is the BEGIN line, without the line break.
func (l beginEndLines) begin() string {
	line := "#+BEGIN_"
	if l.lowerCase {
		line = "#+begin_"
	}

	line += l.variant
	if l.parameters != "" {
		line += " " + l.parameters
	}

	return line
}

// end is the END line, without the line break.
func (l beginEndLines) end() string {
	if l.lowerCase {
		return "#+end_" + l.variant
	}

	return "#+END_" + l.variant
}

func (w *Output) writeBeginEnd(node content.BlockNode, lines beginEndLines, value string) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw(lines.begin() + "\n")
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

	err = w.writeRaw(lines.end())
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeTaskMarker(node *content.TaskMarker) error {
	if node.Status == content.TaskStatusNone {
		return nil
	}

	marker := node.Status.String()
	if marker == "" {
		return fmt.Errorf("unsupported task status: %d", node.Status)
	}

	// A space always follows the marker, as Logseq reads a marker that is
	// followed by a line break as text.
	return w.writeRaw(marker + " ")
}

func (w *Output) writeTaskPriority(node *content.TaskPriority) error {
	if node.Priority == content.PriorityNone {
		return nil
	}

	value := node.Priority.String()
	if value == "" || value == "\n" || value == "\r" {
		return fmt.Errorf("unsupported priority: %d", node.Priority)
	}

	err := w.writeRaw("[#" + value + "]")
	if err != nil {
		return err
	}

	if node.NextSibling() != nil && node.SpaceAfter() {
		err = w.writeRaw(" ")
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Output) writeTaskDate(node *content.TaskDate) error {
	var keyword string
	switch node.Type {
	case content.TaskDateTypeScheduled:
		keyword = "SCHEDULED"
	case content.TaskDateTypeDeadline:
		keyword = "DEADLINE"
	default:
		return fmt.Errorf("unsupported task date type: %d", node.Type)
	}

	value := strings.Builder{}
	if source := node.DateSource(); source != "" && taskDateReadsBack(source, node) {
		// The date is not changed, so its text is written as it is.
		value.WriteString(source)
	} else {
		err := writeTaskDateValue(&value, keyword, node)
		if err != nil {
			return err
		}
	}

	// Task dates belong to the line above them, so they never get a blank line
	// of their own when the previous line type is automatic.
	err := w.startBlockWithAutomaticBehavior(node, "", false)
	if err != nil {
		return err
	}

	err = w.writeRaw(node.Indentation + value.String())
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

// writeTaskDateValue writes a task date in the form that Logseq writes, such
// as `SCHEDULED: <2024-01-15 Mon 09:00 .+3d>`.
func writeTaskDateValue(value *strings.Builder, keyword string, node *content.TaskDate) error {
	value.WriteString(keyword)
	value.WriteString(": <")
	value.WriteString(node.Date.Format("2006-01-02 Mon"))

	if node.HasTime {
		value.WriteString(node.Date.Format(" 15:04"))
	}

	if node.Repeater != nil {
		repeater := node.Repeater.String()
		if repeater == "" {
			return fmt.Errorf("unsupported repeater: %+v", *node.Repeater)
		}

		value.WriteString(" ")
		value.WriteString(repeater)
	}

	value.WriteString(">")
	return nil
}

// logbookLayout is the timestamp format of the graph, which leaves the seconds
// out when the graph is set up without second support.
func (w *Output) logbookLayout() string {
	if w.opts.logbookWithSeconds {
		return logbookTimeLayout
	}

	return logbookTimeLayoutWithoutSeconds
}

// logbookClock formats a clock entry, deriving the duration from the two
// times so it always matches them.
//
// The duration is the difference between the wall clock times as they are
// written, the same as Logseq computes it. On a day when the clocks change,
// it is not the time that went by: 01:30 to 03:30 is 02:00 also when the
// clocks go forward between them.
func (w *Output) logbookClock(node *content.LogbookEntryClock) (string, error) {
	layout := w.logbookLayout()

	value := "CLOCK: [" + node.Start.Format(layout) + "]"
	if node.IsRunning() {
		return value, nil
	}

	if node.Duration() < 0 {
		return "", fmt.Errorf("logbook clock ends before it starts: %s", value)
	}

	value += "--[" + node.End.Format(layout) + "] =>  "

	start, end := wallClock(node.Start), wallClock(node.End)
	if !w.opts.logbookWithSeconds {
		// The duration is measured between the times as they are written, so
		// that it adds up for whoever reads the entry.
		start, end = start.Truncate(time.Minute), end.Truncate(time.Minute)
	}

	duration := end.Sub(start)
	if duration < 0 {
		// The clocks went back between the two times, so the end is written
		// before the start. The time that went by is used instead.
		duration = node.Duration()
	}

	if !w.opts.logbookWithSeconds {
		return value + fmt.Sprintf(
			"%02d:%02d",
			int(duration/time.Hour),
			int(duration/time.Minute)%60,
		), nil
	}

	return value + fmt.Sprintf(
		"%02d:%02d:%02d",
		int(duration/time.Hour),
		int(duration/time.Minute)%60,
		int(duration/time.Second)%60,
	), nil
}

// wallClock gets the wall clock time of a time, which is its date, hour,
// minute and second in its own time zone, as a time in UTC.
func wallClock(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}

func (w *Output) logbookStateChange(node *content.LogbookEntryStateChange) (string, error) {
	to := node.To.String()
	if to == "" {
		return "", fmt.Errorf("unsupported task status: %d", node.To)
	}

	value := `* State "` + to + `"`

	if node.From != content.TaskStatusNone {
		from := node.From.String()
		if from == "" {
			return "", fmt.Errorf("unsupported task status: %d", node.From)
		}

		value += ` from "` + from + `"`
	}

	return value + " [" + node.Time.Format(w.logbookLayout()) + "]", nil
}

func (w *Output) writeLogbook(node *content.Logbook) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw(":LOGBOOK:" + node.StartLineText + "\n")
	if err != nil {
		return err
	}

	for _, entry := range node.Children() {
		var value string
		if aware, ok := entry.(content.LogbookEntrySourceAware); ok {
			if source := aware.EntrySource(); source != "" && logbookEntryReadsBack(source, entry.(content.LogbookEntry)) {
				// The entry is not changed, so its text is written as it is.
				value = source
			}
		}

		switch e := entry.(type) {
		case *content.LogbookEntryRaw:
			value = e.Value
		case *content.LogbookEntryClock:
			if value == "" {
				value, err = w.logbookClock(e)
			}
		case *content.LogbookEntryStateChange:
			if value == "" {
				value, err = w.logbookStateChange(e)
			}
		default:
			return fmt.Errorf("unsupported logbook entry: %T", entry)
		}

		if err != nil {
			return err
		}

		err = w.writeRaw(value)
		if err != nil {
			return err
		}

		if !strings.HasSuffix(value, "\n") {
			err = w.writeRaw("\n")
			if err != nil {
				return err
			}
		}
	}

	err = w.writeRaw(":END:")
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}
