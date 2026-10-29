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
	// Revision is the revision of the page that the event is for, the same as
	// Page.Revision. Use it with WithExpectedRevision to save changes on top
	// of this version of the page.
	Revision string
	// OwnWrite is true when the change was made by saving a transaction of
	// this graph, and not by another program such as Logseq. Changes made via
	// another Graph, even in the same process, are not own writes.
	OwnWrite bool
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
	// OwnWrite is true when the page was deleted by saving a transaction of
	// this graph, and not by another program such as Logseq.
	OwnWrite bool
}

func (p *PageDeleted) isChangeEvent() {}

// PageRenamed is a change that indicates a page was moved to a new file, such
// as when a page is renamed in Logseq.
//
// The file system reports a rename as a removed file and a created file. These
// are reported as a rename when both changes happen within the same second and
// the created file has the same content as the removed file had. If more than
// one file matches, or the content was changed as part of the rename, the
// changes are reported as PageDeleted and PageUpdated instead.
//
// Renames are only reported for graphs that are opened with an index, as the
// index keeps the revision of the removed file.
type PageRenamed struct {
	// Page is the page at its new location.
	Page Page
	// Revision is the revision of the page that the event is for, the same as
	// Page.Revision.
	Revision string
	// OldType is the type of the page before it was renamed.
	OldType PageType
	// OldTitle is the title of the page before it was renamed.
	OldTitle string
	// OldDate is the date of the page before it was renamed. Set for journal
	// pages.
	OldDate time.Time
	// OwnWrite is true when both the removal and the write were made by saving
	// a transaction of this graph, and not by another program such as Logseq.
	OwnWrite bool
}

func (p *PageRenamed) isChangeEvent() {}

// WatchError is a change that indicates that the graph could not watch for
// changes or could not handle a change. The change that failed is not reported
// as another event, and the index can be out of date for that file until the
// file changes again.
//
// When the graph can not start to watch for changes, this is the only event
// that the watcher gets.
type WatchError struct {
	// Path is the path of the file that the error is for, relative to the
	// graph directory. It is empty when the error is not for a single file.
	Path string
	// Err is the error that occurred.
	Err error
}

func (e *WatchError) isChangeEvent() {}

// Error returns the error message, with the path when there is one.
func (e *WatchError) Error() string {
	if e.Path == "" {
		return e.Err.Error()
	}

	return e.Path + ": " + e.Err.Error()
}

// Unwrap returns the error that occurred.
func (e *WatchError) Unwrap() error {
	return e.Err
}
