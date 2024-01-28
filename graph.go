package logseq

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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

func (g *Graph) Directory() string {
	return g.directory
}

func (g *Graph) NewTransaction() *Transaction {
	return newTransaction(g)
}

// Journal returns a read-only version of the journal page for the given date.
func (g *Graph) OpenJournalPage(date time.Time) (*JournalPage, error) {
	path, err := g.journalPath(date)
	if err != nil {
		return nil, err
	}

	templatePath := filepath.Join(g.directory, g.config.DefaultTemplates.Journals)

	pageImpl, err := newPage(path, templatePath)
	if err != nil {
		return nil, err
	}

	return &JournalPage{
		pageImpl: *pageImpl,
		date:     date,
	}, nil
}

func (g *Graph) journalPath(date time.Time) (string, error) {
	filename := date.Format(g.journalNameFormat) + ".md"
	return filepath.Join(g.directory, g.config.JournalsDir, filename), nil
}

// Page returns a read-only version of a page for the given path.
func (g *Graph) OpenPage(title string) (*NotePage, error) {
	path, err := g.pagePath(title)
	if err != nil {
		return nil, err
	}

	pageImpl, err := newPage(path, "")
	if err != nil {
		return nil, err
	}

	return &NotePage{
		pageImpl: *pageImpl,
		title:    title,
	}, nil
}

func (g *Graph) pagePath(title string) (string, error) {
	path, err := utils.TitleToFilename(g.config.File.NameFormat, title)
	if err != nil {
		return "", err
	}

	return filepath.Join(g.directory, path+".md"), nil
}
