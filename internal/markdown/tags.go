package markdown

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var tagKind = ast.NewNodeKind("Hashtag")

type tag struct {
	ast.BaseInline
	Page string

	// Segment is the source of the whole tag, from the # to the end of the
	// name or the closing ]].
	Segment text.Segment

	// Extended is true for a tag in the form #[[name]].
	Extended bool
}

func (*tag) Kind() ast.NodeKind {
	return tagKind
}

func (n *tag) Dump(src []byte, level int) {
}

// tagDelimiters are the characters that can end a tag name. See tagNameLength
// for the rule.
const tagDelimiters = ",;.!?'\":#"

// tagInnerDelimiters are the tag delimiters that can also be in the middle
// of a tag name, so "#a.b" has the tag "a.b".
const tagInnerDelimiters = ".;"

// tagPrefixes are the characters that can come directly before the # of a
// tag, in addition to Unicode spaces. Logseq accepts the tag delimiters, so
// "a,#tag" has the tag "tag". Logseq also makes a tag after "[", so
// "[#[[a tag]]]" has the tag "a tag".
const tagPrefixes = tagDelimiters + "["

// tagParser parses Logseq style tags in Goldmark.
//
// A # starts a tag only at the start of the text, after a Unicode space or
// after one of tagPrefixes. Thus "C#Sharp" and "https://a.com/#top" do not
// contain tags. A directive at the start of a line, such as `#+title: X`, is
// not a tag either.
type tagParser struct {
}

func (t *tagParser) Trigger() []byte {
	return []byte{'#'}
}

func (t *tagParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, seg := block.PeekLine()

	if len(line) == 0 || line[0] != '#' {
		return nil
	}

	if isDirective(line) && atLineStart(parent, block) {
		// A line such as `#+title: X` is a directive, see directives.go.
		return nil
	}

	prev := block.PrecendingCharacter()
	if !unicode.IsSpace(prev) && !strings.ContainsRune(tagPrefixes, prev) && !followsTag(parent) {
		return nil
	}

	line = line[1:]

	// Logseq does not make tags in the text of a link, so
	// "[Teams:#](https://a.com/)" is a link and not a tag.
	if insideLinkLabel(pc) && comesLater(block, pc, "]", 1) && closesAsInlineLink(line) {
		return nil
	}

	// Two # in sequence is not a tag, but the second # can start a tag.
	if len(line) > 0 && line[0] == '#' {
		return nil
	}

	end := 0
	var value []byte
	extended := false

	// Check for extended tag syntax which wraps tags in [[...]], this is how
	// Logseq supports tags with spaces.
	if len(line) > 1 && line[0] == '[' && line[1] == '[' {
		if !comesLater(block, pc, "]]", 3) {
			return nil
		}

		// This is an extended tag, which ends in the same way as a page link.
		end = pageLinkClose(line)

		// Didn't find the closing ]], so this isn't a tag.
		if end == 0 {
			return nil
		}

		// The value of the tag is the text between the [[ and ]].
		value = line[2:end]
		end += 2
		extended = true
	} else {
		end = tagNameLength(line)
		if closer := formattingCloserIndex(line, end, pc); closer >= 0 {
			end = closer
		}
		value = line[:end]
	}

	if len(value) == 0 {
		// A tag needs a name, so a # on its own is just text.
		return nil
	}

	seg = seg.WithStop(seg.Start + end + 1)

	n := tag{
		Page:     string(value),
		Segment:  seg,
		Extended: extended,
	}
	block.Advance(seg.Len())
	return &n
}

// tagNameLength returns the length in bytes of the tag name at the start of
// line. It uses the same rule as Logseq:
//
//   - A space, a tab, a line break or a form feed ends the name, see
//     isTagSpace. Other Unicode spaces, such as a non-breaking space, are
//     part of the name, so "#a\u00a0b" has the tag "a\u00a0b".
//   - A run of tag delimiters before a space or the end of line ends the
//     name, so "#tag,.?" has the tag "tag" followed by the text ",.?".
//   - Other tag delimiters end the name, except "." and ";" which stay in
//     it. Thus "#a:b" has the tag "a", and "#a.b" has the tag "a.b".
func tagNameLength(line []byte) int {
	for i := 0; i < len(line); {
		r, size := utf8.DecodeRune(line[i:])
		if isTagSpace(r) {
			return i
		}

		if strings.ContainsRune(tagDelimiters, r) {
			j := i
			for j < len(line) && strings.IndexByte(tagDelimiters, line[j]) >= 0 {
				j++
			}

			if j == len(line) {
				return i
			}

			next, _ := utf8.DecodeRune(line[j:])
			if isTagSpace(next) || !strings.ContainsRune(tagInnerDelimiters, r) {
				return i
			}
		}

		i += size
	}

	return len(line)
}

// isTagSpace checks if the character ends the name of a tag. Logseq (mldoc)
// ends a tag only at ASCII whitespace other than the vertical tab: a space, a
// tab, a line break, a carriage return or a form feed.
func isTagSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f'
}

// formattingCloserIndex returns the position of the first run of markers in
// the tag name line[:end] that closes emphasis, strong text, strikethrough or
// a highlight that is open before the tag. Returns -1 if there is no such run.
//
// Logseq reads the formatting before the tag, so the run ends the name. Thus
// "*see #x*" is italic text, and the name does not take the closing "*".
// A run that cannot close stays in the name, so "_a #snake_case" has the tag
// "snake_case".
func formattingCloserIndex(line []byte, end int, pc parser.Context) int {
	if pc.LastDelimiter() == nil {
		return -1
	}

	// The search starts after the first character, so that the name is not
	// empty.
	for i := 1; i < end; i++ {
		processor, min := delimiterProcessorFor(line[i])
		if processor == nil || !hasOpenDelimiter(pc, line[i]) {
			continue
		}

		before, _ := utf8.DecodeLastRune(line[:i])
		closer := scanDelimiter(line[i:], before, min, processor)
		if closer != nil && closer.CanClose {
			return i
		}
	}

	return -1
}

// hasOpenDelimiter returns true if a run of the marker c that can open
// formatting is waiting for a closing run.
func hasOpenDelimiter(pc parser.Context, c byte) bool {
	for d := pc.LastDelimiter(); d != nil; d = d.PreviousDelimiter {
		if d.Char == c && d.CanOpen {
			return true
		}
	}

	return false
}

// tagsInFormattingASTTransformer turns tags inside emphasis, strong text,
// strikethrough and highlights back into text. Logseq reads the formatting
// before the tags, and the text of formatting has no tags. Thus "*see #x*"
// has no tag. A page link stays a page link, so "*see #[[a b]]*" has the text
// "see #" and the page link "a b".
type tagsInFormattingASTTransformer struct {
}

var defaultTagsInFormattingASTTransformer = &tagsInFormattingASTTransformer{}

func (t *tagsInFormattingASTTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	// The tags are found before they are replaced, as a replaced node has no
	// next sibling for the walk to continue to.
	var tags []*tag
	_ = ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if hashtag, ok := node.(*tag); ok && entering && insideFormatting(hashtag) {
			tags = append(tags, hashtag)
		}

		return ast.WalkContinue, nil
	})

	for _, hashtag := range tags {
		parent := hashtag.Parent()
		if hashtag.Extended {
			hash := ast.NewTextSegment(hashtag.Segment.WithStop(hashtag.Segment.Start + 1))
			parent.InsertBefore(parent, hashtag, hash)
			parent.ReplaceChild(parent, hashtag, &pageLink{Page: hashtag.Page})
		} else {
			// The text is next to the text around it in the source, so it is
			// merged with that text when it is converted.
			parent.ReplaceChild(parent, hashtag, ast.NewTextSegment(hashtag.Segment))
		}
	}
}

// insideFormatting returns true if the node is inside emphasis, strong text,
// strikethrough or a highlight.
func insideFormatting(node ast.Node) bool {
	for p := node.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case ast.KindEmphasis, east.KindStrikethrough, highlightKind:
			return true
		}

		if p.Type() == ast.TypeBlock {
			return false
		}
	}

	return false
}

var _ parser.ASTTransformer = (*tagsInFormattingASTTransformer)(nil)

// followsTag returns true if the last inline node of parent is a tag. A tag
// can end at a "#" that starts the next tag, so "#a#b" has the tags "a" and
// "b".
func followsTag(parent ast.Node) bool {
	_, ok := parent.LastChild().(*tag)
	return ok
}

// insideLinkLabel returns true if the inline content has a "[" or "![" that
// Goldmark has not closed yet. Goldmark keeps the open brackets in the parse
// context until it finds the closing "]", so this does not have to look at
// the content that is already parsed, which would make a long line slow.
func insideLinkLabel(pc parser.Context) bool {
	return pc.IsInLinkLabel()
}

// closesAsInlineLink returns true if rest has the "]" that closes the open
// link label, and a "(" comes directly after it. Brackets in rest must be in
// balanced pairs, and a backslash escapes the character after it.
func closesAsInlineLink(rest []byte) bool {
	depth := 0
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '\\':
			i++
		case '[':
			depth++
		case ']':
			if depth == 0 {
				return i+1 < len(rest) && rest[i+1] == '('
			}
			depth--
		}
	}
	return false
}

var _ parser.InlineParser = (*tagParser)(nil)
