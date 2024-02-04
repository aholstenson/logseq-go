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

	// DeleteDocument removes a document from the index.
	DeleteDocument(ctx context.Context, subPath string) error

	// IndexDocument indexes a page or journal in the search index.
	IndexDocument(ctx context.Context, doc *Document) error

	// GetLastModified returns the last modified time for a page. Should return
	// a zero time if the page does not exist in the index.
	GetLastModified(ctx context.Context, subPath string) (time.Time, error)

	ListDocuments(ctx context.Context, query Query) (Iterator[*Document], error)
}

type Iterator[V any] interface {
	Next() (V, error)
}

type DocumentType int

const (
	DocumentTypePage DocumentType = iota
	DocumentTypeJournal
)

type Document struct {
	// SubPath is the sub path of the document in the graph.
	SubPath string

	// Type is the type of the document.
	Type DocumentType

	// LastModified is the last time the document was modified on disk.
	LastModified time.Time

	// Title is the title of the page. Only used for pages.
	Title string
	// Date is the date of the journal. Only used for journals.
	Date time.Time

	// Blocks is the blocks of the document, only used while indexing.
	Blocks content.BlockList
}
