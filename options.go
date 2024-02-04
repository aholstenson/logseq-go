package logseq

import "github.com/aholstenson/logseq-go/content"

type Option func(*options)

type options struct {
	blockTimeFormat       string
	blockTimeFormatToNode func(string) content.InlineNode
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
