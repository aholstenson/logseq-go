package logseq

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
)

type Transaction struct {
	graph *Graph

	openedPages map[string]Page

	// removedPaths are the files of pages that will be removed from the graph
	// when the transaction is saved.
	removedPaths []string
}

func newTransaction(graph *Graph) *Transaction {
	return &Transaction{
		graph:       graph,
		openedPages: make(map[string]Page),
	}
}

func (t *Transaction) OpenJournal(date time.Time) (Page, error) {
	// The date has to be normalized the same way the graph does it, or two
	// times on the same journal day would open the page twice.
	path, err := t.graph.journalPath(journalDate(date))
	if err != nil {
		return nil, err
	}

	page, ok := t.openedPages[path]
	if ok {
		return page, nil
	}

	page, err = t.graph.openJournal(date, t)
	if err != nil {
		return nil, err
	}

	t.openedPages[path] = page
	return page, nil
}

func (t *Transaction) OpenPage(title string) (Page, error) {
	path, err := t.graph.pagePath(title)
	if err != nil {
		return nil, err
	}

	page, ok := t.openedPages[path]
	if ok {
		return page, nil
	}

	page, err = t.graph.openPage(title, t)
	if err != nil {
		return nil, err
	}

	// Opening a page by one of its aliases gives the page the alias belongs to,
	// which is stored elsewhere and may already be part of this transaction.
	if impl, ok := page.(*pageImpl); ok {
		path = impl.path

		if opened, ok := t.openedPages[path]; ok {
			return opened, nil
		}
	}

	t.openedPages[path] = page
	return page, nil
}

// openViaPath opens the page stored at the given path, returning the instance
// already opened in this transaction if there is one. The page only becomes
// part of the transaction if the caller puts it in openedPages, so pages that
// end up unchanged are not written back.
func (t *Transaction) openViaPath(path string) (*pageImpl, error) {
	page, ok := t.openedPages[path]
	if !ok {
		var err error
		page, err = t.graph.openViaPath(path, t)
		if err != nil {
			return nil, err
		}

		if page == nil {
			return nil, nil
		}
	}

	impl, ok := page.(*pageImpl)
	if !ok {
		return nil, fmt.Errorf("unknown page type: %T", page)
	}

	return impl, nil
}

// DeletePage removes a page from the graph when the transaction is saved.
// Deleting a page that does not exist does nothing.
//
// References to the page are left as they are, the same way Logseq leaves them
// pointing at a page that no longer has any content.
//
// Any changes made to the page in this transaction are dropped, as writing them
// would only recreate the file that is being removed.
func (t *Transaction) DeletePage(title string) error {
	path, err := t.graph.pagePath(title)
	if err != nil {
		return err
	}

	t.deletePath(path)
	return nil
}

// DeleteJournal removes the journal for the given date from the graph when the
// transaction is saved. See DeletePage for the details of how pages are
// removed.
func (t *Transaction) DeleteJournal(date time.Time) error {
	path, err := t.graph.journalPath(journalDate(date))
	if err != nil {
		return err
	}

	t.deletePath(path)
	return nil
}

func (t *Transaction) deletePath(path string) {
	delete(t.openedPages, path)

	for _, removed := range t.removedPaths {
		if removed == path {
			return
		}
	}

	t.removedPaths = append(t.removedPaths, path)
}

// RenameOption is an option for renaming a page.
type RenameOption func(*renameOptions)

type renameOptions struct {
	namespaceChildren bool
}

// WithNamespaceChildren renames the pages in the namespace of a page along with
// it, so that renaming `Parent` also renames `Parent/Child` to `New/Child` and
// `Parent/Child/Grandchild` to `New/Child/Grandchild`. Without it those pages
// keep the titles they have, staying in the namespace of the old title.
//
// A namespace does not need a page of its own, so with this option the page
// being renamed is allowed to not exist as long as the namespace has pages in
// it.
func WithNamespaceChildren() RenameOption {
	return func(o *renameOptions) {
		o.namespaceChildren = true
	}
}

// RenamePage changes the title of a page. The page moves to the file its new
// title belongs in and every reference to it in the graph, such as `[[Old]]`,
// `#Old` and `{{embed [[Old]]}}`, is pointed at the new title. As with all
// other changes, this happens when the transaction is saved.
//
// Pages in the namespace of the page keep their titles unless
// WithNamespaceChildren is used.
//
// The references are found via the index, so this requires the graph to have
// been opened with indexing enabled. Pages that have changed on disk without
// having been indexed yet may keep references to the old title.
func (t *Transaction) RenamePage(ctx context.Context, from string, to string, opts ...RenameOption) error {
	if t.graph.index == nil {
		return fmt.Errorf("indexing is not enabled, which is required to rename pages")
	}

	options := &renameOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// The titles are collected before anything is renamed, as they are the
	// titles the pages have in the index.
	var children []string
	if options.namespaceChildren {
		var err error
		children, err = t.graph.pageTitlesUnderNamespace(ctx, from)
		if err != nil {
			return fmt.Errorf("failed to find the pages in the namespace %s: %w", from, err)
		}
	}

	// The new titles are checked before anything is renamed, so that one of
	// them being taken does not leave a namespace that is only half renamed.
	for _, child := range children {
		title, ok := namespaceRenameTarget(child, from, to)
		if !ok {
			continue
		}

		if err := t.checkRename(child, title); err != nil {
			return err
		}
	}

	err := t.renamePage(ctx, from, to)
	if err != nil {
		// A namespace does not need a page of its own, so the page not existing
		// is only a problem when there is nothing in the namespace either.
		if len(children) == 0 || !errors.Is(err, ErrPageNotFound) {
			return err
		}
	}

	for _, child := range children {
		title, ok := namespaceRenameTarget(child, from, to)
		if !ok {
			continue
		}

		// Pages that are in the index but no longer on disk have nothing to
		// rename, so they are left alone.
		if err := t.renamePage(ctx, child, title); err != nil && !errors.Is(err, ErrPageNotFound) {
			return err
		}
	}

	return nil
}

func (t *Transaction) renamePage(ctx context.Context, from string, to string) error {
	fromPath, err := t.graph.pagePath(from)
	if err != nil {
		return err
	}

	toPath, err := t.graph.pagePath(to)
	if err != nil {
		return err
	}

	page, err := t.OpenPage(from)
	if err != nil {
		return err
	}

	if page.IsNew() {
		return fmt.Errorf("%w: %s", ErrPageNotFound, from)
	}

	current := page.(*pageImpl)
	if current.path != fromPath {
		// The title belongs to another page, as an alias of it, so there is no
		// page of its own to rename here.
		return fmt.Errorf("%w: %s", ErrPageNotFound, from)
	}

	renamed := current
	if fromPath != toPath {
		if err := t.checkRenameTarget(fromPath, toPath, to); err != nil {
			return err
		}

		// The page keeps its content but is written to the file of the new
		// title, leaving the old file to be removed.
		renamed, err = openOrCreatePage(t, toPath, PageTypeDedicated, to, time.Time{}, "", t.graph.markdownParseOptions()...)
		if err != nil {
			return err
		}

		renamed.root = current.root

		t.deletePath(fromPath)
	} else {
		// Only the case of the title changed, so the page stays in its file.
		renamed.title = to
	}

	t.openedPages[toPath] = renamed

	// The renamed page may reference itself, and its content is now part of the
	// page at the new title, so it is retargeted here rather than as one of the
	// referencing pages below.
	retargetPageReferences(renamed.root, from, to)

	subPaths, err := t.graph.referencingSubPaths(ctx, from)
	if err != nil {
		return fmt.Errorf("failed to find references to %s: %w", from, err)
	}

	for _, subPath := range subPaths {
		path := filepath.Join(t.graph.directory, subPath)
		if path == fromPath || path == toPath {
			continue
		}

		referrer, err := t.openViaPath(path)
		if err != nil {
			return fmt.Errorf("failed to open referencing page %s: %w", subPath, err)
		}

		if referrer == nil {
			continue
		}

		if retargetPageReferences(referrer.root, from, to) > 0 {
			// Only pages that actually reference the renamed page are written.
			t.openedPages[path] = referrer
		}
	}

	return nil
}

// checkRename makes sure a page can be renamed, which needs its new title to be
// free. Titles without a page of their own are left to renaming to skip over.
func (t *Transaction) checkRename(from string, to string) error {
	fromPath, err := t.graph.pagePath(from)
	if err != nil {
		return err
	}

	toPath, err := t.graph.pagePath(to)
	if err != nil {
		return err
	}

	if fromPath == toPath {
		return nil
	}

	if _, err := os.Stat(fromPath); err != nil {
		return nil
	}

	return t.checkRenameTarget(fromPath, toPath, to)
}

// checkRenameTarget makes sure a page can be moved to the given path, which it
// can as long as no other page is stored there.
func (t *Transaction) checkRenameTarget(fromPath string, toPath string, to string) error {
	toInfo, err := os.Stat(toPath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check if page %s exists: %w", to, err)
	}

	// On a file system that ignores case a title that only changed in case maps
	// to a different path but the same file, in which case the page is simply
	// staying where it is.
	fromInfo, err := os.Stat(fromPath)
	if err == nil && os.SameFile(fromInfo, toInfo) {
		return nil
	}

	return fmt.Errorf("%w: %s", ErrPageExists, to)
}

func (t *Transaction) SearchPages(ctx context.Context, options ...SearchOption) (SearchResults[PageResult], error) {
	return t.graph.searchPages(ctx, options, t)
}

func (t *Transaction) SearchBlocks(ctx context.Context, options ...SearchOption) (SearchResults[BlockResult], error) {
	return t.graph.searchBlocks(ctx, options, t)
}

// OpenBlock opens the block with the given id, opening the page it belongs to
// as part of this transaction so that changes to the block are saved with it.
// See Graph.OpenBlock for the details of how blocks are found.
func (t *Transaction) OpenBlock(ctx context.Context, id string) (*content.Block, Page, error) {
	return t.graph.openBlock(ctx, id, t)
}

// AddJournalBlock adds a block to the journal page for the given date.
func (t *Transaction) AddJournalBlock(time time.Time, block *content.Block) error {
	// Change the timezone to the local one
	time = time.Local()

	page, err := t.OpenJournal(time)
	if err != nil {
		return err
	}

	timeFormat := t.graph.options.blockTimeFormat

	// Go through all the blocks on the page and figure out where we fit in
	var insertAfter *content.Block
	for _, b := range page.Blocks() {
		t := parseBlockTime(timeFormat, time, b)
		if t != nil && t.After(time) {
			break
		}

		if b.FirstChild() != nil {
			insertAfter = b
		}
	}

	if timeFormat != "" {
		// Add the timestamp to the block
		timeNode := t.graph.options.blockTimeFormatToNode(time.Format(timeFormat))
		firstChild := block.FirstChild()

		if _, ok := firstChild.(*content.Properties); ok {
			// Skip properties block
			firstChild = firstChild.NextSibling()
		}

		if p, ok := firstChild.(*content.Paragraph); ok {
			p.PrependChild(timeNode)
			p.InsertChildAfter(content.NewText(" "), timeNode)
		} else {
			block.PrependChild(content.NewParagraph(timeNode, content.NewText(" ")))
		}
	}

	if insertAfter == nil {
		// All blocks have timestamps after the new block, prepend it
		page.PrependBlock(block)
	} else {
		// Insert the block after the block with the timestamp before the new
		// block, or at the end of the page if there are no timestamps
		page.InsertBlockAfter(block, insertAfter)
	}

	return nil
}

func parseBlockTime(format string, reference time.Time, block *content.Block) *time.Time {
	firstParagraph := block.Children().FindDeep(content.IsOfType[*content.Paragraph]())
	if firstParagraph == nil {
		return nil
	}

	firstText := firstParagraph.(*content.Paragraph).Children().FindDeep(content.IsOfType[*content.Text]())
	if firstText == nil {
		return nil
	}

	// The first text node should be the timestamp
	text := firstText.(*content.Text)
	if text == nil {
		return nil
	}

	t, err := time.Parse(format, text.Value)
	if err != nil {
		return nil
	}

	// Combine the date and time
	t = time.Date(reference.Year(), reference.Month(), reference.Day(), t.Hour(), t.Minute(), 0, 0, reference.Location())
	return &t
}

func (t *Transaction) Save() error {
	// Pages are written to the path they were opened from, which is the only
	// place they can be written back to without moving them.
	pages := make([]*pageImpl, 0, len(t.openedPages))
	for _, page := range t.openedPages {
		impl, ok := page.(*pageImpl)
		if !ok {
			return fmt.Errorf("unknown page type: %T", page)
		}

		pages = append(pages, impl)
	}

	for _, page := range pages {
		path := page.path

		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			if !page.IsNew() {
				return fmt.Errorf("page at %s no longer exists", path)
			}

			continue
		} else if err != nil {
			return fmt.Errorf("failed to check if page can be saved at %s: %w", path, err)
		}

		if info.IsDir() {
			return fmt.Errorf("page at %s is a directory", path)
		}

		// Check that the page has not been modified since it was opened
		if info.ModTime() != page.LastModified() {
			return fmt.Errorf("page at %s has been modified since it was opened", path)
		}
	}

	// written keeps track of the files pages were written to, so that removing
	// a page can tell if a page was written to the same file. A rename that only
	// changes the case of a title ends up doing that on a file system that
	// ignores case.
	written := make([]os.FileInfo, 0, len(pages))

	for _, page := range pages {
		path := page.path

		data, err := markdown.AsString(page.root, t.graph.markdownOptions()...)
		if err != nil {
			if page.Type() == PageTypeJournal {
				return fmt.Errorf("failed to convert journal %s: %w", page.Date().Format("2006-01-02"), err)
			} else {
				return fmt.Errorf("failed to convert page %s: %w", page.Title(), err)
			}
		}

		// Pages are line based, so make sure the file ends with a newline the
		// same way Logseq and most editors write them. AsString is left alone
		// so that it stays usable for fragments that should not gain one.
		if data != "" && !strings.HasSuffix(data, "\n") {
			data += "\n"
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", path, err)
		}

		err = os.WriteFile(path, []byte(data), 0644)
		if err != nil {
			return fmt.Errorf("failed to write page to %s: %w", path, err)
		}

		if info, err := os.Stat(path); err == nil {
			written = append(written, info)
		}
	}

	// Removals happen after the writes so that a page moved into the file of a
	// removed page keeps its content.
	for _, path := range t.removedPaths {
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return fmt.Errorf("failed to check if page at %s can be removed: %w", path, err)
		}

		if wasWritten(written, info) {
			continue
		}

		if err := t.graph.removePageFile(path); err != nil {
			return err
		}
	}

	t.removedPaths = nil
	return nil
}

func wasWritten(written []os.FileInfo, info os.FileInfo) bool {
	for _, w := range written {
		if os.SameFile(w, info) {
			return true
		}
	}

	return false
}
