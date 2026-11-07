package markdown

import (
	"io"
)

type writer struct {
	output io.Writer

	currentIndent []byte
	indentLengths []int

	trailingSpaceIndex int
	didWrite           []bool

	lastWasLineBreak bool

	// lazyLevel is the indentation level of a quote marker that is left out
	// of the next line, or -1. See SkipIndentationOnNextLine.
	lazyLevel int

	// lineEnding is written for each line break, or "\n" if it is empty.
	lineEnding string
}

func newWriter(out io.Writer) *writer {
	return &writer{
		output:           out,
		lastWasLineBreak: true,
		didWrite:         []bool{false},
		lazyLevel:        -1,
	}
}

// SkipIndentationOnNextLine leaves the indentation of the given level, and of
// the levels after it, out of the next line that has text. It is used for a
// lazy line of a quote, which is written without its `>` marker.
func (w *writer) SkipIndentationOnNextLine(level int) {
	w.lazyLevel = level
}

// WriteIndentationOnly writes the indentation of a new line without its
// trailing spaces, such as the `>` of a line of a quote that has no text. It
// does not count as text that is written at the current indentation.
func (w *writer) WriteIndentationOnly() error {
	if !w.lastWasLineBreak {
		return nil
	}

	_, err := w.output.Write(w.OnlyIndent())
	w.lastWasLineBreak = false
	return err
}

// WriteLineBreak writes a line break. It does not count as text that is
// written at the current indentation.
func (w *writer) WriteLineBreak() error {
	var err error
	if w.lineEnding == "" {
		_, err = io.WriteString(w.output, "\n")
	} else {
		_, err = io.WriteString(w.output, w.lineEnding)
	}

	w.lastWasLineBreak = true
	return err
}

// AtLineStart returns true if the last write ended with a line break, so that
// the next write starts a new line.
func (w *writer) AtLineStart() bool {
	return w.lastWasLineBreak
}

// IndentationFrom gets the indentation of the given level and of the levels
// after it.
func (w *writer) IndentationFrom(level int) string {
	if level >= len(w.indentLengths) {
		return ""
	}

	return string(w.currentIndent[w.indentLengths[level]:])
}

func (w *writer) OnlyIndent() []byte {
	return w.currentIndent[:w.trailingSpaceIndex]
}

func (w *writer) TrailingWhitespace() []byte {
	return w.currentIndent[w.trailingSpaceIndex:]
}

func (w *writer) IndentationLevel() int {
	return len(w.indentLengths)
}

func (w *writer) PushIndentation(v string) {
	w.indentLengths = append(w.indentLengths, len(w.currentIndent))
	w.currentIndent = append(w.currentIndent, []byte(v)...)
	w.updateTrailingSpaceIndex()
	w.didWrite = append(w.didWrite, false)
}

func (w *writer) PopIndentation() string {
	last := w.indentLengths[len(w.indentLengths)-1]
	lastIndent := w.currentIndent[last:]

	w.indentLengths = w.indentLengths[:len(w.indentLengths)-1]
	w.currentIndent = w.currentIndent[:last]
	w.updateTrailingSpaceIndex()

	// Pop the last didWrite value and propagate it upwards.
	didWrite := w.didWrite[len(w.didWrite)-1]
	w.didWrite = w.didWrite[:len(w.didWrite)-1]

	if didWrite {
		w.didWrite[len(w.didWrite)-1] = didWrite
	}

	return string(lastIndent)
}

func (w *writer) updateTrailingSpaceIndex() {
	for i := len(w.currentIndent) - 1; i >= 0; i-- {
		if w.currentIndent[i] != ' ' && w.currentIndent[i] != '\t' {
			w.trailingSpaceIndex = i + 1
			return
		}
	}

	w.trailingSpaceIndex = 0
}

func (w *writer) HasWrittenAtCurrentIndent() bool {
	return w.didWrite[len(w.didWrite)-1]
}

func (w *writer) WriteString(v string) error {
	if len(v) == 0 {
		return nil
	}

	w.didWrite[len(w.didWrite)-1] = true

	asBytes := []byte(v)
	lastWrite := 0
	for i, c := range v {
		if w.lastWasLineBreak && c != '\n' && w.lazyLevel >= 0 {
			// The line is written with the indentation before the skipped
			// level only.
			level := w.lazyLevel
			w.lazyLevel = -1

			if level < len(w.indentLengths) {
				_, err := w.output.Write(w.currentIndent[:w.indentLengths[level]])
				if err != nil {
					return err
				}

				w.lastWasLineBreak = false
				continue
			}
		}

		if w.lastWasLineBreak {
			// The previous write was a a line break, so we need to write the
			// indentation.
			_, err := w.output.Write(w.OnlyIndent())
			if err != nil {
				return err
			}
		}

		if c == '\n' {
			// TODO: Do we need to keep track of list bullets here and write them in some cases?

			var err error
			if w.lineEnding == "" {
				_, err = w.output.Write(asBytes[lastWrite : i+1])
			} else {
				_, err = w.output.Write(asBytes[lastWrite:i])
				if err == nil {
					_, err = io.WriteString(w.output, w.lineEnding)
				}
			}
			if err != nil {
				return err
			}

			w.lastWasLineBreak = true
			lastWrite = i + 1
		} else {
			// This is not a line break, but we may need to write trailing
			// whitespace.
			if w.lastWasLineBreak {
				_, err := w.output.Write(w.TrailingWhitespace())
				if err != nil {
					return err
				}
			}

			w.lastWasLineBreak = false
		}
	}

	if lastWrite < len(asBytes) {
		_, err := w.output.Write(asBytes[lastWrite:])
		if err != nil {
			return err
		}
	}

	return nil
}
