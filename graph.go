package logseq

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
	"github.com/aholstenson/logseq-go/internal/utils"
	"golang.org/x/net/context"
)

// Graph represents a Logseq graph. In Logseq a graph is a directory that
// contains Markdown files for pages and journals.
type Graph struct {
	options *options

	directory string

	config *utils.GraphConfig

	journalNameFormat  string
	journalTitleFormat string

	index indexing.Index
}

func Open(directory string, opts ...Option) (*Graph, error) {
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
	}

	// Sync the graph with the index
	err = g.sync(options.syncListener)
	if err != nil {
		return nil, fmt.Errorf("failed to sync graph: %w", err)
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
func (g *Graph) OpenJournal(date time.Time) (*Journal, error) {
	path, err := g.journalPath(date)
	if err != nil {
		return nil, err
	}

	templatePath := ""
	if g.config.DefaultTemplates.Journals != "" {
		templatePath = filepath.Join(g.directory, g.config.DefaultTemplates.Journals)
	}

	pageImpl, err := openOrCreateDocument(path, templatePath)
	if err != nil {
		return nil, err
	}

	title := date.Format(g.journalTitleFormat)

	return &Journal{
		documentImpl: *pageImpl,
		title:        title,
		date:         date,
	}, nil
}

func (g *Graph) journalPath(date time.Time) (string, error) {
	filename := date.Format(g.journalNameFormat) + ".md"
	return filepath.Join(g.directory, g.config.JournalsDir, filename), nil
}

// Page returns a read-only version of a page for the given path.
func (g *Graph) OpenPage(title string) (*Page, error) {
	path, err := g.pagePath(title)
	if err != nil {
		return nil, err
	}

	pageImpl, err := openOrCreateDocument(path, "")
	if err != nil {
		return nil, err
	}

	return &Page{
		documentImpl: *pageImpl,
		title:        title,
	}, nil
}

func (g *Graph) pagePath(title string) (string, error) {
	path, err := utils.TitleToFilename(g.config.File.NameFormat, title)
	if err != nil {
		return "", err
	}

	return filepath.Join(g.directory, g.config.PagesDir, path+".md"), nil
}

func (g *Graph) Close() error {
	if g.index != nil {
		return g.index.Close()
	}

	// TODO: Close fsnotify watcher

	return nil
}

// sync performs a sync of the graph with the index.
func (g *Graph) sync(syncListener func(subPath string)) error {
	if g.index == nil {
		return nil
	}

	ctx := context.Background()

	// Sync the journal pages
	journalsDir := filepath.Join(g.directory, g.config.JournalsDir)
	err := filepath.Walk(journalsDir, g.createWalker(ctx, func(name string, doc *documentImpl) *indexing.Document {
		date, err := time.Parse(g.journalNameFormat, name)
		if err != nil {
			return nil
		}

		return &indexing.Document{
			Type:         indexing.DocumentTypeJournal,
			LastModified: doc.LastModified(),
			Date:         date,
			Blocks:       doc.Blocks(),
		}
	}))
	if err != nil {
		return fmt.Errorf("failed to sync journals: %w", err)
	}

	// Sync the note pages
	notesDir := filepath.Join(g.directory, g.config.PagesDir)
	err = filepath.Walk(notesDir, g.createWalker(ctx, func(name string, doc *documentImpl) *indexing.Document {
		title, err := utils.FilenameToTitle(g.config.File.NameFormat, name)
		if err != nil {
			return nil
		}

		return &indexing.Document{
			Type:         indexing.DocumentTypePage,
			LastModified: doc.LastModified(),
			Title:        title,
			Blocks:       doc.Blocks(),
		}
	}))
	if err != nil {
		return fmt.Errorf("failed to sync pages: %w", err)
	}

	g.index.Sync()
	return nil
}

func (g *Graph) createWalker(ctx context.Context, mapper func(name string, doc *documentImpl) *indexing.Document) filepath.WalkFunc {
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

		pageImpl, err := openOrCreateDocument(path, "")
		if err != nil {
			return fmt.Errorf("failed to open page: %w", err)
		}

		name := filepath.Base(path)
		name = name[:len(name)-3]

		doc := mapper(name, pageImpl)
		if doc == nil {
			return nil
		}

		err = g.index.IndexDocument(ctx, doc)
		if err != nil {
			return err
		}

		if g.options.syncListener != nil {
			g.options.syncListener(subPath)
		}
		return nil
	}
}

func (g *Graph) List(ctx context.Context, query Query) (DocumentIterator[Document], error) {
	if g.index == nil {
		return nil, fmt.Errorf("indexing is not enabled")
	}

	iter, err := g.index.ListDocuments(ctx, query)
	if err != nil {
		return nil, err
	}

	return &documentIterator[Document]{
		iterator: iter,
		mapper: func(doc *indexing.Document) DocumentMetadata[Document] {
			if doc.Type == indexing.DocumentTypeJournal {
				return &documentMetadataImpl[Document]{
					graph: g,

					docType: DocumentTypeJournal,
					title:   doc.Date.Format(g.journalTitleFormat),
					date:    doc.Date,

					opener: func() (Document, error) {
						return g.OpenJournal(doc.Date)
					},
				}
			} else {
				return &documentMetadataImpl[Document]{
					graph: g,

					docType: DocumentTypePage,
					title:   doc.Title,
					date:    time.Time{},

					opener: func() (Document, error) {
						return g.OpenPage(doc.Title)
					},
				}
			}
		},
	}, nil
}

type documentIterator[D Document] struct {
	iterator indexing.Iterator[*indexing.Document]
	mapper   func(*indexing.Document) DocumentMetadata[D]
}

func (i *documentIterator[D]) Next() (DocumentMetadata[D], error) {
	doc, err := i.iterator.Next()
	if err != nil {
		return nil, err
	}

	if doc == nil {
		return nil, nil
	}

	return i.mapper(doc), nil
}
