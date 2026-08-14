package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"olympos.io/encoding/edn"
)

// GraphConfig is the configuration of a graph, as read from its
// `logseq/config.edn`. Keys are matched by their full name, so keys in a
// namespace are tagged as `namespace/key`.
type GraphConfig struct {
	JournalsDir string `edn:"journals-directory"`

	// JournalPageTitleFormat is how the date of a journal is turned into the
	// title of its page.
	JournalPageTitleFormat string `edn:"journal/page-title-format"`

	// DateFormatter is what JournalPageTitleFormat was called before Logseq
	// moved it into the `journal` namespace. It is only used when
	// JournalPageTitleFormat is not set.
	DateFormatter string `edn:"date-formatter"`

	// JournalFileNameFormat is how the date of a journal is turned into the
	// name of its file.
	JournalFileNameFormat string `edn:"journal/file-name-format"`

	PagesDir string `edn:"pages-directory"`

	// FileNameFormat is how the title of a page is turned into the name of
	// its file.
	FileNameFormat FilenameFormat `edn:"file/name-format"`

	DefaultTemplates DefaultTemplates `edn:"default-templates"`

	// PropertiesSeparatedByCommas are the properties whose values Logseq
	// reads as a comma separated list.
	PropertiesSeparatedByCommas KeywordList `edn:"property/separated-by-commas"`

	IgnoredPageReferencesKeywords KeywordList `edn:"ignored-page-references-keywords"`

	// PreferredWorkflow is the pair of task markers the graph is set up for.
	PreferredWorkflow PreferredWorkflow `edn:"preferred-workflow"`

	// PreferredFormat is the markup that pages in the graph are written in.
	PreferredFormat PreferredFormat `edn:"preferred-format"`

	// Favorites are the titles of the pages that Logseq lists in its left
	// sidebar.
	Favorites []string `edn:"favorites"`

	// Hidden are the files and directories that Logseq leaves out of the
	// graph, such as `/archived` or `../assets/archived`. Paths that start
	// with a slash are relative to the root of the graph.
	Hidden []string `edn:"hidden"`

	// DefaultQueries are the queries that Logseq shows at the bottom of
	// journal pages.
	DefaultQueries DefaultQueries `edn:"default-queries"`

	// Logbook is how logbooks are recorded and displayed in the graph.
	Logbook LogbookConfig `edn:"logbook/settings"`
}

type DefaultTemplates struct {
	Journals string `edn:"journals"`
}

// KeywordList is a list of names that Logseq writes as a set of keywords,
// such as `#{:author :website}`, and that may also be written as strings.
type KeywordList []string

func (l *KeywordList) UnmarshalEDN(data []byte) error {
	var values []edn.RawMessage
	err := edn.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	list := make(KeywordList, 0, len(values))
	for _, value := range values {
		name, err := ednName(value)
		if err != nil {
			return err
		}

		list = append(list, name)
	}

	*l = list
	return nil
}

// PreferredWorkflow is the pair of task markers that Logseq cycles through
// when working with tasks in a graph.
type PreferredWorkflow string

const (
	// PreferredWorkflowNow is the LATER/NOW workflow, which is what Logseq
	// uses for graphs that do not pick a workflow.
	PreferredWorkflowNow PreferredWorkflow = "now"

	// PreferredWorkflowTodo is the TODO/DOING workflow.
	PreferredWorkflowTodo PreferredWorkflow = "todo"
)

// UnmarshalEDN reads the workflow, which is written as a keyword such as
// `:now`. Following what Logseq does, a value that mentions `now` is the
// LATER/NOW workflow and any other value is TODO/DOING.
func (w *PreferredWorkflow) UnmarshalEDN(data []byte) error {
	name, err := ednName(data)
	if err != nil {
		return fmt.Errorf("failed to read preferred workflow: %w", err)
	}

	if name == "" || strings.Contains(strings.ToLower(name), "now") {
		*w = PreferredWorkflowNow
	} else {
		*w = PreferredWorkflowTodo
	}

	return nil
}

// PreferredFormat is the markup that the pages of a graph are written in.
type PreferredFormat string

const (
	// PreferredFormatMarkdown is used by graphs written in Markdown, which is
	// what Logseq uses for graphs that do not pick a format.
	PreferredFormatMarkdown PreferredFormat = "markdown"

	// PreferredFormatOrg is used by graphs written in Org mode.
	PreferredFormatOrg PreferredFormat = "org"
)

// UnmarshalEDN reads the format, which is written either as a string such as
// `"Markdown"` or as a keyword such as `:markdown`. As Logseq does, the value
// is lower-cased and formats other than the known ones are kept as they were
// written.
func (f *PreferredFormat) UnmarshalEDN(data []byte) error {
	name, err := ednName(data)
	if err != nil {
		return fmt.Errorf("failed to read preferred format: %w", err)
	}

	if name == "" {
		*f = PreferredFormatMarkdown
	} else {
		*f = PreferredFormat(strings.ToLower(name))
	}

	return nil
}

// DefaultQueries are the queries that Logseq shows in addition to the content
// of a page.
type DefaultQueries struct {
	// Journals are the queries shown at the bottom of the journal page for
	// today.
	Journals []DefaultQuery `edn:"journals"`
}

// DefaultQuery is a single query in DefaultQueries. The parts of it that are
// Datalog or Clojure are kept as the EDN they were written as, as this library
// does not run queries.
type DefaultQuery struct {
	// Title is what is shown above the results of the query.
	Title QueryTitle `edn:"title"`

	// Query is the Datalog query to run.
	Query edn.RawMessage `edn:"query"`

	// Inputs are the values bound to the `:in` arguments of Query, such as
	// `:today` or `:14d`.
	Inputs edn.RawMessage `edn:"inputs"`

	// ResultTransform is a function applied to the results before they are
	// shown.
	ResultTransform edn.RawMessage `edn:"result-transform"`

	// GroupByPage is whether the results are grouped by the page they are on.
	// Nil when the query does not say, in which case Logseq decides based on
	// the rest of the query.
	GroupByPage *bool `edn:"group-by-page?"`

	// Collapsed is whether the query starts out collapsed.
	Collapsed bool `edn:"collapsed?"`
}

// QueryTitle is the title of a query. Logseq accepts either a string or Hiccup
// markup, such as `[:b "NOW"]`, so the EDN is kept as it was written and Text
// is only filled in for titles written as a string.
type QueryTitle struct {
	// Text is the title of the query if it was written as a string, and an
	// empty string for other titles.
	Text string

	// Raw is the title exactly as it appears in the config.
	Raw edn.RawMessage
}

func (t *QueryTitle) UnmarshalEDN(data []byte) error {
	t.Raw = make(edn.RawMessage, len(data))
	copy(t.Raw, data)

	t.Text = ""
	var text string
	if err := edn.Unmarshal(data, &text); err == nil {
		t.Text = text
	}

	return nil
}

// LogbookConfig is how the logbooks of tasks are recorded and displayed.
type LogbookConfig struct {
	// WithSecondSupport is whether the times in clock entries include
	// seconds.
	WithSecondSupport bool `edn:"with-second-support?"`

	// EnabledInAllBlocks is whether logbooks are shown on every block instead
	// of only on the blocks that have time tracked on them.
	EnabledInAllBlocks bool `edn:"enabled-in-all-blocks"`

	// EnabledInTimestampedBlocks is whether logbooks are shown on the blocks
	// that have time tracked on them. When this is off Logseq does not show
	// logbooks at all.
	EnabledInTimestampedBlocks bool `edn:"enabled-in-timestamped-blocks"`
}

func ParseConfig(data []byte) (*GraphConfig, error) {
	config := GraphConfig{
		JournalsDir:           "journals",
		JournalFileNameFormat: "yyyy_MM_dd",
		PagesDir:              "pages",
		FileNameFormat:        FilenameFormatTripleLowbar,
		PreferredWorkflow:     PreferredWorkflowNow,
		PreferredFormat:       PreferredFormatMarkdown,
		Logbook: LogbookConfig{
			WithSecondSupport:          true,
			EnabledInAllBlocks:         false,
			EnabledInTimestampedBlocks: true,
		},
	}

	err := edn.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to read EDN: %w", err)
	}

	if config.JournalPageTitleFormat == "" {
		// Graphs that do not set the format may still use the key it had
		// before it moved into the `journal` namespace.
		config.JournalPageTitleFormat = config.DateFormatter
	}

	if config.JournalPageTitleFormat == "" {
		config.JournalPageTitleFormat = "MMM do, yyyy"
	}

	return &config, nil
}

// IsHidden checks if a path is one that Logseq leaves out of the graph, as set
// by `:hidden`. The path is relative to the root of the graph.
//
// As in Logseq a pattern matches the start of a path, so `logseq/bak` hides
// everything in that directory.
func (c *GraphConfig) IsHidden(path string) bool {
	if len(c.Hidden) == 0 {
		return false
	}

	path = "/" + strings.TrimPrefix(filepath.ToSlash(path), "/")

	for _, pattern := range c.Hidden {
		if !strings.HasPrefix(pattern, "/") {
			pattern = "/" + pattern
		}

		if strings.HasPrefix(path, pattern) {
			return true
		}
	}

	return false
}

// ednName reads a value that Logseq writes as either a keyword or a string,
// returning it without the colon that a keyword starts with. `nil` becomes an
// empty string, which callers treat as the value not being set.
func ednName(data []byte) (string, error) {
	var value any
	err := edn.Unmarshal(data, &value)
	if err != nil {
		return "", err
	}

	switch v := value.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case edn.Keyword:
		return string(v), nil
	case edn.Symbol:
		return string(v), nil
	}

	return "", fmt.Errorf("expected a keyword or a string, got: %s", string(data))
}
