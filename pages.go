package logseq

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	"github.com/aholstenson/logseq-go/internal/utils"
)

type PageType int

const (
	PageTypeDedicated PageType = iota
	PageTypeJournal
)

// ErrPageNotFound is returned for operations that need a page to exist in the
// graph, such as renaming one.
var ErrPageNotFound = errors.New("page not found")

// ErrPageExists is returned when a page would overwrite another page, such as
// when renaming a page to a title that is already taken.
var ErrPageExists = errors.New("page already exists")

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

	// Aliases returns the alternative titles of the page, as declared by its
	// `alias` property. In a graph with indexing enabled, opening a page by one
	// of its aliases opens the page itself.
	Aliases() []string

	// Namespace returns the namespace the page is in, which is the part of its
	// title before the last `/`. Pages that are not in a namespace, and
	// journals, have no namespace and return an empty string.
	Namespace() string

	// NamespaceChildren finds the pages that are directly below this page in
	// the namespace hierarchy, so `Parent/Child` for a page titled `Parent`.
	// Pages deeper in the namespace, such as `Parent/Child/Grandchild`, are
	// children of the page between them and this one.
	//
	// Search options such as WithMaxHits and FromHit can be used to page through
	// the children, and WithQuery narrows them down further.
	//
	// Children are found via the index, so this requires the graph to have been
	// opened with indexing enabled.
	NamespaceChildren(ctx context.Context, opts ...SearchOption) (SearchResults[PageResult], error)

	// Blocks returns the blocks for the page. If the page has content before
	// its first bullet, such as page properties, that content is the first
	// block and reports true from Block.IsPreBlock.
	Blocks() content.BlockList

	// LinkedReferences finds the blocks in the graph that reference this page,
	// which is what Logseq shows as the linked references of a page. Blocks
	// reference a page via `[[Title]]`, `#Title`, `{{embed [[Title]]}}` or a
	// property such as `related:: [[Title]]`, and blocks on the page itself are
	// included if they do.
	//
	// Search options such as WithMaxHits and FromHit can be used to page through
	// the references, and WithQuery narrows them down further.
	//
	// References are found via the index, so this requires the graph to have
	// been opened with indexing enabled.
	LinkedReferences(ctx context.Context, opts ...SearchOption) (SearchResults[BlockResult], error)

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
	// source is where the page was opened from, used to look up other content
	// in the graph. Pages opened in a transaction get the transaction, so that
	// pages reached from this one become part of it as well.
	source pageSource

	path         string
	isNew        bool
	lastModified time.Time

	pageType PageType
	title    string
	date     time.Time

	root *content.Block
}

func openOrCreatePage(source pageSource, path string, pageType PageType, title string, date time.Time, templatePath string, parseOptions ...markdown.ParseOption) (*pageImpl, error) {
	// Get the last modified time for the file
	info, err := os.Stat(path)
	var root *content.Block
	if os.IsNotExist(err) {
		// This page does not exist, let's try to load the template
		if templatePath == "" {
			// No template, start with an empty page
			root = content.NewBlock()
		} else {
			root, err = loadRootBlock(templatePath, parseOptions...)
			if err != nil {
				return nil, fmt.Errorf("failed to load template: %w", err)
			}
		}
	} else if err != nil {
		// Other type of error, return it
		return nil, err
	} else {
		// This page exists, load it
		root, err = loadRootBlock(path, parseOptions...)
		if err != nil {
			return nil, fmt.Errorf("failed to load page: %w", err)
		}
	}

	lastModified := time.Now()
	if info != nil {
		lastModified = info.ModTime()
	}

	return &pageImpl{
		source: source,

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

	// The page has no properties yet, create them at the start of the pre-block
	// so they are written the way Logseq writes them: as the first lines of the
	// page, before the first bullet.
	properties := content.NewProperties()
	p.preBlock().PrependChild(properties)
	return properties
}

func (p *pageImpl) Namespace() string {
	if p.pageType != PageTypeDedicated {
		// The title of a journal is a formatted date, which can contain slashes
		// without any of it being a namespace.
		return ""
	}

	return utils.NamespaceOf(p.title)
}

func (p *pageImpl) NamespaceChildren(ctx context.Context, opts ...SearchOption) (SearchResults[PageResult], error) {
	options := make([]SearchOption, 0, len(opts)+1)
	options = append(options, WithQuery(InNamespace(p.title)))
	options = append(options, opts...)

	return p.source.SearchPages(ctx, options...)
}

func (p *pageImpl) Aliases() []string {
	properties := p.findProperties()
	if properties == nil {
		return nil
	}

	return propertyTitles(properties.GetAsNode("alias"))
}

// propertyTitles reads the page titles that are the value of a property, such
// as `alias:: [[Example]], Another`. Page references are taken as they are,
// while the rest of the value is split on commas, which is how Logseq writes
// properties that hold several values.
func propertyTitles(property *content.Property) []string {
	if property == nil {
		return nil
	}

	titles := make([]string, 0)

	var text strings.Builder
	takeText := func() {
		for _, part := range strings.Split(text.String(), ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				titles = append(titles, part)
			}
		}

		text.Reset()
	}

	for _, node := range property.Children() {
		switch n := node.(type) {
		case content.PageRef:
			takeText()
			titles = append(titles, n.GetTo())
		case *content.Text:
			text.WriteString(n.Value)
		}
	}

	takeText()

	if len(titles) == 0 {
		return nil
	}

	return titles
}

// findProperties locates the properties of the page, returning nil if the page
// does not have any.
func (p *pageImpl) findProperties() *content.Properties {
	// Logseq treats properties in the first block of a page as properties of
	// the page, whether that block is the pre-block or a bullet.
	if first, ok := p.root.FirstChild().(*content.Block); ok {
		return first.FindProperties()
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

func (p *pageImpl) LinkedReferences(ctx context.Context, opts ...SearchOption) (SearchResults[BlockResult], error) {
	options := make([]SearchOption, 0, len(opts)+1)
	options = append(options, WithQuery(References(p.title)))
	options = append(options, opts...)

	return p.source.SearchBlocks(ctx, options...)
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

func loadRootBlock(path string, parseOptions ...markdown.ParseOption) (*content.Block, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, err := markdown.Parse(data, parseOptions...)
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
