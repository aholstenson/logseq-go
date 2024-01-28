package logseq

import (
	"time"

	"github.com/aholstenson/logseq-go/content"
)

type JournalOption func(*Journal)

// WithBlockTime sets the time format to use for timestamps on blocks added to
// the journal.
func WithBlockTime(format string) JournalOption {
	return func(o *Journal) {
		o.timeFormat = format
	}
}

// WithBlockTime24Hour sets the time format to use for timestamps on blocks
// added to the journal to 24 hour format.
func WithBlockTime24Hour() JournalOption {
	return WithBlockTime("15:04")
}

// WithBlockTime12Hour sets the time format to use for timestamps on blocks
// added to the journal to 12 hour format.
func WithBlockTime12Hour() JournalOption {
	return WithBlockTime("3:04 PM")
}

// WithBlockTimeFormatter sets the function to use for formatting the timestamp
// on blocks added to the journal. If not set, the default is to use a bold
// timestamp.
func WithBlockTimeFormatter(f func(string) content.InlineNode) JournalOption {
	return func(o *Journal) {
		o.timeFormatToNode = f
	}
}

// Journal is a helper for updating Logseq journals. It provides methods for
// adding blocks to journal pages based on a specific time.
//
// It can optionally timestamp the blocks with a time.
type Journal struct {
	tx *Transaction

	timeFormat       string
	timeFormatToNode func(string) content.InlineNode
}

func newJournal(tx *Transaction, opts ...JournalOption) *Journal {
	j := &Journal{
		tx: tx,
	}

	for _, opt := range opts {
		opt(j)
	}

	if j.timeFormatToNode == nil {
		j.timeFormatToNode = func(s string) content.InlineNode {
			return content.NewStrong(content.NewText(s))
		}
	}

	return j
}

// AddBlock adds a block to the journal page for the given date.
func (j *Journal) AddBlock(time time.Time, block *content.Block) error {
	// Change the timezone to the local one
	time = time.Local()

	page, err := j.tx.OpenJournalPage(time)
	if err != nil {
		return err
	}

	// Go through all the blocks on the page and figure out where we fit in
	var insertAfter *content.Block
	for _, b := range page.Blocks() {
		t := parseTime(j.timeFormat, time, b)
		if t != nil && t.After(time) {
			break
		}

		if b.FirstChild() != nil {
			insertAfter = b
		}
	}

	if j.timeFormat != "" {
		// Add the timestamp to the block
		timeNode := j.timeFormatToNode(time.Format(j.timeFormat))
		firstChild := block.FirstChild()
		if p := firstChild.(*content.Paragraph); p != nil {
			p.PrependChild(timeNode)
			p.InsertChildAfter(content.NewText(" "), timeNode)
		} else {
			block.PrependChild(content.NewParagraph(timeNode, content.NewText(" ")))
		}
	}

	if insertAfter == nil {
		// All blocks have timestamps after the new block, prepend it
		page.PrependBlock(block)
	} else {
		// Insert the block after the block with the timestamp before the new
		// block, or at the end of the page if there are no timestamps
		page.InsertBlockAfter(block, insertAfter)
	}

	return nil
}

func parseTime(format string, reference time.Time, block *content.Block) *time.Time {
	firstText := block.Children().FindDeep(content.IsOfType[*content.Text]())
	if firstText == nil {
		return nil
	}

	// The first text node should be the timestamp
	text := firstText.(*content.Text)
	if text == nil {
		return nil
	}

	t, err := time.Parse(format, text.Value)
	if err != nil {
		return nil
	}

	// Combine the date and time
	t = time.Date(reference.Year(), reference.Month(), reference.Day(), t.Hour(), t.Minute(), 0, 0, reference.Location())
	return &t
}
