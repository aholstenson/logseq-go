package query

import (
	"github.com/aholstenson/logseq-go/content"
)

// Query is a query on a graph: the filter that results must match, and how
// the results are sorted.
type Query struct {
	// Filter is what the results must match. Nil matches everything.
	Filter Filter

	// Sort is how the results are sorted. Results that are the same for the
	// first entry are sorted by the next entry.
	Sort []Sort

	// Sample is the number of random results to pick, or zero for all of
	// them.
	Sample int
}

// New creates a query with the given filter.
func New(filter Filter) *Query {
	return &Query{
		Filter: filter,
	}
}

// SortBy adds a built-in value to sort the results by. Use SortByProperty to
// sort by a property.
func (q *Query) SortBy(field SortField, order Order) *Query {
	q.Sort = append(q.Sort, Sort{
		Field: field,
		Order: order,
	})
	return q
}

// SortByProperty adds a property to sort the results by.
func (q *Query) SortByProperty(name string, order Order) *Query {
	q.Sort = append(q.Sort, Sort{
		Field:    SortProperty,
		Property: name,
		Order:    order,
	})
	return q
}

// FindsBlocks returns true if the query finds blocks, and false if it finds
// pages. The same as in Logseq, a query finds blocks if one or more of its
// filters match blocks, such as TaskFilter or PageRefFilter. It finds pages if
// all of its filters match pages, such as PagePropertyFilter, or if it has no
// filters.
func (q *Query) FindsBlocks() bool {
	return findsBlocks(q.Filter)
}

func findsBlocks(filter Filter) bool {
	switch f := filter.(type) {
	case *AndFilter:
		return anyFindsBlocks(f.Filters)
	case *OrFilter:
		return anyFindsBlocks(f.Filters)
	case *NotFilter:
		return findsBlocks(f.Filter)
	case *PageRefFilter, *RefsFilter, *FullTextFilter, *TaskFilter, *PriorityFilter,
		*PageFilter, *ParentFilter, *AncestorFilter, *PropertyFilter, *DateFilter:
		return true
	}

	return false
}

func anyFindsBlocks(filters []Filter) bool {
	for _, filter := range filters {
		if findsBlocks(filter) {
			return true
		}
	}

	return false
}

// Sort is one value to sort the results of a query by.
type Sort struct {
	// Field is what to sort by.
	Field SortField

	// Property is the name of the property to sort by when Field is
	// SortProperty.
	Property string

	// Order is the direction to sort in.
	Order Order
}

// SortField is what a query sorts by.
type SortField int

const (
	// SortPriority sorts by the priority of a task, written as `priority`.
	SortPriority SortField = iota
	// SortScheduled sorts by the `SCHEDULED` date, written as `scheduled`.
	SortScheduled
	// SortDeadline sorts by the `DEADLINE` date, written as `deadline`.
	SortDeadline
	// SortJournalDay sorts by the date of the journal a block is on, written
	// as `journal-day`.
	SortJournalDay
	// SortTitle sorts by the title of the page, written as `title`.
	SortTitle
	// SortProperty sorts by the property named in Sort.Property.
	SortProperty
)

var sortFieldNames = map[SortField]string{
	SortPriority:   "priority",
	SortScheduled:  "scheduled",
	SortDeadline:   "deadline",
	SortJournalDay: "journal-day",
	SortTitle:      "title",
}

// Order is the direction to sort in.
type Order int

const (
	// Asc sorts from the lowest value, such as the earliest date or priority
	// A, to the highest.
	Asc Order = iota
	// Desc sorts from the highest value to the lowest.
	Desc
)

// Filter is a condition that the results of a query must match. The filters
// are the types in this package with names that end with Filter.
type Filter interface {
	// String returns the filter in the text form that Parse reads.
	String() string

	isFilter()
}

// AndFilter matches when all of its filters match. It matches everything if
// it has no filters.
type AndFilter struct {
	Filters []Filter
}

// And creates a filter that matches when all of the filters match.
func And(filters ...Filter) *AndFilter {
	return &AndFilter{Filters: filters}
}

// OrFilter matches when one or more of its filters match. It matches nothing
// if it has no filters.
type OrFilter struct {
	Filters []Filter
}

// Or creates a filter that matches when one or more of the filters match.
func Or(filters ...Filter) *OrFilter {
	return &OrFilter{Filters: filters}
}

// NotFilter matches when its filter does not match.
type NotFilter struct {
	Filter Filter
}

// Not creates a filter that matches when the filter does not match.
func Not(filter Filter) *NotFilter {
	return &NotFilter{Filter: filter}
}

// TaskFilter matches tasks with one of the statuses.
type TaskFilter struct {
	Statuses []content.TaskStatus
}

// Task creates a filter that matches tasks with one of the statuses.
func Task(statuses ...content.TaskStatus) *TaskFilter {
	return &TaskFilter{Statuses: statuses}
}

// PriorityFilter matches blocks with one of the priorities.
type PriorityFilter struct {
	Priorities []content.Priority
}

// Priority creates a filter that matches blocks with one of the priorities.
func Priority(priorities ...content.Priority) *PriorityFilter {
	return &PriorityFilter{Priorities: priorities}
}

// PageRefFilter matches the blocks that are about a page: the blocks on the
// page, the blocks that reference it, and the blocks below them. Logseq calls
// these the path refs of a block. It is written as `[[page]]` or `#page`.
type PageRefFilter struct {
	Page string
}

// PageRef creates a filter that matches the blocks that are about the page.
func PageRef(page string) *PageRefFilter {
	return &PageRefFilter{Page: page}
}

// RefsFilter matches the blocks that reference a page in their own content or
// properties. Unlike PageRefFilter it does not match the blocks below them or
// the blocks on the page. Logseq calls these the refs of a block. It is written
// as `(refs [[page]])`.
type RefsFilter struct {
	Page string
}

// Refs creates a filter that matches the blocks that reference the page
// themselves.
func Refs(page string) *RefsFilter {
	return &RefsFilter{Page: page}
}

// FullTextFilter matches the blocks that contain the text. It is written as
// the text in quotes.
type FullTextFilter struct {
	Text string
}

// FullText creates a filter that matches the blocks that contain the text.
func FullText(text string) *FullTextFilter {
	return &FullTextFilter{Text: text}
}

// PageFilter matches the blocks on a page.
type PageFilter struct {
	Page string
}

// Page creates a filter that matches the blocks on the page.
func Page(page string) *PageFilter {
	return &PageFilter{Page: page}
}

// ParentFilter matches the blocks directly below a block. The block is named
// by its UUID, which is the value of its `id::` property, and is written as
// `(parent ((uuid)))`.
//
// A block without the property is named by its place instead: the title of
// its page and its location on it, written as `(parent [[page]] 0 1)`. The
// location is the position of the block among the blocks next to it at each
// level, from the top of the page, so `0 1` is the second child of the first
// block. A location changes when blocks are added or moved above the block,
// so use it right after the page is read.
type ParentFilter struct {
	// Block is the UUID of the block.
	Block string

	// Page is the title of the page of the block, for a block without a
	// UUID.
	Page string

	// Location is the place of the block on Page.
	Location []int
}

// Parent creates a filter that matches the blocks directly below the block
// with the UUID.
func Parent(block string) *ParentFilter {
	return &ParentFilter{Block: block}
}

// ParentAt creates a filter that matches the blocks directly below the block
// at the location on the page. See ParentFilter for the location.
func ParentAt(page string, location ...int) *ParentFilter {
	return &ParentFilter{Page: page, Location: location}
}

// AncestorFilter matches the blocks below a block at any depth: its children,
// the children of those and so on. The block is named the same way as for
// ParentFilter, and is written as `(ancestor ((uuid)))` or
// `(ancestor [[page]] 0 1)`.
type AncestorFilter struct {
	// Block is the UUID of the block.
	Block string

	// Page is the title of the page of the block, for a block without a
	// UUID.
	Page string

	// Location is the place of the block on Page.
	Location []int
}

// Ancestor creates a filter that matches the blocks below the block with the
// UUID at any depth.
func Ancestor(block string) *AncestorFilter {
	return &AncestorFilter{Block: block}
}

// AncestorAt creates a filter that matches the blocks below the block at the
// location on the page at any depth. See ParentFilter for the location.
func AncestorAt(page string, location ...int) *AncestorFilter {
	return &AncestorFilter{Page: page, Location: location}
}

// PropertyFilter matches the blocks with a property. If Value is set, the
// property must have that value, or reference a page with that title.
type PropertyFilter struct {
	Name  string
	Value string
}

// Property creates a filter that matches the blocks where the property has
// the value.
func Property(name string, value string) *PropertyFilter {
	return &PropertyFilter{Name: name, Value: value}
}

// HasProperty creates a filter that matches the blocks with the property,
// whatever its value is.
func HasProperty(name string) *PropertyFilter {
	return &PropertyFilter{Name: name}
}

// PagePropertyFilter matches the pages with a property. If Value is set, the
// property must have that value, or reference a page with that title.
type PagePropertyFilter struct {
	Name  string
	Value string
}

// PageProperty creates a filter that matches the pages where the property has
// the value.
func PageProperty(name string, value string) *PagePropertyFilter {
	return &PagePropertyFilter{Name: name, Value: value}
}

// HasPageProperty creates a filter that matches the pages with the property,
// whatever its value is.
func HasPageProperty(name string) *PagePropertyFilter {
	return &PagePropertyFilter{Name: name}
}

// PageTagsFilter matches the pages that have one of the tags in their `tags`
// property.
type PageTagsFilter struct {
	Tags []string
}

// PageTags creates a filter that matches the pages with one of the tags.
func PageTags(tags ...string) *PageTagsFilter {
	return &PageTagsFilter{Tags: tags}
}

// AllPageTagsFilter matches the pages that other pages use as tags.
type AllPageTagsFilter struct{}

// AllPageTags creates a filter that matches the pages that other pages use as
// tags.
func AllPageTags() *AllPageTagsFilter {
	return &AllPageTagsFilter{}
}

// NamespaceFilter matches the pages directly in a namespace, so `Parent`
// matches `Parent/Child` but not `Parent/Child/Grandchild`.
type NamespaceFilter struct {
	Namespace string
}

// Namespace creates a filter that matches the pages directly in the
// namespace.
func Namespace(namespace string) *NamespaceFilter {
	return &NamespaceFilter{Namespace: namespace}
}

func (*AndFilter) isFilter()          {}
func (*OrFilter) isFilter()           {}
func (*NotFilter) isFilter()          {}
func (*TaskFilter) isFilter()         {}
func (*PriorityFilter) isFilter()     {}
func (*PageRefFilter) isFilter()      {}
func (*RefsFilter) isFilter()         {}
func (*FullTextFilter) isFilter()     {}
func (*PageFilter) isFilter()         {}
func (*ParentFilter) isFilter()       {}
func (*AncestorFilter) isFilter()     {}
func (*PropertyFilter) isFilter()     {}
func (*PagePropertyFilter) isFilter() {}
func (*PageTagsFilter) isFilter()     {}
func (*AllPageTagsFilter) isFilter()  {}
func (*NamespaceFilter) isFilter()    {}
func (*DateFilter) isFilter()         {}
