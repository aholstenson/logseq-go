package logseq

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	"github.com/aholstenson/logseq-go/internal/utils"
)

// Graph represents a Logseq graph. In Logseq a graph is a directory that
// contains Markdown files for pages and journals.
type Graph struct {
	directory string

	config            *utils.GraphConfig
	journalNameFormat string
}

func Open(directory string) (*Graph, error) {
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

	return &Graph{
		directory:         directory,
		config:            config,
		journalNameFormat: journalNameFormat,
	}, nil
}

// Journal returns the journal for the given date.
func (g *Graph) Journal(date time.Time) (*Page, error) {
	filename := date.Format(g.journalNameFormat) + ".md"
	path := filepath.Join(g.directory, g.config.JournalsDir, filename)
	log.Println(filename)

	page, err := g.openExisting(path)
	if os.IsNotExist(err) {
		// TODO: Create empty journal
		if g.config.DefaultTemplates.Journals == "" {
			// No template, create empty journal
			return &Page{
				path:  path,
				root:  content.NewBlock(),
				isNew: true,
			}, nil
		} else {
			// Load the template
			templatePath := filepath.Join(g.directory, g.config.DefaultTemplates.Journals)
			template, err := g.openExisting(templatePath)
			if err != nil {
				return nil, fmt.Errorf("failed to load template: %w", err)
			}

			return &Page{
				path:  path,
				root:  template.root,
				isNew: true,
			}, nil
		}
	}

	return page, err
}

// Page returns the page for the given path.
func (g *Graph) Page(title string) (*Page, error) {
	path, err := utils.TitleToFilename(g.config.File.NameFormat, title)
	if err != nil {
		return nil, err
	}

	path = filepath.Join(g.directory, path+".md")

	page, err := g.openExisting(path)
	if os.IsNotExist(err) {
		return &Page{
			path:  path,
			root:  content.NewBlock(),
			isNew: true,
		}, nil
	}

	return page, err
}

func (g *Graph) openExisting(path string) (*Page, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// If the file does not exist we return an empty page.
		if os.IsNotExist(err) {
			// TODO: Create empty page
		}

		return nil, err
	}

	block, err := markdown.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Check if the block has content, in which case we wrap it
	if len(block.Content()) > 0 {
		block = content.NewBlock(
			block,
		)
	}

	return &Page{
		path: path,
		root: block,
	}, nil
}
