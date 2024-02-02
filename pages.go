package logseq

import (
	"fmt"
	"os"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
)

type Page interface {
	// IsNew returns true if the page is new and wasn't loaded from disk.
	IsNew() bool

	// LastChanged returns the last time the page was changed. Use `IsNew` to
	// check if the page was loaded from disk or not.
	LastModified() time.Time

	// Properties returns the properties for the page.
	Properties() *content.Properties

	// Blocks returns the blocks for the page.
	Blocks() content.BlockList

	// AddBlock adds a block to the page.
	AddBlock(block *content.Block)

	// RemoveBlock removes a block from the page.
	RemoveBlock(block *content.Block)

	// PrependBlock adds a block to the start of the page.
	PrependBlock(block *content.Block)

	// InsertBlockAfter inserts a block after another block.
	InsertBlockAfter(block *content.Block, after *content.Block)

	// InsertBlockBefore inserts a block before another block.
	InsertBlockBefore(block *content.Block, before *content.Block)
}

type pageImpl struct {
	path         string
	root         *content.Block
	isNew        bool
	lastModified time.Time
}

func newPage(path string, templatePath string) (*pageImpl, error) {
	// Get the last modified time for the file
	info, err := os.Stat(path)
	var root *content.Block
	if os.IsNotExist(err) {
		// This page does not exist, let's try to load the template
		if templatePath == "" {
			// No template, start with an empty page
			root = content.NewBlock()
		} else {
			root, err = loadRootBlock(templatePath)
			if err != nil {
				return nil, fmt.Errorf("failed to load template: %w", err)
			}
		}
	} else if err != nil {
		// Other type of error, return it
		return nil, err
	} else {
		// This page exists, load it
		root, err = loadRootBlock(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load page: %w", err)
		}
	}

	lastModified := time.Now()
	if info != nil {
		lastModified = info.ModTime()
	}

	return &pageImpl{
		path:         path,
		root:         root,
		isNew:        info == nil,
		lastModified: lastModified,
	}, nil
}

func (p *pageImpl) IsNew() bool {
	return p.isNew
}

func (p *pageImpl) LastModified() time.Time {
	return p.lastModified
}

func (p *pageImpl) Properties() *content.Properties {
	blocks := p.root.Blocks()
	if len(blocks) == 0 {
		block := content.NewBlock()
		p.root.AddChild(block)
		return block.Properties()
	}

	return blocks[0].Properties()
}

func (p *pageImpl) Blocks() content.BlockList {
	return p.root.Blocks()
}

func (p *pageImpl) AddBlock(block *content.Block) {
	p.root.AddChild(block)
}

func (p *pageImpl) RemoveBlock(block *content.Block) {
	p.root.RemoveChild(block)
}

func (p *pageImpl) PrependBlock(block *content.Block) {
	p.root.PrependChild(block)
}

func (p *pageImpl) InsertBlockAfter(block *content.Block, after *content.Block) {
	p.root.InsertChildAfter(block, after)
}

func (p *pageImpl) InsertBlockBefore(block *content.Block, before *content.Block) {
	p.root.InsertChildBefore(block, before)
}

func loadRootBlock(path string) (*content.Block, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, err := markdown.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Check if the block has content, in which case we wrap it
	if len(block.Content()) > 0 {
		block = content.NewBlock(block)
	}

	return block, nil
}

type JournalPage struct {
	pageImpl

	date time.Time
}

// Date gets the date for the journal page.
func (p *JournalPage) Date() time.Time {
	return p.date
}

type NotePage struct {
	pageImpl

	title string
}

func (p *NotePage) Title() string {
	return p.title
}
