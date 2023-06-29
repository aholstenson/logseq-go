package logseq

import (
	"os"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
)

type Page struct {
	path  string
	root  *content.Block
	isNew bool
}

// IsNew returns true if the page is new and wasn't loaded from disk.
func (p *Page) IsNew() bool {
	return p.isNew
}

func (p *Page) Properties() *content.Properties {
	blocks := p.root.Blocks()
	if len(blocks) == 0 {
		block := content.NewBlock()
		p.root.AddChild(block)
		return block.Properties()
	}

	return blocks[0].Properties()
}

func (p *Page) Blocks() content.BlockList {
	return p.root.Blocks()
}

func (p *Page) AddBlock(block *content.Block) {
	p.root.AddChild(block)
}

func (p *Page) RemoveBlock(block *content.Block) {
	p.root.RemoveChild(block)
}

func (p *Page) Save() error {
	data, err := markdown.AsString(p.root)
	if err != nil {
		return err
	}

	err = os.WriteFile(p.path, []byte(data), 0644)
	if err != nil {
		return err
	}

	p.isNew = false
	return nil
}
