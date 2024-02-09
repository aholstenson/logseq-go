package logseq

import (
	"errors"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
)

// SearchResults is a result set from a search.
type SearchResults[R any] interface {
	// Size is the number of results available in this result set.
	Size() int

	// Count is the number of results that are available in total. For the
	// number of results available via Results, use Size.
	Count() int

	// Results is a slice of all the results in this result set.
	Results() []R
}

// SearchOption is an option for doing a search.
type SearchOption func(*searchOptions)

type searchOptions struct {
	query Query

	size int
	from int

	sortBy []indexing.SortField
}

// WithMaxHits sets the maximum number of hits to return. The default is 10.
func WithMaxHits(n int) SearchOption {
	return func(o *searchOptions) {
		o.size = n
	}
}

// FromHit sets the offset to start returning results from. This can be used
// for pagination.
func FromHit(n int) SearchOption {
	return func(o *searchOptions) {
		o.from = n
	}
}

// WithQuery sets the query to use for the search. If no query is set the
// default is to match everything. This option can be used multiple times in
// which case the queries are combined with a logical AND.
func WithQuery(q Query) SearchOption {
	return func(o *searchOptions) {
		if o.query == nil {
			o.query = q
		} else {
			o.query = And(o.query, q)
		}
	}
}

type searchResultsImpl[R any] struct {
	size    int
	count   int
	results []R
}

func (s *searchResultsImpl[R]) Size() int {
	return s.size
}

func (s *searchResultsImpl[R]) Count() int {
	return s.count
}

func (s *searchResultsImpl[R]) Results() []R {
	return s.results
}

func newSearchResults[I any, O any](r indexing.SearchResults[I], mapper func(I) O) SearchResults[O] {
	results := make([]O, len(r.Results()))
	for i, r := range r.Results() {
		results[i] = mapper(r)
	}
	return &searchResultsImpl[O]{
		size:    r.Size(),
		count:   r.Count(),
		results: results,
	}
}

type PageResult interface {
	// Type returns the type of the page.
	Type() PageType

	// Title returns the title of the page.
	Title() string

	// Date returns the date if this page is a journal.
	Date() time.Time

	// Open the page.
	Open() (Page, error)
}

type pageResultImpl struct {
	docType PageType
	title   string
	date    time.Time
	opener  func() (Page, error)
}

func (d *pageResultImpl) Type() PageType {
	return d.docType
}

func (d *pageResultImpl) Title() string {
	return d.title
}

func (d *pageResultImpl) Date() time.Time {
	return d.date
}

func (d *pageResultImpl) Open() (Page, error) {
	return d.opener()
}

var ErrBlockNotFound = errors.New("block not found")

// BlockResult represents a block in a page.
type BlockResult interface {
	// PageType gets the type of the page that this block belongs to.
	PageType() PageType

	// PageTitle gets the title of the page that this block belongs to.
	PageTitle() string

	// PageDate gets the date of the journal that this block belongs to. If
	// the page is not a journal, this will return the zero time.
	PageDate() time.Time

	// ID returns the stable identifier of the block, for use with block
	// references. If no ID is available, this will return an empty string.
	ID() string

	// Preview gets a preview of the block.
	Preview() string

	// OpenPage opens the page that this block belongs to.
	OpenPage() (Page, error)

	// Open the page of the block and return the block and page.
	Open() (*content.Block, Page, error)
}

type blockResultImpl struct {
	pageType  PageType
	pageTitle string
	pageDate  time.Time

	id       string
	preview  string
	location []int

	opener func() (Page, error)
}

func (b *blockResultImpl) PageType() PageType {
	return b.pageType
}

func (b *blockResultImpl) PageTitle() string {
	return b.pageTitle
}

func (b *blockResultImpl) PageDate() time.Time {
	return b.pageDate
}

func (b *blockResultImpl) ID() string {
	return b.id
}

func (b *blockResultImpl) Preview() string {
	return b.preview
}

func (b *blockResultImpl) OpenPage() (Page, error) {
	return b.opener()
}

func (b *blockResultImpl) Open() (*content.Block, Page, error) {
	page, err := b.opener()
	if err != nil {
		return nil, nil, err
	}

	if b.id != "" {
		// We have a stable identifier, use that to find the block
		block := page.Blocks().FindDeep(func(block *content.Block) bool {
			return b.id == block.ID()
		})

		if block != nil {
			return block, page, nil
		}

		return nil, nil, ErrBlockNotFound
	}

	// No stable id, walk the location and return the block
	blocks := page.Blocks()
	var block *content.Block
	for _, i := range b.location {
		if i > len(blocks) {
			return nil, nil, ErrBlockNotFound
		}

		block = blocks[i]
		blocks = block.Blocks()
	}

	return block, page, nil
}
