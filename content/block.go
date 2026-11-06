package content

import (
	"strings"

	"github.com/google/uuid"
)

// Block is a piece of information in an outline, either belonging to a page
// or another block.
type Block struct {
	baseNodeWithChildren
	previousLineAwareImpl

	properties   *Properties
	preBlock     bool
	headingBlock bool
}

func NewBlock(children ...Node) *Block {
	block := &Block{}
	block.self = block
	block.childValidator = allowOnlyBlockNodes
	for _, child := range AddAutomaticParagraphs(children) {
		block.AddChild(child)
	}
	return block
}

// NewPreBlock creates a block that holds the content before the first bullet
// of a page, which is what Logseq calls a pre-block. Page properties live
// there, and the block is written without a bullet marker.
//
// A pre-block is only written that way when it is the first block of a page,
// as that is the only place the bullet-less form parses back the same way.
func NewPreBlock(children ...Node) *Block {
	block := NewBlock(children...)
	block.preBlock = true
	return block
}

// IsPreBlock returns true if this block holds the content before the first
// bullet of a page. See NewPreBlock for details.
func (b *Block) IsPreBlock() bool {
	return b.preBlock
}

// SetPreBlock sets whether this block holds the content before the first
// bullet of a page. See NewPreBlock for details.
func (b *Block) SetPreBlock(preBlock bool) {
	b.preBlock = preBlock
}

// NewHeadingBlock creates a block that starts with a heading line that is
// not on a bullet line, such as `# Title`. Logseq reads each such line as the
// start of a block, and the text and properties after it are part of that
// block:
//
//	# Title
//	type:: book
//	- The next block
//
// A heading line after the first line of a block starts a child block of
// that block. In `- Notes\n  ## Section\n  - Item`, the block `Notes` has two
// child blocks: the heading block `## Section` and the block `Item`.
//
// The block is written without a bullet marker while it starts with a
// heading, and is a top-level block or a child of a block that is written
// with a bullet and has content. Otherwise it is written as a usual block.
func NewHeadingBlock(children ...Node) *Block {
	block := NewBlock(children...)
	block.headingBlock = true
	return block
}

// IsHeadingBlock returns true if this block was read from a heading line that
// is not on a bullet line. See NewHeadingBlock for details.
func (b *Block) IsHeadingBlock() bool {
	return b.headingBlock
}

// Content gets the content part of this block, which is all children that
// are not blocks.
func (b *Block) Content() NodeList {
	return b.Children().Filter(func(node Node) bool {
		_, ok := node.(*Block)
		return !ok
	})
}

// Blocks gets all of the children that are blocks.
func (b *Block) Blocks() BlockList {
	blocks := make(BlockList, 0)
	for node := b.FirstChild(); node != nil; node = node.NextSibling() {
		if block, ok := node.(*Block); ok {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// ID gets the identifier of the block. If the block does not have an ID this
// will return an empty string. This never changes the block, use WithID to
// give the block an ID.
//
// The ID is read in the same way as Logseq reads it: from `custom-id` or
// `custom_id` before `id`, and only if the value is a uuid such as
// `6500e8a3-0000-4000-8000-000000000000`. Logseq gives a block with an id
// that is not a uuid a new random id, so such an id is not returned.
func (b *Block) ID() string {
	p := b.FindProperties()
	if p == nil {
		return ""
	}

	for _, name := range []string{"custom-id", "id"} {
		id := p.GetAsNode(name)
		if id == nil {
			continue
		}

		if child, ok := id.FirstChild().(*Text); ok && isUUID(child.Value) {
			return strings.TrimSpace(child.Value)
		}
	}

	return ""
}

// isUUID checks if a value is a uuid in the form that Logseq accepts as the
// id of a block: 32 hexadecimal digits in groups of 8-4-4-4-12, with spaces
// around them allowed.
func isUUID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 36 {
		return false
	}

	_, err := uuid.Parse(value)
	return err == nil
}

// WithID ensures that the block has an ID. If the block already has an ID this
// will do nothing. If the block has an `id` property that is not a uuid, its
// value is replaced with a new uuid, as Logseq does not use such an id.
func (b *Block) WithID() *Block {
	if b.ID() != "" {
		return b
	}

	b.Properties().Set("id", NewText(uuid.NewString()))
	return b
}

// Properties gets the properties node for this block. This follows the Logseq
// implementation where the properties that belong to a block are the ones that
// appear before any of its sub blocks.
//
// If such properties do not exist, they are created to allow for easy
// manipulation of properties. Use FindProperties to look up properties without
// changing the block.
func (b *Block) Properties() *Properties {
	if b.properties == nil {
		b.properties = b.FindProperties()
	}

	if b.properties == nil {
		properties := NewProperties()

		// Logseq writes the properties of a block on the lines that follow the
		// line its content starts on, so they go after the first thing in the
		// block that is not a block of its own.
		first := b.FirstChild()
		if first == nil {
			b.AddChild(properties)
		} else if _, ok := first.(*Block); ok {
			b.InsertChildBefore(properties, first)
		} else {
			b.InsertChildAfter(properties, first)
		}

		b.properties = properties
	}

	return b.properties
}

// FindProperties gets the properties node for this block, returning nil if the
// block does not have any. Unlike Properties this never changes the block.
func (b *Block) FindProperties() *Properties {
	if b.properties != nil {
		return b.properties
	}

	for node := b.FirstChild(); node != nil; node = node.NextSibling() {
		switch n := node.(type) {
		case *Properties:
			return n
		case *FrontMatter:
			// Logseq reads front matter as the properties of the page. Front
			// matter without properties is skipped, so that properties after
			// it are found.
			if properties := n.FindProperties(); properties != nil {
				return properties
			}
		case *Block:
			// Anything after a sub block belongs to that part of the outline
			// and not to this block.
			return nil
		}
	}

	return nil
}

// WithPreviousLineType sets the type of the line before the bullet of the
// block and returns the block. Only PreviousLineTypeBlank changes the output,
// as it keeps a blank line between the block and the content before it.
func (b *Block) WithPreviousLineType(t PreviousLineType) *Block {
	b.previousLineType = t
	return b
}

func (n *Block) debug(p *debugPrinter) {
	p.StartType("Block")
	debugPreviousLineAware(p, n)
	p.Children(n)
	p.EndType()
}

func (n *Block) GomegaString() string {
	printer := newDebugPrinter()
	n.debug(printer)
	return printer.String()
}

func (n *Block) isBlock() {}

var _ Node = (*Block)(nil)
var _ HasChildren = (*Block)(nil)
var _ BlockNode = (*Block)(nil)
var _ PreviousLineAware = (*Block)(nil)

type BlockList []*Block

func (l BlockList) Find(predicate func(block *Block) bool) *Block {
	for _, block := range l {
		if predicate(block) {
			return block
		}
	}

	return nil
}

func (l BlockList) FindDeep(predicate func(block *Block) bool) *Block {
	for _, block := range l {
		if predicate(block) {
			return block
		}

		if found := block.Blocks().FindDeep(predicate); found != nil {
			return found
		}
	}

	return nil
}

func (l BlockList) Filter(predicate func(block *Block) bool) BlockList {
	filtered := make(BlockList, 0)
	for _, block := range l {
		if predicate(block) {
			filtered = append(filtered, block)
		}
	}

	return filtered
}

func (l BlockList) FilterDeep(predicate func(block *Block) bool) BlockList {
	filtered := make(BlockList, 0)
	for _, block := range l {
		if predicate(block) {
			filtered = append(filtered, block)
		}

		filtered = append(filtered, block.Blocks().FilterDeep(predicate)...)
	}

	return filtered
}
