package markdown

import (
	"bytes"

	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Some inline parsers look for the end of what they read, such as the `]]`
// of a page link, from each place where it can start. On a long line with
// many starts and no end, looking to the end of the line each time makes the
// time to read the line grow with the square of its length. The searches here
// remember where they found the last match, so that all of them together read
// the source about once.

// nextMatch remembers where a search for the first match at or after an
// offset found it. A later search from an offset that is not before the
// last one, and not after the match it found, has the same answer.
type nextMatch struct {
	valid bool
	from  int
	at    int
}

// find gives the first match at or after the offset, or -1 if there is none.
// search does the search when the remembered answer cannot be used.
func (m *nextMatch) find(from int, search func(from int) int) int {
	if m.valid && from >= m.from && (m.at < 0 || m.at >= from) {
		return m.at
	}

	m.valid = true
	m.from = from
	m.at = search(from)
	return m.at
}

// sourceSearchesKey is where the sourceSearches of the current parse are
// kept.
var sourceSearchesKey = parser.NewContextKey()

// sourceSearches are the searches for text in the source of a parse, one for
// each text that is looked for.
type sourceSearches struct {
	src      []byte
	searches map[string]*nextMatch
}

// comesLater checks if the text comes later on the line that the reader is
// on, at or after the given number of bytes from the position of the reader.
func comesLater(block text.Reader, pc parser.Context, needle string, skip int) bool {
	line, segment := block.PeekLine()
	if skip > len(line) {
		return false
	}

	if segment.Padding != 0 || pc == nil {
		// The line does not match the source, so it is searched on its own.
		return bytes.Contains(line[skip:], []byte(needle))
	}

	at := nextInSource(block.Source(), pc, needle, segment.Start+skip)
	return at >= 0 && at+len(needle) <= segment.Start+len(line)
}

// nextInSource finds the first place of the text in the source at or after
// the offset, or -1 if it is not there.
func nextInSource(source []byte, pc parser.Context, needle string, from int) int {
	searches, _ := pc.Get(sourceSearchesKey).(*sourceSearches)
	if searches == nil || len(searches.src) != len(source) || (len(source) > 0 && &searches.src[0] != &source[0]) {
		searches = &sourceSearches{src: source, searches: map[string]*nextMatch{}}
		pc.Set(sourceSearchesKey, searches)
	}

	search := searches.searches[needle]
	if search == nil {
		search = &nextMatch{}
		searches.searches[needle] = search
	}

	return search.find(from, func(from int) int {
		if i := bytes.Index(source[from:], []byte(needle)); i >= 0 {
			return from + i
		}

		return -1
	})
}
