package query

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/aholstenson/logseq-go/content"
)

// String returns the query in the text form that Parse reads. The sort order
// and sample are written at the end of a top-level `and`, which is where
// Logseq reads them. A query that matches everything and has no sort order is
// an empty string.
func (q *Query) String() string {
	var extras []string
	for _, sort := range q.Sort {
		extras = append(extras, sort.String())
	}
	if q.Sample > 0 {
		extras = append(extras, "(sample "+strconv.Itoa(q.Sample)+")")
	}

	if len(extras) == 0 {
		if q.Filter == nil {
			return ""
		}

		return q.Filter.String()
	}

	var parts []string
	switch f := q.Filter.(type) {
	case nil:
		if len(extras) == 1 {
			return extras[0]
		}
	case *AndFilter:
		parts = filterStrings(f.Filters)
	default:
		parts = []string{f.String()}
	}

	return "(and " + strings.Join(append(parts, extras...), " ") + ")"
}

// String returns the sort in the text form that Parse reads.
func (s Sort) String() string {
	name := sortFieldNames[s.Field]
	if s.Field == SortProperty {
		name = formatValue(s.Property)
	}

	order := "asc"
	if s.Order == Desc {
		order = "desc"
	}

	return "(sort-by " + name + " " + order + ")"
}

func (f *AndFilter) String() string {
	return list("and", filterStrings(f.Filters)...)
}

func (f *OrFilter) String() string {
	return list("or", filterStrings(f.Filters)...)
}

func (f *NotFilter) String() string {
	if f.Filter == nil {
		return list("not")
	}

	return list("not", f.Filter.String())
}

func (f *TaskFilter) String() string {
	markers := make([]string, 0, len(f.Statuses))
	for _, status := range f.Statuses {
		markers = append(markers, status.String())
	}
	return list("task", markers...)
}

func (f *PriorityFilter) String() string {
	priorities := make([]string, 0, len(f.Priorities))
	for _, priority := range f.Priorities {
		priorities = append(priorities, priorityNames[priority])
	}
	return list("priority", priorities...)
}

func (f *PageRefFilter) String() string {
	if !canBePageRef(f.Page) {
		// A string on its own is a search for text, so the title is written
		// in the longer form.
		return list("page-ref", formatString(f.Page))
	}

	return formatPageRef(f.Page)
}

func (f *RefsFilter) String() string {
	return list("refs", formatPageRef(f.Page))
}

func (f *ReferencesFilter) String() string {
	name := "refs"
	if f.Path {
		name = "page-ref"
	}

	if f.Pages == nil {
		return list(name)
	}
	return list(name, f.Pages.String())
}

func (f *FullTextFilter) String() string {
	return formatString(f.Text)
}

func (f *PageFilter) String() string {
	return list("page", formatPageRef(f.Page))
}

func (f *ParentFilter) String() string {
	return blockFilterString("parent", f.Block, f.Page, f.Location)
}

func (f *AncestorFilter) String() string {
	return blockFilterString("ancestor", f.Block, f.Page, f.Location)
}

// blockFilterString writes a filter for a block, which is named by its UUID
// or by its page and location.
func blockFilterString(name string, block string, page string, location []int) string {
	if block == "" && page != "" {
		args := []string{formatPageRef(page)}
		for _, idx := range location {
			args = append(args, strconv.Itoa(idx))
		}
		return list(name, args...)
	}

	if !isBlockID(block) {
		return list(name, formatString(block))
	}

	return list(name, "(("+block+"))")
}

func (f *PropertyFilter) String() string {
	return propertyString("property", f.Name, f.Value)
}

func (f *PagePropertyFilter) String() string {
	return propertyString("page-property", f.Name, f.Value)
}

func (f *PageTagsFilter) String() string {
	tags := make([]string, 0, len(f.Tags))
	for _, tag := range f.Tags {
		tags = append(tags, formatPageRef(tag))
	}
	return list("page-tags", tags...)
}

func (f *AllPageTagsFilter) String() string {
	return list("all-page-tags")
}

func (f *NamespaceFilter) String() string {
	return list("namespace", formatPageRef(f.Namespace))
}

func (f *TitleFilter) String() string {
	return list("title", titleOpNames[f.Op], formatString(f.Text))
}

func (f *AliasFilter) String() string {
	return list("alias", formatPageRef(f.Alias))
}

func (f *HasBlockFilter) String() string {
	if f.Filter == nil {
		return list("has-block")
	}
	return list("has-block", f.Filter.String())
}

func (f *DateFilter) String() string {
	name := dateFieldNames[f.Field]

	switch f.Op {
	case Exists:
		return list(name)
	case Equal:
		return list(name, f.Date.String())
	case Range:
		if f.Field == FieldJournal {
			// Logseq reads a range of journal dates as `between`.
			name = "between"
		}
		return list(name, f.Date.String(), f.End.String())
	default:
		return list(name, opNames[f.Op], f.Date.String())
	}
}

var priorityNames = map[content.Priority]string{
	content.PriorityA: "A",
	content.PriorityB: "B",
	content.PriorityC: "C",
}

func filterStrings(filters []Filter) []string {
	parts := make([]string, 0, len(filters))
	for _, filter := range filters {
		if filter == nil {
			continue
		}
		parts = append(parts, filter.String())
	}
	return parts
}

func list(name string, args ...string) string {
	if len(args) == 0 {
		return "(" + name + ")"
	}

	return "(" + name + " " + strings.Join(args, " ") + ")"
}

func propertyString(name string, key string, value string) string {
	if value == "" {
		return list(name, formatValue(key))
	}

	return list(name, formatValue(key), formatValue(value))
}

// formatPageRef writes a page title as `[[title]]`, or as a string if it can
// not be written that way, see canBePageRef.
func formatPageRef(title string) string {
	if !canBePageRef(title) {
		return formatString(title)
	}

	return "[[" + title + "]]"
}

// canBePageRef returns false for the titles that can not be written as
// `[[title]]`, as they are blank or have brackets that would change where the
// reference ends.
func canBePageRef(title string) bool {
	return strings.TrimSpace(title) != "" && !strings.ContainsAny(title, "[]")
}

// formatValue writes a value as it is if it can be read back as a single
// symbol, and as a string otherwise.
func formatValue(value string) string {
	if value == "" || strings.ContainsAny(value, "()[]{}\",;#:<>=") || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return formatString(value)
	}

	return value
}

// formatString writes a string in quotes, with the escapes that Parse reads.
func formatString(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
