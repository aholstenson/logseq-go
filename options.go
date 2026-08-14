package logseq

import "github.com/aholstenson/logseq-go/content"

type Option func(*options)

type options struct {
	index          bool
	indexDirectory string

	recycleDeletedPages bool

	listener func(event OpenEvent)

	blockTimeFormat       string
	blockTimeFormatToNode func(string) content.InlineNode
}

// WithIndex enables indexing of the graph in the given directory. The index
// will only be partially rebuilt when the graph is opened.
func WithIndex(directory string) Option {
	return func(o *options) {
		o.index = true
		o.indexDirectory = directory
	}
}

// WithInMemoryIndex enables indexing of the graph in memory. This will rebuild
// the index when the graph is opened.
func WithInMemoryIndex() Option {
	return func(o *options) {
		o.index = true
		o.indexDirectory = ""
	}
}

// WithRecycleDeletedPages makes deleted pages move into the `logseq/.recycle`
// directory of the graph instead of being removed. That is where Logseq keeps
// pages deleted in the app, so they can be recovered by moving them back.
//
// Files in the recycle directory keep their name, so recycling a page whose
// name is already there replaces the previously recycled file.
func WithRecycleDeletedPages() Option {
	return func(o *options) {
		o.recycleDeletedPages = true
	}
}

// WithListener sets a listener that will be invoked for events that occur
// while the graph is being opened.
func WithListener(listener func(event OpenEvent)) Option {
	return func(o *options) {
		o.listener = listener
	}
}

// WithBlockTime sets the time format to use for timestamps on blocks added to
// the journal.
func WithBlockTime(format string) Option {
	return func(o *options) {
		o.blockTimeFormat = format
	}
}

// WithBlockTime24Hour sets the time format to use for timestamps on blocks
// added to the journal to 24 hour format.
func WithBlockTime24Hour() Option {
	return WithBlockTime("15:04")
}

// WithBlockTime12Hour sets the time format to use for timestamps on blocks
// added to the journal to 12 hour format.
func WithBlockTime12Hour() Option {
	return WithBlockTime("3:04 PM")
}

// WithBlockTimeFormatter sets the function to use for formatting the timestamp
// on blocks added to the journal. If not set, the default is to use a bold
// timestamp.
func WithBlockTimeFormatter(f func(string) content.InlineNode) Option {
	return func(o *options) {
		o.blockTimeFormatToNode = f
	}
}
