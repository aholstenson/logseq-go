package logseq

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
	"github.com/aholstenson/logseq-go/internal/markdown"
	"github.com/aholstenson/logseq-go/internal/utils"
)

// ErrTemplateNotFound is returned when no block in the graph is a template
// with the requested name.
var ErrTemplateNotFound = errors.New("template not found")

// TemplateOption is an option for filling in a template.
type TemplateOption func(*templateOptions)

type templateOptions struct {
	time time.Time
	page string
}

// WithTemplateTime sets the time that the template is inserted at. Dates such
// as `<% today %>` and `<% next friday %>` are relative to the day of this
// time, and `<% time %>` is its clock time. The default is the current time.
func WithTemplateTime(t time.Time) TemplateOption {
	return func(o *templateOptions) {
		o.time = t
	}
}

// WithTemplatePage sets the title of the page that the template is inserted
// into, which is what `<% current page %>` links to. The default is the
// journal of the day the template is inserted at.
func WithTemplatePage(title string) TemplateOption {
	return func(o *templateOptions) {
		o.page = title
	}
}

// templateVariable matches a variable in a template, such as `<% today %>`.
// This is the same pattern Logseq uses.
var templateVariable = regexp.MustCompile(`<%([^%].*?)%>`)

// Template returns a copy of the blocks of the template with the given name,
// ready to be added to a page. In Logseq a template is a block with the
// property `template:: Name`, and the name is matched without regard for case.
//
// The blocks are copied the way Logseq inserts a template:
//
//   - The block with the `template` property is included, unless it has
//     `template-including-parent:: false`. Then only its child blocks are
//     included.
//   - The properties `template`, `template-including-parent` and `id` are
//     removed from all copied blocks, so that the copies are not templates of
//     their own and do not share an id with the original blocks.
//   - Variables are filled in. `<% today %>`, `<% yesterday %>` and
//     `<% tomorrow %>` become links to journals, `<% time %>` becomes the time
//     as `15:04` and `<% current page %>` becomes a link to the page. Dates in
//     words, such as `<% next friday %>`, also become links to journals. Any
//     other variable becomes its text, so `<% unknown %>` becomes `unknown`.
//
// Use WithTemplateTime and WithTemplatePage to set what the variables are
// filled in with.
//
// Templates are found via the index, so this requires the graph to have been
// opened with indexing enabled. If no template has the name,
// ErrTemplateNotFound is returned.
func (g *Graph) Template(ctx context.Context, name string, opts ...TemplateOption) (content.BlockList, error) {
	options := &templateOptions{
		time: time.Now(),
	}
	for _, opt := range opts {
		opt(options)
	}

	template, err := g.findTemplate(ctx, name)
	if err != nil {
		return nil, err
	}

	return g.expandTemplate(template, options)
}

// findTemplate opens the block that is the template with the given name.
func (g *Graph) findTemplate(ctx context.Context, name string) (*content.Block, error) {
	if g.index == nil {
		return nil, fmt.Errorf("indexing is not enabled")
	}

	name = strings.ToLower(strings.TrimSpace(name))

	var found BlockResult
	err := eachResult(func(opts indexing.SearchOptions) (indexing.SearchResults[BlockResult], error) {
		return g.searchBlocks(ctx, []SearchOption{
			WithQuery(HasProperty("template")),
			WithMaxHits(opts.Size),
			FromHit(opts.From),
		}, g)
	}, func(result BlockResult) {
		if found == nil && strings.ToLower(strings.TrimSpace(result.Properties()["template"])) == name {
			found = result
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to look up the template %s: %w", name, err)
	}

	if found == nil {
		return nil, ErrTemplateNotFound
	}

	block, _, err := found.Open()
	if errors.Is(err, ErrBlockNotFound) {
		// The index is behind the page the template was found on.
		return nil, ErrTemplateNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to open the template %s: %w", name, err)
	}

	return block, nil
}

// expandTemplate copies the blocks of a template and fills in its variables.
// The template block is taken from the page it was opened on, so the page must
// not be used afterwards.
func (g *Graph) expandTemplate(template *content.Block, options *templateOptions) (content.BlockList, error) {
	blocks := content.BlockList{template}
	if !templateIncludesParent(template) {
		blocks = template.Blocks()
	}

	// The blocks are copied by writing them as Markdown and reading them back.
	// Logseq fills in the variables of a template in the Markdown of each
	// block, so that is also where they are filled in here.
	root := content.NewBlock()
	for _, block := range blocks {
		root.AddChild(block)
	}

	text, err := markdown.AsString(root, g.markdownOptions()...)
	if err != nil {
		return nil, fmt.Errorf("failed to write the template: %w", err)
	}

	text = g.fillTemplateVariables(text, options)

	copied, err := markdown.ParseString(text, g.markdownParseOptions()...)
	if err != nil {
		return nil, fmt.Errorf("failed to read the template: %w", err)
	}

	result := copied.Blocks()
	for _, block := range result {
		removeTemplateProperties(block)
	}

	return result, nil
}

// templateIncludesParent checks if the block with the `template` property is
// part of what the template inserts, which it is unless it has the property
// `template-including-parent:: false`.
func templateIncludesParent(template *content.Block) bool {
	props := template.FindProperties()
	if props == nil {
		return true
	}

	value := strings.TrimSpace(textOf(props.Get("template-including-parent")))
	return !strings.EqualFold(value, "false")
}

// textOf returns the text of nodes, such as the value of a property.
func textOf(nodes content.NodeList) string {
	var text strings.Builder
	for _, node := range nodes {
		if t, ok := node.(*content.Text); ok {
			text.WriteString(t.Value)
		}
	}

	return text.String()
}

// removeTemplateProperties removes the properties that a copy of a template
// must not keep from a block and all blocks below it.
func removeTemplateProperties(block *content.Block) {
	if props := block.FindProperties(); props != nil {
		props.Remove("template")
		props.Remove("template-including-parent")
		props.Remove("id")

		if props.FirstChild() == nil {
			block.RemoveChild(props)
		}
	}

	for _, child := range block.Blocks() {
		removeTemplateProperties(child)
	}
}

// fillTemplateVariables replaces the variables in the Markdown of a template
// with their values, the same way Logseq does it.
func (g *Graph) fillTemplateVariables(text string, options *templateOptions) string {
	day := journalDate(options.time)

	page := options.page
	if page == "" {
		page = g.journalTitleFormat.Format(day)
	}

	return templateVariable.ReplaceAllStringFunc(text, func(match string) string {
		variable := strings.TrimSpace(templateVariable.FindStringSubmatch(match)[1])

		switch strings.ToLower(variable) {
		case "":
			return ""
		case "time":
			return options.time.Local().Format("15:04")
		case "current page":
			return pageLink(page)
		}

		if date, ok := utils.ParseRelativeDate(variable, day); ok {
			return pageLink(g.journalTitleFormat.Format(date))
		}

		return variable
	})
}

// pageLink writes a link to a page as Markdown, such as `[[Title]]`.
func pageLink(title string) string {
	return "[[" + markdown.EscapeWikiLink(title) + "]]"
}
