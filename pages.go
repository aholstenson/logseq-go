package logseq

import (
	"fmt"
	"os"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
)

type DocumentType int

const (
	DocumentTypePage DocumentType = iota
	DocumentTypeJournal
)

type Document interface {
	// Type returns the type of the document.
	Type() DocumentType

	// IsNew returns true if the page is new and wasn't loaded from disk.
	IsNew() bool

	// Title returns the title for the document.
	Title() string

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

type documentImpl struct {
	path         string
	root         *content.Block
	isNew        bool
	lastModified time.Time
}

func openOrCreateDocument(path string, templatePath string) (*documentImpl, error) {
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

	return &documentImpl{
		path:         path,
		root:         root,
		isNew:        info == nil,
		lastModified: lastModified,
	}, nil
}

func (p *documentImpl) IsNew() bool {
	return p.isNew
}

func (p *documentImpl) LastModified() time.Time {
	return p.lastModified
}

func (p *documentImpl) Properties() *content.Properties {
	blocks := p.root.Blocks()
	if len(blocks) == 0 {
		block := content.NewBlock()
		p.root.AddChild(block)
		return block.Properties()
	}

	return blocks[0].Properties()
}

func (p *documentImpl) Blocks() content.BlockList {
	return p.root.Blocks()
}

func (p *documentImpl) AddBlock(block *content.Block) {
	p.root.AddChild(block)
}

func (p *documentImpl) RemoveBlock(block *content.Block) {
	p.root.RemoveChild(block)
}

func (p *documentImpl) PrependBlock(block *content.Block) {
	p.root.PrependChild(block)
}

func (p *documentImpl) InsertBlockAfter(block *content.Block, after *content.Block) {
	p.root.InsertChildAfter(block, after)
}

func (p *documentImpl) InsertBlockBefore(block *content.Block, before *content.Block) {
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

type Journal struct {
	documentImpl

	title string
	date  time.Time
}

func (p *Journal) Type() DocumentType {
	return DocumentTypeJournal
}

func (p *Journal) Title() string {
	return p.title
}

// Date gets the date for the journal page.
func (p *Journal) Date() time.Time {
	return p.date
}

var _ Document = &Journal{}

type Page struct {
	documentImpl

	title string
}

func (p *Page) Type() DocumentType {
	return DocumentTypePage
}

func (p *Page) Title() string {
	return p.title
}

var _ Document = &Page{}
