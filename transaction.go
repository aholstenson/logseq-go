package logseq

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
)

type Transaction struct {
	graph *Graph

	openedPages map[string]Page
}

func newTransaction(graph *Graph) *Transaction {
	return &Transaction{
		graph:       graph,
		openedPages: make(map[string]Page),
	}
}

func (t *Transaction) OpenJournal(date time.Time) (Page, error) {
	path, err := t.graph.journalPath(date)
	if err != nil {
		return nil, err
	}

	page, ok := t.openedPages[path].(Page)
	if ok {
		return page, nil
	}

	page, err = t.graph.OpenJournal(date)
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

	page, ok := t.openedPages[path].(Page)
	if ok {
		return page, nil
	}

	page, err = t.graph.OpenPage(title)
	if err != nil {
		return nil, err
	}

	t.openedPages[path] = page
	return page, nil
}

func (t *Transaction) SearchPages(ctx context.Context, options ...SearchOption) (SearchResults[PageResult], error) {
	return t.graph.searchPages(ctx, options, t)
}

func (t *Transaction) SearchBlocks(ctx context.Context, options ...SearchOption) (SearchResults[BlockResult], error) {
	return t.graph.searchBlocks(ctx, options, t)
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
	for path, page := range t.openedPages {
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

	for path, page := range t.openedPages {
		var root *content.Block
		if j, ok := page.(*pageImpl); ok {
			root = j.root
		} else {
			return fmt.Errorf("unknown page type: %T", page)
		}

		data, err := markdown.AsString(root)
		if err != nil {
			if page.Type() == PageTypeJournal {
				return fmt.Errorf("failed to convert journal %s: %w", page.Date().Format("2006-01-02"), err)
			} else {
				return fmt.Errorf("failed to convert page %s: %w", page.Title(), err)
			}
		}

		err = os.WriteFile(path, []byte(data), 0644)
		if err != nil {
			return fmt.Errorf("failed to write page to %s: %w", path, err)
		}
	}

	return nil
}
