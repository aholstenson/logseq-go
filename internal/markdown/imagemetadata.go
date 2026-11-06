package markdown

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// imageMetadataAttribute is the name of the attribute that holds the metadata
// of an image, such as `{:height 100, :width 200}`.
const imageMetadataAttribute = "logseq-metadata"

// imageMetadataParser reads the metadata that Logseq writes directly after an
// image when the image is resized, such as
// `![img](../assets/a.png){:height 100, :width 200}`.
//
// The metadata starts with the `{` directly after the `)` of the image and
// stops at the first `}` on the same line. It is kept on the image, so it is
// not text next to the image. A `{{` is a macro and not metadata.
type imageMetadataParser struct {
}

func (p *imageMetadataParser) Trigger() []byte {
	return []byte{'{'}
}

func (p *imageMetadataParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	// The text before this parser is added to the parent first, so the image
	// is the last child only when nothing is between it and the `{`.
	image, ok := parent.LastChild().(*ast.Image)
	if !ok || block.PrecendingCharacter() != ')' {
		return nil
	}

	line, _ := block.PeekLine()
	length := imageMetadataLength(line)
	if length == 0 {
		return nil
	}

	metadata := line[:length]
	block.Advance(length)

	// Goldmark adds the node that is returned to the parent. The image is
	// taken out and returned, so that it is added back at the same place
	// with its metadata.
	image.SetAttributeString(imageMetadataAttribute, string(metadata))
	parent.RemoveChild(parent, image)
	return image
}

var _ parser.InlineParser = (*imageMetadataParser)(nil)

// imageMetadataLength gets the length of the image metadata at the start of
// the value, including the braces. Returns 0 if the value does not start with
// image metadata.
func imageMetadataLength(value []byte) int {
	if len(value) < 2 || value[0] != '{' || value[1] == '{' {
		return 0
	}

	end := bytes.IndexAny(value, "}\n")
	if end < 0 || value[end] != '}' {
		return 0
	}

	return end + 1
}
