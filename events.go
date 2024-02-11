package logseq

import "time"

// OpenEvent is an event that occurs while the graph is being opened.
type OpenEvent interface {
	isOpenEvent()
}

// PageIndexed is an event that occurs when a page is indexed.
type PageIndexed struct {
	SubPath string
}

func (p *PageIndexed) isOpenEvent() {}

type ChangeEvent interface {
	isChangeEvent()
}

// PageUpdated is a change that indicates a page was updated or created.
type PageUpdated struct {
	// Page is the page that was updated.
	Page Page
}

func (p *PageUpdated) isChangeEvent() {}

// PageDeleted is a change that indicates a page was deleted.
type PageDeleted struct {
	// Type is the type of the page that was deleted.
	Type PageType
	// Title is the title of the page that was deleted.
	Title string
	// Date is the date the page was deleted. Set for journal pages.
	Date time.Time
}

func (p *PageDeleted) isChangeEvent() {}
