package indexing

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/utils"
)

// propertyPages decides which property names link a block to the page with
// the same name. Logseq links a block with `type:: book` to the page `type`,
// unless the graph turns this off with `:property-pages/enabled?` or lists the
// property in `:property-pages/excludelist`. Built-in properties, such as
// `alias` or `collapsed`, never link to a page.
type propertyPages struct {
	disabled bool
	excluded map[string]struct{}
}

func newPropertyPages(graphConfig *utils.GraphConfig) propertyPages {
	pages := propertyPages{
		excluded: make(map[string]struct{}),
	}

	if graphConfig == nil {
		return pages
	}

	if graphConfig.PropertyPagesEnabled != nil && !*graphConfig.PropertyPagesEnabled {
		pages.disabled = true
	}

	for _, name := range graphConfig.PropertyPagesExcludelist {
		pages.excluded[propertyPageTitle(name)] = struct{}{}
	}

	return pages
}

// titlesOf returns the titles of the pages that the names of the properties
// link to, normalized with normalizeRef.
func (p propertyPages) titlesOf(properties *content.Properties) []string {
	if p.disabled || properties == nil {
		return nil
	}

	titles := make([]string, 0)
	for _, node := range properties.Children() {
		property, ok := node.(*content.Property)
		if !ok || utils.IsBuiltInProperty(property.Name) {
			continue
		}

		title := propertyPageTitle(property.Name)
		if title == "" {
			continue
		}

		if _, excluded := p.excluded[title]; excluded {
			continue
		}

		if !containsString(titles, title) {
			titles = append(titles, title)
		}
	}

	return titles
}

// propertyPageTitle is the title of the page that a property name links to.
// Logseq uses the name of the property as a keyword, so for a name in a
// namespace, such as `ns/key`, only the part after the slash is used. A name
// with more than one slash is not split into a namespace and a name.
func propertyPageTitle(name string) string {
	name = content.NormalizePropertyName(strings.TrimPrefix(name, ":"))
	if parts := strings.Split(name, "/"); len(parts) == 2 {
		name = parts[1]
	}

	return normalizeRef(name)
}
