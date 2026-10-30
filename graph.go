package logseq

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
	"github.com/aholstenson/logseq-go/internal/utils"
	"github.com/fsnotify/fsnotify"
)

// pageSource is an interface that is used to open pages and journals, and to
// find the blocks that relate to them. Used to delegate operations to the graph
// but allow for transactions to be used for opening pages, so that pages
// reached from other pages become part of the same transaction.
type pageSource interface {
	OpenJournal(date time.Time) (Page, error)

	OpenPage(title string) (Page, error)

	SearchPages(ctx context.Context, opts ...SearchOption) (SearchResults[PageResult], error)

	SearchBlocks(ctx context.Context, opts ...SearchOption) (SearchResults[BlockResult], error)
}

// Graph represents a Logseq graph. In Logseq a graph is a directory that
// contains Markdown files for pages and journals.
type Graph struct {
	options *options

	directory string

	config *utils.GraphConfig

	journalNameFormat  *utils.DateFormat
	journalTitleFormat *utils.DateFormat

	index         indexing.Index
	changeWatcher *fsnotify.Watcher

	// changeHandlers tracks the goroutines that debounce and index file
	// changes, so they can be waited for before the index is closed.
	changeHandlers sync.WaitGroup

	// ownWrites remembers what the graph wrote, so that the change events for
	// those writes can be marked as its own.
	ownWrites ownWrites

	mu       sync.Mutex
	watchers []*Watcher
}

func Open(ctx context.Context, directory string, opts ...Option) (*Graph, error) {
	// Apply the options
	options := &options{
		blockTimeFormatToNode: func(s string) content.InlineNode {
			return content.NewStrong(content.NewText(s))
		},
	}
	for _, option := range opts {
		option(options)
	}

	// Load the logseq/config.edn file.
	configFile := filepath.Join(directory, "logseq", "config.edn")
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the config file.
	config, err := utils.ParseConfig(configData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// This library reads and writes Markdown, so a graph in another format
	// can not be handled without silently mangling it.
	if config.PreferredFormat != utils.PreferredFormatMarkdown {
		return nil, fmt.Errorf("only Markdown graphs are supported, graph uses: %s", config.PreferredFormat)
	}

	// Parse the journal file name format.
	journalNameFormat := utils.NewDateFormat(config.JournalFileNameFormat)
	journalTitleFormat := utils.NewDateFormat(config.JournalPageTitleFormat)

	var index indexing.Index
	if options.index {
		index, err = indexing.NewBlugeIndex(config, options.indexDirectory)
		if err != nil {
			return nil, fmt.Errorf("failed to open index: %w", err)
		}
	}

	g := &Graph{
		options:   options,
		directory: directory,
		config:    config,

		journalNameFormat:  journalNameFormat,
		journalTitleFormat: journalTitleFormat,

		index: index,

		watchers: make([]*Watcher, 0),
	}

	// Sync the graph with the index
	err = g.sync(ctx, options.listener)
	if err != nil {
		if g.index != nil {
			g.index.Close()
		}
		return nil, fmt.Errorf("failed to sync graph: %w", err)
	}

	if g.index != nil {
		// The index must follow the files, so a graph with an index can not
		// be used if changes can not be watched.
		if err := g.watchForChanges(); err != nil {
			g.index.Close()
			return nil, err
		}
	}

	if options.listener != nil {
		options.listener(&OpenCompleted{})
	}

	return g, nil
}

func (g *Graph) Directory() string {
	return g.directory
}

func (g *Graph) NewTransaction() *Transaction {
	return newTransaction(g)
}

// journalDate normalizes a time into the date of a journal, which is midnight
// on the local calendar day the time falls on. Journals cover a calendar day,
// so this is how their dates are represented throughout the graph, letting a
// date survive a trip through the index and back.
func journalDate(t time.Time) time.Time {
	t = t.Local()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// Journal returns a read-only version of the journal page for the given date.
func (g *Graph) OpenJournal(date time.Time) (Page, error) {
	return g.openJournal(date, g)
}

func (g *Graph) openJournal(date time.Time, source pageSource) (Page, error) {
	date = journalDate(date)

	path, err := g.journalPath(date)
	if err != nil {
		return nil, err
	}

	title := g.journalTitleFormat.Format(date)

	page, err := openOrCreatePage(source, path, PageTypeJournal, title, date, g.markdownParseOptions()...)
	if err != nil {
		return nil, err
	}

	if page.IsNew() {
		err = g.applyJournalTemplate(page)
		if err != nil {
			return nil, err
		}
	}

	return page, nil
}

// applyJournalTemplate adds the blocks of the template that the graph uses for
// new journals, which is set with `:default-templates {:journals "Name"}` in
// the config of the graph. The value is the name of a template, as in
// `template:: Name`.
//
// Logseq leaves a new journal empty if the template does not exist, and so
// does this. Templates are found via the index, so without an index new
// journals are also empty.
func (g *Graph) applyJournalTemplate(page *pageImpl) error {
	name := g.config.DefaultTemplates.Journals
	if name == "" || g.index == nil {
		return nil
	}

	// The variables in the template are filled in as if the template was
	// inserted on the day of the journal at the current time, so that
	// `<% today %>` links to the journal itself.
	now := time.Now()
	date := page.Date()
	insertedAt := time.Date(date.Year(), date.Month(), date.Day(), now.Hour(), now.Minute(), now.Second(), 0, date.Location())

	blocks, err := g.Template(context.Background(), name,
		WithTemplateTime(insertedAt),
		WithTemplatePage(page.Title()),
	)
	if errors.Is(err, ErrTemplateNotFound) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to apply the journal template %s: %w", name, err)
	}

	for _, block := range blocks {
		page.AddBlock(block)
	}

	return nil
}

func (g *Graph) journalPath(date time.Time) (string, error) {
	filename := g.journalNameFormat.Format(date) + ".md"
	return filepath.Join(g.directory, g.config.JournalsDir, filename), nil
}

// Page returns a read-only version of a page for the given path.
func (g *Graph) OpenPage(title string) (Page, error) {
	return g.openPage(title, g)
}

func (g *Graph) openPage(title string, source pageSource) (Page, error) {
	path, err := g.pagePath(title)
	if err != nil {
		return nil, err
	}

	page, err := openOrCreatePage(source, path, PageTypeDedicated, title, time.Time{}, g.markdownParseOptions()...)
	if err != nil {
		return nil, err
	}

	if !page.IsNew() || g.index == nil {
		// The page has content of its own, or there is no index to look for
		// aliases in.
		return page, nil
	}

	// Nothing is stored under this title, so it may be an alias of another
	// page. Opening the page it is an alias of keeps a second page for the same
	// content from being created.
	target, err := g.pageTitleForAlias(context.Background(), title)
	if err != nil {
		return nil, err
	}

	if target == "" {
		return page, nil
	}

	path, err = g.pagePath(target)
	if err != nil {
		return nil, err
	}

	return openOrCreatePage(source, path, PageTypeDedicated, target, time.Time{}, g.markdownParseOptions()...)
}

// pageTitleForAlias finds the title of the page that has the given title as one
// of its aliases, returning an empty string if no page does. If more than one
// page uses the alias, which Logseq does not expect, one of them is picked.
func (g *Graph) pageTitleForAlias(ctx context.Context, alias string) (string, error) {
	results, err := g.index.SearchPages(ctx, indexing.HasAlias(alias), indexing.SearchOptions{
		Size: 1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to look up the page with the alias %s: %w", alias, err)
	}

	for _, page := range results.Results() {
		if page.Type == indexing.PageTypeDedicated {
			return page.Title, nil
		}
	}

	return "", nil
}

func (g *Graph) pagePath(title string) (string, error) {
	path, err := utils.TitleToFilename(g.config.FileNameFormat, title)
	if err != nil {
		return "", err
	}

	return filepath.Join(g.directory, g.config.PagesDir, path+".md"), nil
}

// removePageFile removes the file a page is stored in. If the graph was opened
// with WithRecycleDeletedPages the file is moved into the recycle directory
// instead, so that it can be recovered.
func (g *Graph) removePageFile(path string) error {
	if !g.options.recycleDeletedPages {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove page at %s: %w", path, err)
		}

		return nil
	}

	recycleDir := filepath.Join(g.directory, "logseq", ".recycle")
	if err := os.MkdirAll(recycleDir, 0755); err != nil {
		return fmt.Errorf("failed to create recycle directory: %w", err)
	}

	target := filepath.Join(recycleDir, filepath.Base(path))
	if err := os.Rename(path, target); err != nil {
		return fmt.Errorf("failed to recycle page at %s: %w", path, err)
	}

	return nil
}

// openViaPath opens the page stored at the given path. A nil page is returned
// for files that are not part of the graph, see pageFile.
func (g *Graph) openViaPath(path string, source pageSource) (Page, error) {
	file, ok := g.pageFile(path)
	if !ok {
		return nil, nil
	}

	switch file.pageType {
	case PageTypeJournal:
		title := g.journalTitleFormat.Format(file.date)

		return openOrCreatePage(source, path, PageTypeJournal, title, file.date, g.markdownParseOptions()...)
	default:
		title, err := utils.FilenameToTitle(g.config.FileNameFormat, file.name)
		if err != nil {
			return nil, fmt.Errorf("failed to get title from filename: %w", err)
		}

		return openOrCreatePage(source, path, PageTypeDedicated, title, time.Time{}, g.markdownParseOptions()...)
	}
}

// pageFileInfo describes a file that holds a page of the graph.
type pageFileInfo struct {
	pageType PageType
	// name is the file name without the extension.
	name string
	// date is the date of a journal, parsed from the file name.
	date time.Time
}

// pageFile tells if the file at the given path holds a page of the graph,
// without reading the file. Files that are not Markdown, files outside the
// pages and journals directories, such as in a subdirectory of them, and
// journals whose name does not match the configured format are not part of
// the graph.
func (g *Graph) pageFile(path string) (pageFileInfo, bool) {
	name := filepath.Base(path)
	if filepath.Ext(name) != ".md" {
		return pageFileInfo{}, false
	}

	name = name[:len(name)-3]
	dir := filepath.Dir(path)

	switch dir {
	case filepath.Join(g.directory, g.config.JournalsDir):
		date, err := g.journalNameFormat.Parse(name)
		if err != nil {
			// Ignore files that don't match the journal name format
			return pageFileInfo{}, false
		}

		return pageFileInfo{pageType: PageTypeJournal, name: name, date: date}, true
	case filepath.Join(g.directory, g.config.PagesDir):
		return pageFileInfo{pageType: PageTypeDedicated, name: name}, true
	}

	return pageFileInfo{}, false
}

func (g *Graph) Close() error {
	if g.changeWatcher != nil {
		g.changeWatcher.Close()
	}

	// Changes are indexed as they come in, so wait for that to stop before
	// closing the index out from under it.
	g.changeHandlers.Wait()

	if g.index != nil {
		return g.index.Close()
	}

	return nil
}

// sync makes the index match the files of the graph. It first scans the
// journals and pages directories for files that changed since they were
// indexed, then indexes those files, and last removes the pages of files that
// no longer exist. Scanning first gives the total for the progress events.
func (g *Graph) sync(ctx context.Context, listener func(event OpenEvent)) error {
	if g.index == nil {
		return nil
	}

	notify := func(event OpenEvent) {
		if listener != nil {
			listener(event)
		}
	}

	started := time.Now()
	notify(&ScanStarted{})

	scan := &syncScan{
		seen: make(map[string]struct{}),
	}

	journalsDir := filepath.Join(g.directory, g.config.JournalsDir)
	if err := filepath.Walk(journalsDir, g.createScanner(ctx, scan)); err != nil {
		return fmt.Errorf("failed to scan journals: %w", err)
	}

	pagesDir := filepath.Join(g.directory, g.config.PagesDir)
	if err := filepath.Walk(pagesDir, g.createScanner(ctx, scan)); err != nil {
		return fmt.Errorf("failed to scan pages: %w", err)
	}

	total := len(scan.changed)
	notify(&SyncStarted{
		Total:     total,
		Unchanged: len(scan.seen) - total,
	})

	failed := 0
	for i, subPath := range scan.changed {
		if err := ctx.Err(); err != nil {
			return err
		}

		path := filepath.Join(g.directory, subPath)

		// A page that can not be read is skipped, so that one bad file does
		// not keep the graph from opening. The index is not changed for the
		// file, so it is tried again the next time the graph is opened.
		page, err := g.openViaPath(path, g)
		if err != nil {
			failed++
			notify(&PageIndexFailed{
				SubPath: subPath,
				Err:     err,
				Done:    i + 1,
				Total:   total,
			})
			continue
		}

		// The scan only keeps files that hold pages, so page is not nil.
		// Errors from the index are not about this page and will most likely
		// happen for all pages, so they stop the sync.
		if err := g.indexPage(ctx, page, path); err != nil {
			return fmt.Errorf("failed to index %s: %w", subPath, err)
		}

		notify(&PageIndexed{
			SubPath: subPath,
			Done:    i + 1,
			Total:   total,
		})
	}

	removed, err := g.removeMissingPages(ctx, scan.seen)
	if err != nil {
		return err
	}

	if err := g.index.Sync(); err != nil {
		return err
	}

	notify(&SyncCompleted{
		Indexed:  total - failed,
		Failed:   failed,
		Removed:  removed,
		Duration: time.Since(started),
	})

	return nil
}

// syncScan is the result of scanning the graph for files to index.
type syncScan struct {
	// seen has the sub path of every file that holds a page of the graph.
	seen map[string]struct{}
	// changed has the sub paths of the files that must be indexed, in the
	// order they were found.
	changed []string
}

// createScanner returns a function that walks a directory and adds the files
// that hold pages of the graph to the scan.
func (g *Graph) createScanner(ctx context.Context, scan *syncScan) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}

		subPath, err := filepath.Rel(g.directory, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		if g.config.IsHidden(subPath) {
			// Files and directories hidden via the config are not part of the
			// graph as far as Logseq is concerned.
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if info.IsDir() {
			return nil
		}

		if _, ok := g.pageFile(path); !ok {
			return nil
		}

		scan.seen[subPath] = struct{}{}

		state, err := g.index.GetFileState(ctx, subPath)
		if err != nil {
			return fmt.Errorf("failed to get last modified: %w", err)
		} else if state.LastModified.Equal(info.ModTime()) && state.Revision != "" {
			// Page is assumed to be up to date if times match. Pages indexed
			// without a revision are indexed again, as the revision is needed
			// to report renames.
			return nil
		}

		scan.changed = append(scan.changed, subPath)
		return nil
	}
}

// removeMissingPages removes the pages from the index whose files were not
// seen during the scan, such as files that were deleted or hidden while the
// graph was closed. Returns the number of pages that were removed.
func (g *Graph) removeMissingPages(ctx context.Context, seen map[string]struct{}) (int, error) {
	indexed, err := g.index.PageSubPaths(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list indexed pages: %w", err)
	}

	removed := 0
	for _, subPath := range indexed {
		if _, ok := seen[subPath]; ok {
			continue
		}

		if err := g.index.DeletePage(ctx, subPath); err != nil {
			return removed, fmt.Errorf("failed to remove %s from index: %w", subPath, err)
		}

		removed++
	}

	return removed, nil
}

func (g *Graph) indexDocument(ctx context.Context, docPath string) (Page, error) {
	page, err := g.openViaPath(docPath, g)
	if err != nil {
		return nil, fmt.Errorf("failed to open page: %w", err)
	}

	if page == nil {
		// If the page didn't pass validation skip it
		return nil, nil
	}

	if err := g.indexPage(ctx, page, docPath); err != nil {
		return page, err
	}

	return page, nil
}

// indexPage adds a page that is open to the index, replacing what the index
// has for the file at the given path.
func (g *Graph) indexPage(ctx context.Context, page Page, docPath string) error {
	doc := &indexing.Page{
		Type:         indexing.PageType(page.Type()),
		LastModified: page.LastModified(),
		Revision:     page.Revision(),
		Date:         page.Date(),
		Blocks:       page.Blocks(),
		Title:        page.Title(),
	}

	doc.SubPath, _ = filepath.Rel(g.directory, docPath)

	if impl, ok := page.(*pageImpl); ok {
		// Look up the properties without creating them, as indexing should not
		// modify the page.
		doc.Properties = impl.findProperties()
		doc.Aliases = impl.Aliases()
	}

	if err := g.index.IndexPage(ctx, doc); err != nil {
		return fmt.Errorf("failed to index page: %w", err)
	}

	return nil
}

// watchForChanges starts to watch the journals and pages directories for
// changes. Must be called with g.mu held or before the graph is shared.
func (g *Graph) watchForChanges() error {
	changeWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	for _, dir := range []string{g.config.JournalsDir, g.config.PagesDir} {
		err = changeWatcher.Add(filepath.Join(g.directory, dir))
		if err != nil {
			changeWatcher.Close()
			return fmt.Errorf("failed to watch %s directory: %w", dir, err)
		}
	}

	g.changeWatcher = changeWatcher

	changes := make(chan *pendingChange)
	errs := make(chan error)
	// done is closed when the watcher has stopped, which lets debounce timers
	// that fire around that point give up instead of sending on a channel that
	// no longer has a receiver.
	done := make(chan struct{})
	pending := newPendingChanges()

	g.changeHandlers.Add(2)

	go func() {
		defer g.changeHandlers.Done()

	_outer:
		for {
			select {
			case event, ok := <-changeWatcher.Events:
				if !ok {
					break _outer
				}

				// A file that is moved away is reported as renamed, while the
				// file it is moved to is reported as created.
				if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) && !event.Has(fsnotify.Remove) && !event.Has(fsnotify.Rename) {
					continue
				}

				if filepath.Ext(event.Name) != ".md" {
					// Only handle Markdown files
					continue
				}

				if subPath, err := filepath.Rel(g.directory, event.Name); err == nil && g.config.IsHidden(subPath) {
					// Hidden files are not part of the graph
					continue
				}

				// Logseq will save as you write, so debounce changes to files
				// so we don't index too often
				pending.schedule(event.Name, changeDebounce, func(change *pendingChange) {
					select {
					case changes <- change:
					case <-done:
					}
				})
			case err, ok := <-changeWatcher.Errors:
				if !ok {
					break _outer
				}

				// Errors are sent to watchers from the other goroutine, so
				// that all events to watchers are sent in order.
				errs <- err
			}
		}

		// When the watcher is closed remove all of the current timers. Timers
		// that already fired are released via done instead of being stopped.
		close(done)
		pending.stop()
	}()

	go func() {
		defer g.changeHandlers.Done()

		ctx := context.Background()
		for {
			var change *pendingChange
			select {
			case change = <-changes:
			case err := <-errs:
				g.notifyWatchers(&WatchError{Err: fmt.Errorf("file watcher failed: %w", err)}, done)
				continue
			case <-done:
				return
			}

			if !pending.take(change) {
				// The change was already handled together with another one,
				// such as the other half of a rename.
				continue
			}

			report := func(path string, err error) {
				subPath, _ := filepath.Rel(g.directory, path)
				g.notifyWatchers(&WatchError{Path: subPath, Err: err}, done)
			}

			event := g.handleChange(ctx, change.path, pending, report)
			if event != nil {
				g.notifyWatchers(event, done)
			}
		}
	}()

	return nil
}

// notifyWatchers sends an event to all current watchers. It gives up on a
// watcher when the watcher is closed, and on all of them when done is closed.
func (g *Graph) notifyWatchers(event ChangeEvent, done <-chan struct{}) {
	var watchers []*Watcher
	g.mu.Lock()
	watchers = append(watchers, g.watchers...)
	g.mu.Unlock()

	for _, watcher := range watchers {
		select {
		case watcher.changes <- event:
		case <-watcher.done:
		case <-done:
		}
	}
}

// handleChange updates the index for a changed file and returns the event to
// send to watchers, or nil if there is nothing to report. Other changes that
// wait in pending are checked to find renames. Errors are given to report,
// with the path of the file that they are for.
func (g *Graph) handleChange(ctx context.Context, path string, pending *pendingChanges, report func(path string, err error)) ChangeEvent {
	// Figure out if the page still exists
	exists := true
	_, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			report(path, fmt.Errorf("failed to check if page exists: %w", err))
			return nil
		}

		exists = false
	}

	if g.index == nil {
		// Without an index there are no revisions of removed files to find
		// renames with, so changes are reported as they are.
		if !exists {
			return g.deletedEvent(path)
		}

		page, err := g.openViaPath(path, g)
		if err != nil {
			report(path, fmt.Errorf("failed to open page: %w", err))
			return nil
		}

		if page == nil {
			return nil
		}

		return g.updatedEvent(path, page)
	}

	// Sync after indexing so changes are visible
	defer func() {
		if err := g.index.Sync(); err != nil {
			report(path, fmt.Errorf("failed to sync index: %w", err))
		}
	}()

	subPath, _ := filepath.Rel(g.directory, path)
	state, err := g.index.GetFileState(ctx, subPath)
	if err != nil {
		// Without the state the change is handled as a file that was not
		// indexed before.
		report(path, fmt.Errorf("failed to get indexed state of page: %w", err))
	}

	if !exists {
		if err := g.index.DeletePage(ctx, subPath); err != nil {
			report(path, fmt.Errorf("failed to remove page from index: %w", err))
		}

		if g.createPageDeletedEvent(path) == nil {
			return nil
		}

		if to := g.claimRenameTarget(ctx, pending, state.Revision); to != "" {
			page, err := g.indexDocument(ctx, to)
			if err != nil {
				report(to, err)
			}

			if page != nil {
				return g.renamedEvent(path, to, page)
			}
		} else {
			pending.handledUnpaired(state.Revision)
		}

		return g.deletedEvent(path)
	}

	page, err := g.indexDocument(ctx, path)
	if err != nil {
		report(path, err)
		return nil
	}

	if page == nil {
		return nil
	}

	if state == (indexing.FileState{}) {
		// The file was not indexed before, so it may be the new location of a
		// file that was removed.
		if from := g.claimRenameSource(ctx, pending, page.Revision()); from != "" {
			fromSubPath, _ := filepath.Rel(g.directory, from)
			if err := g.index.DeletePage(ctx, fromSubPath); err != nil {
				report(from, fmt.Errorf("failed to remove page from index: %w", err))
			}

			return g.renamedEvent(from, path, page)
		}

		pending.handledUnpaired(page.Revision())
	}

	return g.updatedEvent(path, page)
}

func (g *Graph) updatedEvent(path string, page Page) *PageUpdated {
	return &PageUpdated{
		Page:     page,
		Revision: page.Revision(),
		OwnWrite: g.ownWrites.consume(path, page.Revision()),
	}
}

func (g *Graph) deletedEvent(path string) ChangeEvent {
	deleted := g.createPageDeletedEvent(path)
	if deleted == nil {
		return nil
	}

	deleted.OwnWrite = g.ownWrites.consume(path, "")
	return deleted
}

func (g *Graph) renamedEvent(from string, to string, page Page) *PageRenamed {
	// Both pages are checked to be part of the graph before they are paired.
	deleted := g.createPageDeletedEvent(from)

	// Both writes are consumed, so that neither of them is left to mark a
	// later change as an own write.
	removed := g.ownWrites.consume(from, "")
	written := g.ownWrites.consume(to, page.Revision())

	return &PageRenamed{
		Page:     page,
		Revision: page.Revision(),
		OldType:  deleted.Type,
		OldTitle: deleted.Title,
		OldDate:  deleted.Date,
		OwnWrite: removed && written,
	}
}

func (g *Graph) createPageDeletedEvent(path string) *PageDeleted {
	name := filepath.Base(path)
	name = name[:len(name)-3]

	dir := filepath.Dir(path)
	if dir == filepath.Join(g.directory, g.config.JournalsDir) {
		date, err := g.journalNameFormat.Parse(name)
		if err != nil {
			// Ignore files that don't match the journal name format
			return nil
		}

		return &PageDeleted{
			Type:  PageTypeJournal,
			Date:  date,
			Title: g.journalTitleFormat.Format(date),
		}
	} else if dir == filepath.Join(g.directory, g.config.PagesDir) {
		title, err := utils.FilenameToTitle(g.config.FileNameFormat, name)
		if err != nil {
			return nil
		}

		return &PageDeleted{
			Type:  PageTypeDedicated,
			Title: title,
		}
	}

	return nil
}

// SearchPages searches for pages in the graph.
func (g *Graph) SearchPages(ctx context.Context, opts ...SearchOption) (SearchResults[PageResult], error) {
	return g.searchPages(ctx, opts, g)
}

func (g *Graph) searchPages(ctx context.Context, opts []SearchOption, source pageSource) (SearchResults[PageResult], error) {
	if g.index == nil {
		return nil, fmt.Errorf("indexing is not enabled")
	}

	options := &searchOptions{
		size:   10,
		sortBy: []indexing.SortField{},
	}

	for _, opt := range opts {
		opt(options)
	}

	if options.query == nil {
		options.query = indexing.All()
	}

	if options.size <= 0 {
		options.size = 10
	}

	results, err := g.index.SearchPages(ctx, options.query, indexing.SearchOptions{
		Size:   options.size,
		From:   options.from,
		SortBy: options.sortBy,
	})
	if err != nil {
		return nil, err
	}

	return newSearchResults(results, func(page *indexing.Page) PageResult {
		if page.Type == indexing.PageTypeJournal {
			date := journalDate(page.Date)
			return &pageResultImpl{
				docType: PageTypeJournal,
				title:   g.journalTitleFormat.Format(date),
				date:    date,
				props:   page.PropertyValues,

				opener: func() (Page, error) {
					return source.OpenJournal(date)
				},
			}
		} else {
			return &pageResultImpl{
				docType: PageTypeDedicated,
				title:   page.Title,
				date:    time.Time{},
				props:   page.PropertyValues,

				opener: func() (Page, error) {
					return source.OpenPage(page.Title)
				},
			}
		}
	}), nil
}

// OpenBlock opens the block with the given id, which is the identifier that
// block references such as `((id))` point at. The page the block belongs to is
// returned as well, as the block is part of it.
//
// Blocks are found via the index, so this requires the graph to have been
// opened with indexing enabled. If no block with the id is found, or the block
// no longer exists on the page it was indexed on, ErrBlockNotFound is returned.
func (g *Graph) OpenBlock(ctx context.Context, id string) (*content.Block, Page, error) {
	return g.openBlock(ctx, id, g)
}

func (g *Graph) openBlock(ctx context.Context, id string, source pageSource) (*content.Block, Page, error) {
	if g.index == nil {
		return nil, nil, fmt.Errorf("indexing is not enabled")
	}

	if id == "" {
		return nil, nil, ErrBlockNotFound
	}

	results, err := g.searchBlocks(ctx, []SearchOption{
		WithQuery(BlockIDEquals(id)),
		WithMaxHits(1),
	}, source)
	if err != nil {
		return nil, nil, err
	}

	if results.Size() == 0 {
		return nil, nil, ErrBlockNotFound
	}

	return results.Results()[0].Open()
}

// SearchBlocks searches for blocks in the graph.
func (g *Graph) SearchBlocks(ctx context.Context, opts ...SearchOption) (SearchResults[BlockResult], error) {
	return g.searchBlocks(ctx, opts, g)
}

func (g *Graph) searchBlocks(ctx context.Context, opts []SearchOption, source pageSource) (SearchResults[BlockResult], error) {
	if g.index == nil {
		return nil, fmt.Errorf("indexing is not enabled")
	}

	options := &searchOptions{
		size:   10,
		sortBy: []indexing.SortField{},
	}

	for _, opt := range opts {
		opt(options)
	}

	if options.query == nil {
		options.query = indexing.All()
	}

	if options.size <= 0 {
		options.size = 10
	}

	results, err := g.index.SearchBlocks(ctx, options.query, indexing.SearchOptions{
		Size:   options.size,
		From:   options.from,
		SortBy: options.sortBy,
	})
	if err != nil {
		return nil, err
	}

	return newSearchResults(results, func(block *indexing.Block) BlockResult {
		dir := filepath.Dir(block.PageSubPath)
		name := filepath.Base(block.PageSubPath)
		name = name[:len(name)-3]

		var err error
		pageType := PageTypeDedicated
		pageDate := time.Time{}
		var pageTitle string
		if dir == g.config.JournalsDir {
			pageType = PageTypeJournal

			pageDate, err = g.journalNameFormat.Parse(name)
			if err != nil {
				// TODO: This is an edge case where the format of journals has changed since indexing
			}

			pageTitle = g.journalTitleFormat.Format(pageDate)
		} else {
			pageTitle, err = utils.FilenameToTitle(g.config.FileNameFormat, name)
			if err != nil {
				// TODO: This page is not in the expected format
			}
		}

		return &blockResultImpl{
			pageType:  pageType,
			pageTitle: pageTitle,
			pageDate:  pageDate,

			id:       block.ID,
			preview:  block.Preview,
			props:    block.PropertyValues,
			location: block.Location,

			opener: func() (Page, error) {
				if pageType == PageTypeJournal {
					return source.OpenJournal(pageDate)
				} else {
					return source.OpenPage(pageTitle)
				}
			},
		}
	}), nil
}

func (g *Graph) Watch() *Watcher {
	watcher := &Watcher{
		changes: make(chan ChangeEvent),
		done:    make(chan struct{}),
	}

	watcher.closer = func() {
		g.mu.Lock()
		defer g.mu.Unlock()

		// Remove from the list of watchers
		for i, w := range g.watchers {
			if w == watcher {
				g.watchers = append(g.watchers[:i], g.watchers[i+1:]...)
				break
			}
		}

		if len(g.watchers) == 0 && g.index == nil && g.changeWatcher != nil {
			// Nothing is watching and there is no index that needs to be kept
			// up to date, so stop watching until something needs it again. With
			// an index the graph keeps watching for the rest of its life.
			g.changeWatcher.Close()
			g.changeWatcher = nil
		}
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.watchers = append(g.watchers, watcher)

	if g.changeWatcher == nil {
		// Start watching for changes if we're not already. Watch has no error
		// to return, so the error is sent as the only event of the watcher.
		if err := g.watchForChanges(); err != nil {
			go func() {
				select {
				case watcher.changes <- &WatchError{Err: err}:
				case <-watcher.done:
				}
			}()
		}
	}

	return watcher
}
