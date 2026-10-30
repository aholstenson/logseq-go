package indexing

import (
	"context"
	"time"

	"github.com/aholstenson/logseq-go/content"
)

type Index interface {
	// Close this index.
	Close() error

	// Sync makes sure the index has been synced and any changes are queryable.
	// Syncing may involve writing to disk.
	Sync() error

	// DeletePage removes a page from the index.
	DeletePage(ctx context.Context, subPath string) error

	// IndexPage indexes a page in the search index.
	IndexPage(ctx context.Context, doc *Page) error

	// GetFileState returns what the index knows about the file of a page.
	// Should return a zero FileState if the page does not exist in the index.
	GetFileState(ctx context.Context, subPath string) (FileState, error)

	// PageSubPaths returns the sub paths of all pages and journals in the
	// index.
	PageSubPaths(ctx context.Context) ([]string, error)

	// SearchPages searches for pages in the index.
	SearchPages(ctx context.Context, query Query, opts SearchOptions) (SearchResults[*Page], error)

	// SearchBlocks searches for blocks in the index.
	SearchBlocks(ctx context.Context, query Query, opts SearchOptions) (SearchResults[*Block], error)
}

type SearchOptions struct {
	// Size is the number of results to return.
	Size int

	// From is the offset to start returning results from.
	From int

	// SortBy is the sort order for the results.
	SortBy []SortField
}

type SortField struct {
	// Field is the field to sort by, one of the SortField constants.
	Field string
	Asc   bool
}

// The fields that results can be sorted by. Results without a value for the
// field come last in both orders.
const (
	// SortFieldPriority sorts tasks by priority, as A, B and C.
	SortFieldPriority = "priority"
	// SortFieldScheduled sorts blocks by their `SCHEDULED` date.
	SortFieldScheduled = "scheduled"
	// SortFieldDeadline sorts blocks by their `DEADLINE` date.
	SortFieldDeadline = "deadline"
	// SortFieldJournalDate sorts journals, and the blocks on them, by the date
	// of the journal.
	SortFieldJournalDate = "date"
	// SortFieldTitle sorts pages by title, and blocks by the title of their
	// page, without regard for case.
	SortFieldTitle = "title:sort"
)

// SortFieldProperty returns the field that sorts pages and blocks by the value
// of a property. Numbers sort by their value and before text, and text sorts
// without regard for case.
func SortFieldProperty(name string) string {
	return "prop:" + name + ":sort"
}

type SearchResults[V any] interface {
	// Size is the number of results available in this result set.
	Size() int

	// Count is the number of results that are available in total. For the
	// number of results available via Results, use Size.
	Count() int

	// Results is a slice of all the results in this result set.
	Results() []V
}

// FileState is what the index stores about the file of a page, which tells
// if the file changed after it was indexed.
type FileState struct {
	// LastModified is the last time the file was modified on disk.
	LastModified time.Time

	// Revision is the revision of the content of the file. It is empty for
	// pages that were indexed before the index stored revisions.
	Revision string
}

type PageType int

const (
	PageTypeDedicated PageType = iota
	PageTypeJournal
)

type Page struct {
	// SubPath is the sub path of the page in the graph.
	SubPath string

	// Type is the type of the page.
	Type PageType

	// LastModified is the last time the page was modified on disk.
	LastModified time.Time

	// Revision is the revision of the content of the page file, only used
	// while indexing.
	Revision string

	// Title is the title of the page. Only used for dedicated pages.
	Title string
	// Date is the date of the journal. Only used for journals.
	Date time.Time

	// Preview string of the page, only used when searching.
	Preview string

	// Blocks is the blocks of the page, only used while indexing.
	Blocks content.BlockList

	// Aliases is the alternative titles of the page, only used while indexing.
	Aliases []string

	// Properties is the properties of the page, nil if it has none. Only used
	// while indexing.
	Properties *content.Properties

	// PropertyValues is the plain text values of the page properties, keyed by
	// property name. Nil if the page has no properties. Only used when
	// searching.
	PropertyValues map[string]string

	// Tags is the titles of the pages in the `tags` property of the page, in
	// lower case. Only used when searching.
	Tags []string
}

type Block struct {
	// PageSubPath is the sub path of the page this block belongs to.
	PageSubPath string

	// ID is the stable id of the block if it has one.
	ID string

	// Location is the location of the block in the page.
	Location []int

	// Preview is a preview of the block.
	Preview string

	// PropertyValues is the plain text values of the block properties, keyed
	// by property name. Nil if the block has no properties. Only used when
	// searching.
	PropertyValues map[string]string
}
