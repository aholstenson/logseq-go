package logseq

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
)

// ParseBlock parses markdown into a block, read the way Logseq reads a graph
// that configures nothing. Use [Graph.ParseBlock] for markdown that belongs to
// a graph, so that the settings of that graph are used instead.
func ParseBlock(text string) (*content.Block, error) {
	return markdown.ParseString(text)
}

// ParseBlock parses markdown into a block as the settings of the graph say it
// should be read, which is the same way opening a page reads it.
func (g *Graph) ParseBlock(text string) (*content.Block, error) {
	return markdown.ParseString(text, g.markdownParseOptions()...)
}

// ParseNodes parses markdown into a list of nodes, read the way Logseq reads a
// graph that configures nothing. Use [Graph.ParseNodes] for markdown that
// belongs to a graph, so that the settings of that graph are used instead.
func ParseNodes(text string) (content.NodeList, error) {
	block, err := ParseBlock(text)
	if err != nil {
		return nil, err
	}

	return block.Children(), nil
}

// ParseNodes parses markdown into a list of nodes as the settings of the graph
// say it should be read.
func (g *Graph) ParseNodes(text string) (content.NodeList, error) {
	block, err := g.ParseBlock(text)
	if err != nil {
		return nil, err
	}

	return block.Children(), nil
}

// AsString converts a node into Markdown, written the way Logseq writes a
// graph that configures nothing. Use [Graph.AsString] for a node that belongs
// to a graph, so that the settings of that graph are used instead.
func AsString(node content.Node) (string, error) {
	return markdown.AsString(node)
}

// AsString converts a node into Markdown as the settings of the graph say it
// should be written, which is the same way saving a page writes it.
func (g *Graph) AsString(node content.Node) (string, error) {
	return markdown.AsString(node, g.markdownOptions()...)
}

// markdownOptions are the options for writing Markdown that match the settings
// of the graph.
func (g *Graph) markdownOptions() []markdown.Option {
	return []markdown.Option{
		markdown.WithLogbookSeconds(g.config.Logbook.WithSecondSupport),
	}
}

// markdownParseOptions are the options for reading Markdown that match the
// settings of the graph.
func (g *Graph) markdownParseOptions() []markdown.ParseOption {
	return []markdown.ParseOption{
		markdown.WithPropertiesSeparatedByCommas(g.config.PropertiesSeparatedByCommas...),
		markdown.WithIgnoredPageReferences(g.config.IgnoredPageReferencesKeywords...),
	}
}
