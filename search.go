package logseq

import (
	"time"
)

type DocumentMetadata[D Document] interface {
	Type() DocumentType

	// Title returns the title for the document.
	Title() string

	// Date returns the date if this document is a journal.
	Date() time.Time

	// Open the document.
	Open() (D, error)
}

type BlockMetadata[S any] interface {
}

type DocumentIterator[D Document] interface {
	Next() (DocumentMetadata[D], error)
}

type documentMetadataImpl[D Document] struct {
	graph *Graph

	docType DocumentType
	title   string
	date    time.Time
	opener  func() (D, error)
}

func (d *documentMetadataImpl[D]) Type() DocumentType {
	return d.docType
}

func (d *documentMetadataImpl[D]) Title() string {
	return d.title
}

func (d *documentMetadataImpl[D]) Date() time.Time {
	return d.date
}

func (d *documentMetadataImpl[D]) Open() (Document, error) {
	return d.opener()
}
