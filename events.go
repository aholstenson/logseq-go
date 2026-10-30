package logseq

import "time"

// OpenEvent is an event that occurs while the graph is being opened. Use
// WithListener to get these events, for example to show the progress of
// opening a graph.
//
// A graph opened with an index sends the events in this order:
//
//  1. ScanStarted, once.
//  2. SyncStarted, once, with the number of pages to index.
//  3. PageIndexed, once for each page that is indexed.
//  4. SyncCompleted, once.
//  5. OpenCompleted, once.
//
// A graph opened without an index sends only OpenCompleted. When opening
// fails, the events stop and OpenCompleted is not sent.
type OpenEvent interface {
	isOpenEvent()
}

// ScanStarted is an event that occurs when the graph starts to look for the
// pages that changed since they were indexed. The number of pages to index is
// not known until SyncStarted.
type ScanStarted struct{}

func (e *ScanStarted) isOpenEvent() {}

// SyncStarted is an event that occurs when the scan is done and the graph
// starts to index the pages that changed.
type SyncStarted struct {
	// Total is the number of pages to index. It is zero when the index is up
	// to date.
	Total int
	// Unchanged is the number of pages that are up to date in the index and
	// are not indexed again.
	Unchanged int
}

func (e *SyncStarted) isOpenEvent() {}

// PageIndexed is an event that occurs when a page is indexed.
type PageIndexed struct {
	// SubPath is the path of the page file, relative to the graph directory.
	SubPath string
	// Done is the number of pages that are indexed, including this page. It
	// is the same as Total for the last page.
	Done int
	// Total is the number of pages to index, the same as SyncStarted.Total.
	Total int
}

func (p *PageIndexed) isOpenEvent() {}

// SyncCompleted is an event that occurs when the index matches the files of
// the graph.
type SyncCompleted struct {
	// Indexed is the number of pages that were indexed.
	Indexed int
	// Removed is the number of pages that were removed from the index, as
	// their files were deleted or hidden while the graph was closed.
	Removed int
	// Duration is the time from ScanStarted to this event.
	Duration time.Duration
}

func (e *SyncCompleted) isOpenEvent() {}

// OpenCompleted is an event that occurs when the graph is open and ready to
// use. For a graph with an index, it occurs after the graph starts to watch
// for changes.
type OpenCompleted struct{}

func (e *OpenCompleted) isOpenEvent() {}

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
