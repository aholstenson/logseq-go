package content

import "github.com/google/uuid"

// Block is a piece of information in an outline, either belonging to a page
// or another block.
type Block struct {
	baseNodeWithChildren

	properties *Properties
}

func NewBlock(children ...Node) *Block {
	block := &Block{}
	block.self = block
	block.childValidator = allowOnlyBlockNodes
	for _, child := range addAutomaticParagraphs(children) {
		block.AddChild(child)
	}
	return block
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

func (b *Block) ID() (string, bool) {
	p := b.Properties()
	id := p.Get("id")
	if id != nil {
		return id.FirstChild().(*Text).Value, false
	}

	p.Set("id", NewText(uuid.NewString()))
	return uuid.NewString(), true
}

func (b *Block) Properties() *Properties {
	if b.properties == nil {
		// There are no properties right now, try to find them.
	_outer:
		for _, child := range b.Children() {
			if properties, ok := child.(*Properties); ok {
				b.properties = properties
				break
			} else if paragraph, ok := child.(*Paragraph); ok {
				// Look for properties in the paragraph
				for _, child := range paragraph.Children() {
					if properties, ok := child.(*Properties); ok {
						b.properties = properties
						break _outer
					}
				}
			}
		}

		if b.properties == nil {
			b.properties = NewProperties()

			if paragraph, ok := b.FirstChild().(*Paragraph); ok {
				// Insert the properties at the start of the paragraph
				paragraph.PrependChild(b.properties)
			} else {
				// The block doesn't start with a paragraph, create one
				paragraph = NewParagraph()
				b.PrependChild(paragraph)
				paragraph.AddChild(b.properties)
			}
		}
	}

	return b.properties
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
