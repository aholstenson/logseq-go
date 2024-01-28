package logseq

import (
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

// Journal creates a new journal helper to simplify adding blocks to one or
// more journal pages.
func (t *Transaction) Journal(opts ...JournalOption) *Journal {
	return newJournal(t, opts...)
}

func (t *Transaction) OpenJournalPage(date time.Time) (*JournalPage, error) {
	path, err := t.graph.journalPath(date)
	if err != nil {
		return nil, err
	}

	page, ok := t.openedPages[path].(*JournalPage)
	if ok {
		return page, nil
	}

	page, err = t.graph.OpenJournalPage(date)
	if err != nil {
		return nil, err
	}

	t.openedPages[path] = page
	return page, nil
}

func (t *Transaction) OpenPage(title string) (*NotePage, error) {
	path, err := t.graph.pagePath(title)
	if err != nil {
		return nil, err
	}

	page, ok := t.openedPages[path].(*NotePage)
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
		if j, ok := page.(*JournalPage); ok {
			root = j.root
		} else if n, ok := page.(*NotePage); ok {
			root = n.root
		} else {
			return fmt.Errorf("unknown page type: %T", page)
		}

		data, err := markdown.AsString(root)
		if err != nil {
			if j, ok := page.(*JournalPage); ok {
				return fmt.Errorf("failed to convert journal %s: %w", j.date.Format("2006-01-02"), err)
			} else if n, ok := page.(*NotePage); ok {
				return fmt.Errorf("failed to convert page %s: %w", n.title, err)
			}
		}

		err = os.WriteFile(path, []byte(data), 0644)
		if err != nil {
			return fmt.Errorf("failed to write page to %s: %w", path, err)
		}
	}

	return nil
}
