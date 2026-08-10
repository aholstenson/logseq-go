package logseq

import (
	"fmt"
	"os"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
)

type PageType int

const (
	PageTypeDedicated PageType = iota
	PageTypeJournal
)

type Page interface {
	// Type returns the type of the page.
	Type() PageType

	// IsNew returns true if the page is new and wasn't loaded from disk.
	IsNew() bool

	// Title returns the title for the page.
	Title() string

	// Date gets the date if this page is a journal. Will return the zero time if
	// the page is not a journal.
	Date() time.Time

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
	isNew        bool
	lastModified time.Time

	pageType PageType
	title    string
	date     time.Time

	root *content.Block
}

func openOrCreatePage(path string, pageType PageType, title string, date time.Time, templatePath string) (*pageImpl, error) {
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
		isNew:        info == nil,
		lastModified: lastModified,

		pageType: pageType,
		title:    title,
		date:     date,

		root: root,
	}, nil
}

func (p *pageImpl) IsNew() bool {
	return p.isNew
}

func (p *pageImpl) LastModified() time.Time {
	return p.lastModified
}

func (p *pageImpl) Type() PageType {
	return p.pageType
}

func (p *pageImpl) Title() string {
	return p.title
}

func (p *pageImpl) Date() time.Time {
	return p.date
}

func (p *pageImpl) Properties() *content.Properties {
	if properties := p.findProperties(); properties != nil {
		return properties
	}

	// The page has no properties yet, create them as content of the root
	// block so they are written the way Logseq writes them: before the first
	// bullet of the page.
	return p.root.Properties()
}

// findProperties locates the properties of the page, returning nil if the page
// does not have any.
func (p *pageImpl) findProperties() *content.Properties {
	switch first := p.root.FirstChild().(type) {
	case *content.Properties:
		// Properties before the first bullet, which is how Logseq writes them.
		return first
	case *content.Block:
		// Properties in the first block of the page are also treated as page
		// properties by Logseq.
		if properties, ok := first.FirstChild().(*content.Properties); ok {
			return properties
		}
	}

	return nil
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

	// Content that appears before the first bullet stays as content of the
	// root block, which is where Logseq puts page properties. Keeping it there
	// makes it round-trip in the same shape instead of turning the rest of the
	// page into children of it.
	//
	// A page without any bullets has no blocks to work with though, so in that
	// case the content becomes the single block of the page.
	if len(block.Blocks()) == 0 && hasContentOtherThanProperties(block) {
		block = content.NewBlock(block)
	}

	return block, nil
}

// hasContentOtherThanProperties checks if a block has content that is not
// properties.
func hasContentOtherThanProperties(block *content.Block) bool {
	for _, node := range block.Content() {
		if _, ok := node.(*content.Properties); !ok {
			return true
		}
	}

	return false
}
