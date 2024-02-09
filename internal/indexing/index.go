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

	// GetLastModified returns the last modified time for a page. Should return
	// a zero time if the page does not exist in the index.
	GetLastModified(ctx context.Context, subPath string) (time.Time, error)

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
	Field string
	Asc   bool
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

	// Title is the title of the page. Only used for dedicated pages.
	Title string
	// Date is the date of the journal. Only used for journals.
	Date time.Time

	// Preview string of the page, only used when searching.
	Preview string

	// Blocks is the blocks of the page, only used while indexing.
	Blocks content.BlockList
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
}
