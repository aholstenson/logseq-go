package logseq

import (
	"context"
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

// pageSource is an interface that is used to open pages and journals. Used
// to delegate operations to the graph but allow for transactions to be used
// for opening pages.
type pageSource interface {
	OpenJournal(date time.Time) (Page, error)

	OpenPage(title string) (Page, error)
}

// Graph represents a Logseq graph. In Logseq a graph is a directory that
// contains Markdown files for pages and journals.
type Graph struct {
	options *options

	directory string

	config *utils.GraphConfig

	journalNameFormat  string
	journalTitleFormat string

	index         indexing.Index
	changeWatcher *fsnotify.Watcher

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

	// Parse the journal file name format.
	journalNameFormat := utils.ConvertDateFormat(config.Journal.FileNameFormat)
	journalTitleFormat := utils.ConvertDateFormat(config.Journal.PageTitleFormat)

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
		return nil, fmt.Errorf("failed to sync graph: %w", err)
	}

	if g.index != nil {
		g.watchForChanges()
	}
	return g, nil
}

func (g *Graph) Directory() string {
	return g.directory
}

func (g *Graph) NewTransaction() *Transaction {
	return newTransaction(g)
}

// Journal returns a read-only version of the journal page for the given date.
func (g *Graph) OpenJournal(date time.Time) (Page, error) {
	date = date.Local().Truncate(24 * time.Hour)

	path, err := g.journalPath(date)
	if err != nil {
		return nil, err
	}

	templatePath := ""
	if g.config.DefaultTemplates.Journals != "" {
		templatePath = filepath.Join(g.directory, g.config.DefaultTemplates.Journals)
	}

	title := date.Format(g.journalTitleFormat)

	return openOrCreatePage(path, PageTypeJournal, title, date, templatePath)
}

func (g *Graph) journalPath(date time.Time) (string, error) {
	filename := date.Format(g.journalNameFormat) + ".md"
	return filepath.Join(g.directory, g.config.JournalsDir, filename), nil
}

// Page returns a read-only version of a page for the given path.
func (g *Graph) OpenPage(title string) (Page, error) {
	path, err := g.pagePath(title)
	if err != nil {
		return nil, err
	}

	return openOrCreatePage(path, PageTypeDedicated, title, time.Time{}, "")
}

func (g *Graph) pagePath(title string) (string, error) {
	path, err := utils.TitleToFilename(g.config.File.NameFormat, title)
	if err != nil {
		return "", err
	}

	return filepath.Join(g.directory, g.config.PagesDir, path+".md"), nil
}

func (g *Graph) openViaPath(path string) (Page, error) {
	name := filepath.Base(path)
	if filepath.Ext(name) != ".md" {
		return nil, fmt.Errorf("not a Markdown file")
	}

	name = name[:len(name)-3]
	dir := filepath.Dir(path)

	if dir == filepath.Join(g.directory, g.config.JournalsDir) {
		date, err := time.Parse(g.journalNameFormat, name)
		if err != nil {
			// Ignore files that don't match the journal name format
			return nil, nil
		}

		title := date.Format(g.journalTitleFormat)

		return openOrCreatePage(path, PageTypeJournal, title, date, "")
	} else if dir == filepath.Join(g.directory, g.config.PagesDir) {
		title, err := utils.FilenameToTitle(g.config.File.NameFormat, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get title from filename: %w", err)
		}

		return openOrCreatePage(path, PageTypeDedicated, title, time.Time{}, "")
	}

	return nil, fmt.Errorf("not a page or journal")
}

func (g *Graph) Close() error {
	if g.changeWatcher != nil {
		g.changeWatcher.Close()
	}

	if g.index != nil {
		return g.index.Close()
	}

	return nil
}

// sync performs a sync of the graph with the index.
func (g *Graph) sync(ctx context.Context, listener func(event OpenEvent)) error {
	if g.index == nil {
		return nil
	}

	walker := g.createWalker(ctx, listener)

	// Sync the journal pages
	journalsDir := filepath.Join(g.directory, g.config.JournalsDir)
	err := filepath.Walk(journalsDir, walker)
	if err != nil {
		return fmt.Errorf("failed to sync journals: %w", err)
	}

	// Sync the note pages
	notesDir := filepath.Join(g.directory, g.config.PagesDir)
	err = filepath.Walk(notesDir, walker)
	if err != nil {
		return fmt.Errorf("failed to sync pages: %w", err)
	}

	return g.index.Sync()
}

func (g *Graph) createWalker(ctx context.Context, listener func(event OpenEvent)) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk journals directory: %w", err)
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".md" {
			return nil
		}

		subPath, err := filepath.Rel(g.directory, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		lastModified, err := g.index.GetLastModified(ctx, subPath)
		if err != nil {
			return fmt.Errorf("failed to get last modified: %w", err)
		} else if lastModified.Equal(info.ModTime()) {
			// Page is assumed to be up to date if times match
			return nil
		}

		_, err = g.indexDocument(ctx, path)
		if err != nil {
			return fmt.Errorf("failed to index document: %w", err)
		}

		if listener != nil {
			listener(&PageIndexed{
				SubPath: subPath,
			})
		}
		return nil
	}
}

func (g *Graph) indexDocument(ctx context.Context, docPath string) (Page, error) {
	page, err := g.openViaPath(docPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open page: %w", err)
	}

	if page == nil {
		// If the page didn't pass validation skip it
		return nil, nil
	}

	doc := &indexing.Page{
		Type:         indexing.PageType(page.Type()),
		LastModified: page.LastModified(),
		Date:         page.Date(),
		Blocks:       page.Blocks(),
		Title:        page.Title(),
	}
	doc.SubPath, _ = filepath.Rel(g.directory, docPath)
	return page, g.index.IndexPage(ctx, doc)
}

func (g *Graph) watchForChanges() {
	changeWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		// TODO: Error reporting
		return
	}

	g.changeWatcher = changeWatcher

	err = g.changeWatcher.Add(filepath.Join(g.directory, g.config.JournalsDir))
	if err != nil {
		return
	}

	err = g.changeWatcher.Add(filepath.Join(g.directory, g.config.PagesDir))
	if err != nil {
		return
	}

	changes := make(chan string)
	changeTimers := make(map[string]*time.Timer)
	var mu sync.Mutex

	go func() {
	_outer:
		for {
			select {
			case event, ok := <-g.changeWatcher.Events:
				if !ok {
					break _outer
				}

				if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) && !event.Has(fsnotify.Remove) {
					continue
				}

				if filepath.Ext(event.Name) != ".md" {
					// Only handle Markdown files
					continue
				}

				path := event.Name

				// Logseq will save as you write, so debounce changes to files
				// so we don't index too often
				mu.Lock()
				if timer, found := changeTimers[path]; found {
					timer.Stop()
				}

				changeTimers[path] = time.AfterFunc(1*time.Second, func() {
					mu.Lock()
					delete(changeTimers, path)
					mu.Unlock()

					changes <- path
				})
				mu.Unlock()
			case _, ok := <-g.changeWatcher.Errors:
				if !ok {
					break _outer
				}

				// TODO: Log error
			}
		}

		// When the watcher is closed remove all of the current timers
		mu.Lock()
		defer mu.Unlock()
		for _, timer := range changeTimers {
			timer.Stop()
		}
		close(changes)
	}()

	go func() {
		ctx := context.Background()
		for path := range changes {
			// Figure out if the page still exists
			exists := true
			_, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					exists = false
				} else {
					// TODO: Log error
				}
			}

			var page Page

			if g.index != nil {
				// Indexing is enabled, update the index and retrieve the page
				var err error
				if exists {
					err = g.index.DeletePage(ctx, path)
				} else {
					page, err = g.indexDocument(ctx, path)
				}

				if err != nil {
					// TODO: Log error
				}

				// Sync after indexing so changes are visible
				g.index.Sync()
			} else if exists {
				// No indexing, open the page directly
				page, err = g.openViaPath(path)
				if err != nil {
					// TODO: Log error
				}
			}

			var event ChangeEvent
			if exists {
				if page != nil {
					event = &PageUpdated{
						Page: page,
					}
				}
			} else {
				event = g.createPageDeletedEvent(path)
			}

			// Notify watchers of the change
			if event != nil {
				var watchers []*Watcher
				g.mu.Lock()
				watchers = append(watchers, g.watchers...)
				g.mu.Unlock()

				for _, watcher := range watchers {
					watcher.changes <- event
				}
			}
		}
	}()
}

func (g *Graph) createPageDeletedEvent(path string) ChangeEvent {
	name := filepath.Base(path)
	name = name[:len(name)-3]

	dir := filepath.Dir(path)
	if dir == filepath.Join(g.directory, g.config.JournalsDir) {
		date, err := time.Parse(g.journalNameFormat, name)
		if err != nil {
			// Ignore files that don't match the journal name format
			return nil
		}

		return &PageDeleted{
			Type:  PageTypeJournal,
			Date:  date,
			Title: date.Format(g.journalTitleFormat),
		}
	} else if dir == filepath.Join(g.directory, g.config.PagesDir) {
		title, err := utils.FilenameToTitle(g.config.File.NameFormat, name)
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
			return &pageResultImpl{
				docType: PageTypeJournal,
				title:   page.Date.Format(g.journalTitleFormat),
				date:    page.Date,

				opener: func() (Page, error) {
					return source.OpenJournal(page.Date)
				},
			}
		} else {
			return &pageResultImpl{
				docType: PageTypeDedicated,
				title:   page.Title,
				date:    time.Time{},

				opener: func() (Page, error) {
					return source.OpenPage(page.Title)
				},
			}
		}
	}), nil
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

			pageDate, err = time.Parse(g.journalNameFormat, name)
			if err != nil {
				// TODO: This is an edge case where the format of journals has changed since indexing
			}

			pageTitle = pageDate.Format(g.journalTitleFormat)
		} else {
			pageTitle, err = utils.FilenameToTitle(g.config.File.NameFormat, name)
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

		if len(g.watchers) == 0 && g.changeWatcher != nil && g.index != nil {
			// Close the change watcher if there are no more watchers
			g.changeWatcher.Close()
			g.changeWatcher = nil
		}
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.watchers = append(g.watchers, watcher)

	if g.changeWatcher == nil {
		// Start watching for changes if we're not already
		g.watchForChanges()
	}

	return watcher
}
