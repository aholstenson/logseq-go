package logseq

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/utils"
)

// Workflow is the pair of task markers that a graph uses for its tasks, set
// via `:preferred-workflow` in the config of the graph.
type Workflow int

const (
	// WorkflowNow is the LATER/NOW workflow, which is what Logseq uses for
	// graphs that do not pick a workflow.
	WorkflowNow Workflow = iota

	// WorkflowTodo is the TODO/DOING workflow.
	WorkflowTodo
)

// Todo is the status of a task that has not been started.
func (w Workflow) Todo() content.TaskStatus {
	if w == WorkflowTodo {
		return content.TaskStatusTodo
	}

	return content.TaskStatusLater
}

// Doing is the status of a task that is being worked on.
func (w Workflow) Doing() content.TaskStatus {
	if w == WorkflowTodo {
		return content.TaskStatusDoing
	}

	return content.TaskStatusNow
}

// Workflow returns the task markers the graph uses, so that tasks created in
// it match the ones Logseq creates:
//
//	block.PrependChild(content.NewTaskMarker(graph.Workflow().Todo()))
func (g *Graph) Workflow() Workflow {
	if g.config.PreferredWorkflow == utils.PreferredWorkflowTodo {
		return WorkflowTodo
	}

	return WorkflowNow
}

// Favorites returns the titles of the pages that are favorited in the graph,
// in the order Logseq lists them in its sidebar.
func (g *Graph) Favorites() []string {
	favorites := make([]string, len(g.config.Favorites))
	copy(favorites, g.config.Favorites)
	return favorites
}

// IsHidden checks if a path is hidden in the graph, which is how Logseq keeps
// files and directories out of a graph without moving them elsewhere. The path
// is relative to the directory of the graph.
//
// Hidden pages are left out when the graph is indexed and changes to them are
// not reported to watchers.
func (g *Graph) IsHidden(path string) bool {
	return g.config.IsHidden(path)
}

// LogbookSettings is how a graph records and displays the logbooks of tasks.
type LogbookSettings struct {
	// WithSeconds is whether the times in clock entries include seconds.
	WithSeconds bool

	// EnabledInAllBlocks is whether logbooks are shown on every block instead
	// of only on the blocks that have time tracked on them.
	EnabledInAllBlocks bool

	// EnabledInTimestampedBlocks is whether logbooks are shown on the blocks
	// that have time tracked on them. When this is off Logseq does not show
	// logbooks at all.
	EnabledInTimestampedBlocks bool
}

// LogbookSettings returns how the graph records and displays logbooks, as set
// via `:logbook/settings`.
func (g *Graph) LogbookSettings() LogbookSettings {
	return LogbookSettings{
		WithSeconds:                g.config.Logbook.WithSecondSupport,
		EnabledInAllBlocks:         g.config.Logbook.EnabledInAllBlocks,
		EnabledInTimestampedBlocks: g.config.Logbook.EnabledInTimestampedBlocks,
	}
}

// DefaultQuery is a query that Logseq shows in addition to the content of a
// page. The parts of it that are Datalog or Clojure are kept as the EDN they
// were written as, as this library does not run queries.
type DefaultQuery struct {
	// Title is what is shown above the results of the query. Queries with a
	// title written as Hiccup markup have an empty title here, with the markup
	// available in TitleEDN.
	Title string

	// TitleEDN is the title exactly as it appears in the config.
	TitleEDN string

	// Query is the Datalog query to run.
	Query string

	// Inputs are the values bound to the `:in` arguments of Query, such as
	// `:today` or `:14d`.
	Inputs string

	// ResultTransform is a function applied to the results before they are
	// shown.
	ResultTransform string

	// GroupByPage is whether the results are grouped by the page they are on.
	// Nil when the query does not say, in which case Logseq decides based on
	// the rest of the query.
	GroupByPage *bool

	// Collapsed is whether the query starts out collapsed.
	Collapsed bool
}

// DefaultJournalQueries returns the queries that Logseq shows at the bottom of
// the journal page for today, as set via `:default-queries`.
func (g *Graph) DefaultJournalQueries() []DefaultQuery {
	queries := make([]DefaultQuery, 0, len(g.config.DefaultQueries.Journals))
	for _, query := range g.config.DefaultQueries.Journals {
		queries = append(queries, DefaultQuery{
			Title:           query.Title.Text,
			TitleEDN:        string(query.Title.Raw),
			Query:           string(query.Query),
			Inputs:          string(query.Inputs),
			ResultTransform: string(query.ResultTransform),
			GroupByPage:     query.GroupByPage,
			Collapsed:       query.Collapsed,
		})
	}

	return queries
}
