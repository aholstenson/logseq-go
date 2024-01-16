package logseq

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
)

// ParseBlock parses markdown into a block.
func ParseBlock(text string) (*content.Block, error) {
	return markdown.ParseString(text)
}

// ParseNodes parses markdown into a list of nodes.
func ParseNodes(text string) (content.NodeList, error) {
	block, err := ParseBlock(text)
	if err != nil {
		return nil, err
	}

	return block.Children(), nil
}
