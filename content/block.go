package content

import "github.com/google/uuid"

// Block is a piece of information in an outline, either belonging to a page
// or another block.
type Block struct {
	baseNodeWithChildren

	properties *Properties
	preBlock   bool
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
// will return an empty string.
func (b *Block) ID() string {
	p := b.Properties()
	id := p.GetAsNode("id")
	if id != nil {
		if child, ok := id.FirstChild().(*Text); ok {
			return child.Value
		}
	}

	return ""
}

// WithID ensures that the block has an ID. If the block already has an ID this
// will do nothing.
func (b *Block) WithID() *Block {
	p := b.Properties()
	id := p.GetAsNode("id")
	if id != nil {
		return b
	}

	p.Set("id", NewText(uuid.NewString()))
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
		case *Block:
			// Anything after a sub block belongs to that part of the outline
			// and not to this block.
			return nil
		}
	}

	return nil
}

func (n *Block) debug(p *debugPrinter) {
	p.StartType("Block")
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
