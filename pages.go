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

	// Properties returns the properties for the page. Properties belong to the
	// first block of the page, which is the pre-block if the page has one.
	Properties() *content.Properties

	// Blocks returns the blocks for the page. If the page has content before
	// its first bullet, such as page properties, that content is the first
	// block and reports true from Block.IsPreBlock.
	Blocks() content.BlockList

	// AddBlock adds a block to the page.
	AddBlock(block *content.Block)

	// RemoveBlock removes a block from the page.
	RemoveBlock(block *content.Block)

	// PrependBlock adds a block to the start of the page, after the pre-block
	// if the page has one.
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

	// The page has no properties yet, create them in the pre-block so they are
	// written the way Logseq writes them: before the first bullet of the page.
	return p.preBlock().Properties()
}

// findProperties locates the properties of the page, returning nil if the page
// does not have any.
func (p *pageImpl) findProperties() *content.Properties {
	// Logseq treats properties in the first block of a page as properties of
	// the page, whether that block is the pre-block or a bullet.
	if first, ok := p.root.FirstChild().(*content.Block); ok {
		if properties, ok := first.FirstChild().(*content.Properties); ok {
			return properties
		}
	}

	return nil
}

// preBlock returns the block holding the content before the first bullet of
// the page, adding an empty one to the start of the page if it does not have
// one yet.
func (p *pageImpl) preBlock() *content.Block {
	if first, ok := p.root.FirstChild().(*content.Block); ok && first.IsPreBlock() {
		return first
	}

	preBlock := content.NewPreBlock()
	p.root.PrependChild(preBlock)
	return preBlock
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
	if first, ok := p.root.FirstChild().(*content.Block); ok && first.IsPreBlock() {
		// The pre-block holds the content before the first bullet, so the new
		// block goes after it to keep it at the start of the page.
		p.root.InsertChildAfter(block, first)
		return
	}

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

	// Content that appears before the first bullet becomes the pre-block of the
	// page, matching how Logseq models it. It stays the first block of the page
	// and is written back without a bullet, so page properties and any prose
	// around them round-trip in the same shape.
	var leading []content.Node
	for node := block.FirstChild(); node != nil; node = node.NextSibling() {
		if _, ok := node.(*content.Block); ok {
			break
		}

		leading = append(leading, node)
	}

	if len(leading) > 0 {
		preBlock := content.NewPreBlock()
		for _, node := range leading {
			preBlock.AddChild(node)
		}

		block.PrependChild(preBlock)
	}

	return block, nil
}
