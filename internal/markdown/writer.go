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
}

func newWriter(out io.Writer) *writer {
	return &writer{
		output:           out,
		lastWasLineBreak: true,
		didWrite:         []bool{false},
	}
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

			_, err := w.output.Write(asBytes[lastWrite : i+1])
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
