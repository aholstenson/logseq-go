package logseq

import (
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
	"github.com/aholstenson/logseq-go/internal/utils"
	"github.com/aholstenson/logseq-go/query"
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
//	block.SetTaskStatus(graph.Workflow().Todo())
func (g *Graph) Workflow() Workflow {
	if g.config.PreferredWorkflow == utils.PreferredWorkflowTodo {
		return WorkflowTodo
	}

	return WorkflowNow
}

// TimeTracking returns true if the graph tracks the time spent on tasks, as
// set via `:feature/enable-timetracking?`. Logseq tracks time unless the graph
// turns it off.
func (g *Graph) TimeTracking() bool {
	return g.config.EnableTimeTracking
}

// SetTaskStatus sets the status of the task in a block in the same way as
// Logseq does in this graph. When the graph tracks time this also starts or
// stops the clock of the task, see content.Block.SetTaskStatusWithClock.
// Otherwise only the status changes, see content.Block.SetTaskStatus.
func (g *Graph) SetTaskStatus(block *content.Block, status content.TaskStatus, at time.Time) {
	if g.TimeTracking() {
		block.SetTaskStatusWithClock(status, at)
		return
	}

	block.SetTaskStatus(status)
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
	// WithSeconds is whether the times in logbook entries include seconds.
	// Saving a page in the graph writes its logbooks this way.
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
// were written as. Use Parse to translate the query and Graph.Query to run it.
type DefaultQuery struct {
	// EDN is the map of the query exactly as it appears in the config, which
	// is the same map as an advanced query written on a page.
	EDN string

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

	// resultTransforms are the transforms of the config, for Parse.
	resultTransforms map[string]string
}

// Parse translates the query with query.ParseAdvanced, the same as an
// advanced query written on a page. The options are given to
// query.ParseAdvanced. The title of the result is the text of the title
// without Hiccup markup, also when the title is written as Hiccup.
//
// The result has the same fields as a query written in a block, such as
// GroupByPage and BreadcrumbShow. A default query is not in a block, so the
// block properties for the table, such as `query-table::`, do not apply.
// Logseq shows the journal queries on the journal page for today, so give its
// title with query.WithCurrentPage if the query uses `:current-page`. The
// transforms of the config are given with query.WithResultTransforms.
//
// Run the result with Graph.Query. Relative dates such as `:today` are
// relative to the time the query runs at, see QueryAt:
//
//	advanced, err := defaultQuery.Parse()
//	...
//	results, err := graph.Query(ctx, advanced.Query)
func (q DefaultQuery) Parse(opts ...query.ParseOption) (*query.AdvancedQuery, error) {
	opts = append([]query.ParseOption{query.WithResultTransforms(q.resultTransforms)}, opts...)
	return query.ParseAdvanced(q.EDN, opts...)
}

// QueryResultTransforms returns the functions that a :result-transform of a
// query can name with a keyword, as set via `:query/result-transforms`. The
// keys are the names of the keywords without the colon, and the values are
// the functions as EDN text. Give them to query.WithResultTransforms.
func (g *Graph) QueryResultTransforms() map[string]string {
	transforms := make(map[string]string, len(g.config.QueryResultTransforms))
	for name, fn := range g.config.QueryResultTransforms {
		transforms[string(name)] = string(fn)
	}

	return transforms
}

// DefaultJournalQueries returns the queries that Logseq shows at the bottom of
// the journal page for today, as set via `:default-queries`. Logseq does not
// show a query from this list when it has no results.
func (g *Graph) DefaultJournalQueries() []DefaultQuery {
	transforms := g.QueryResultTransforms()
	queries := make([]DefaultQuery, 0, len(g.config.DefaultQueries.Journals))
	for _, q := range g.config.DefaultQueries.Journals {
		queries = append(queries, DefaultQuery{
			EDN:             string(q.Raw),
			Title:           q.Title.Text,
			TitleEDN:        string(q.Title.Raw),
			Query:           string(q.Query),
			Inputs:          string(q.Inputs),
			ResultTransform: string(q.ResultTransform),
			GroupByPage:     q.GroupByPage,
			Collapsed:       q.Collapsed,

			resultTransforms: transforms,
		})
	}

	return queries
}

// Config changes settings of the graph for this library only, on top of the
// settings in `logseq/config.edn`. Use it with WithConfig when a program needs
// the graph to be read in another way than Logseq reads it, without changing
// the config of the graph for Logseq.
//
// A field that is not set keeps the value from `config.edn`, or the default of
// Logseq when `config.edn` does not set it. A nil slice is not set, while an
// empty slice that is not nil replaces the value with an empty list.
type Config struct {
	// PropertiesSeparatedByCommas are the properties whose value is a list of
	// pages separated by commas. It replaces `:property/separated-by-commas`.
	// The properties that Logseq always reads that way, `alias` and `tags`,
	// are read that way also when they are not in the list.
	PropertiesSeparatedByCommas []string

	// IgnoredPageReferences are the properties whose value is not split into
	// pages at commas. A link or a tag in the value still points at its page,
	// the same as in Logseq. It replaces `:ignored-page-references-keywords`.
	IgnoredPageReferences []string

	// PropertyPages is whether the name of a property points at the page with
	// that name, so that a block with `type:: book` is a linked reference of
	// the page `type`. It replaces `:property-pages/enabled?`. Logseq does
	// this unless the graph turns it off. Use Bool to set it.
	PropertyPages *bool

	// PropertyPagesExcluded are the properties whose names do not point at a
	// page, also when PropertyPages is on. It replaces
	// `:property-pages/excludelist`.
	PropertyPagesExcluded []string

	// TaskPages is whether the task marker and the priority of a block point
	// at the pages with the same name, so that a block with `TODO [#A] x` is
	// a linked reference of the pages `TODO` and `A`. Logseq always does
	// this, and has no setting for it in `config.edn`. Use Bool to set it.
	TaskPages *bool

	// NamespaceParentRefs is whether a reference to a page in a namespace
	// also points at the namespaces above it, so that a block with
	// `[[a/b/c]]` is a linked reference of the pages `a/b` and `a`. Logseq
	// always does this, and has no setting for it in `config.edn`. Use Bool
	// to set it.
	NamespaceParentRefs *bool

	// FirstBlockPageProperties is whether saving a page writes its first
	// block without a bullet when the block starts with properties, so that
	// `- tags:: x` as the first block is saved as `tags:: x` and becomes the
	// page properties. Logseq does this each time it saves a page, and has no
	// setting for it in `config.edn`. It is on unless it is set to false. Use
	// Bool to set it.
	//
	// Only pages that are written are changed, so a page that is opened and
	// saved without changes keeps its bullet. A first block that has child
	// blocks also keeps its bullet, as its children would be read back as a
	// different outline.
	FirstBlockPageProperties *bool

	// RelaxedTaskDates is whether `SCHEDULED:` and `DEADLINE:` lines are read
	// as dates also in the forms and places that Logseq ignores. Logseq has
	// no setting for it in `config.edn`, and it is off unless it is set to
	// true. Use Bool to set it.
	//
	// When it is off, a date is read the same way as Logseq reads it:
	//
	//   - The date has a weekday of ASCII letters after one space, such as
	//     `<2024-01-01 Mon>`. `<2024-01-01>` and `<2024-01-01 Mån>` are text.
	//   - The line is directly after the first line of the block, or after
	//     properties, code or other content that is not text. A date after
	//     more text, or after a blank line, is text.
	//   - Only the first group of date lines counts. In it, the last date of
	//     each kind is the date of the task.
	//
	// When it is on, each such line is a date, wherever it is in the block,
	// and the weekday can be missing or in any language. Such a date is
	// saved with an English weekday, which Logseq reads, but the place of
	// the line does not change. This is for graphs that are not opened in
	// Logseq.
	RelaxedTaskDates *bool

	// LaterPropertyGroupRefs is whether the pages named in the properties of
	// a block after its first group of properties are references. A block
	// has more than one group when its properties are split by a text line
	// or a blank line, as in:
	//
	//	- a
	//	  b:: c
	//	  text
	//	  d:: [[e]]
	//
	// Logseq reads only the first group, so `d` is not a property of the
	// block and `e` is not a reference. Logseq has no setting for it in
	// `config.edn`, and it is off unless it is set to true. Use Bool to set
	// it. When it is on, `e` is a reference, but `d` is still not a property
	// of the block. This is for graphs that are not opened in Logseq.
	LaterPropertyGroupRefs *bool
}

// Bool returns a pointer to the value, for the fields of Config that are not
// set when they are nil.
func Bool(value bool) *bool {
	return &value
}

// merge returns the config with the fields that are set in other replaced.
func (c Config) merge(other Config) Config {
	if other.PropertiesSeparatedByCommas != nil {
		c.PropertiesSeparatedByCommas = other.PropertiesSeparatedByCommas
	}

	if other.IgnoredPageReferences != nil {
		c.IgnoredPageReferences = other.IgnoredPageReferences
	}

	if other.PropertyPages != nil {
		c.PropertyPages = other.PropertyPages
	}

	if other.PropertyPagesExcluded != nil {
		c.PropertyPagesExcluded = other.PropertyPagesExcluded
	}

	if other.TaskPages != nil {
		c.TaskPages = other.TaskPages
	}

	if other.NamespaceParentRefs != nil {
		c.NamespaceParentRefs = other.NamespaceParentRefs
	}

	if other.FirstBlockPageProperties != nil {
		c.FirstBlockPageProperties = other.FirstBlockPageProperties
	}

	if other.RelaxedTaskDates != nil {
		c.RelaxedTaskDates = other.RelaxedTaskDates
	}

	if other.LaterPropertyGroupRefs != nil {
		c.LaterPropertyGroupRefs = other.LaterPropertyGroupRefs
	}

	return c
}

// firstBlockPageProperties returns true if saving a page moves the properties
// at the start of its first block to the page, see FirstBlockPageProperties.
func (c Config) firstBlockPageProperties() bool {
	return c.FirstBlockPageProperties == nil || *c.FirstBlockPageProperties
}

// relaxedTaskDates returns true if task dates are read also in the forms
// that Logseq ignores, see RelaxedTaskDates.
func (c Config) relaxedTaskDates() bool {
	return c.RelaxedTaskDates != nil && *c.RelaxedTaskDates
}

// laterPropertyGroupRefs returns true if the groups of properties of a block
// after the first make references, see LaterPropertyGroupRefs.
func (c Config) laterPropertyGroupRefs() bool {
	return c.LaterPropertyGroupRefs != nil && *c.LaterPropertyGroupRefs
}

// applyTo replaces the settings in the config of the graph with the fields
// that are set.
func (c Config) applyTo(config *utils.GraphConfig) {
	if c.PropertiesSeparatedByCommas != nil {
		config.PropertiesSeparatedByCommas = copyStrings(c.PropertiesSeparatedByCommas)
	}

	if c.IgnoredPageReferences != nil {
		config.IgnoredPageReferencesKeywords = copyStrings(c.IgnoredPageReferences)
	}

	if c.PropertyPages != nil {
		enabled := *c.PropertyPages
		config.PropertyPagesEnabled = &enabled
	}

	if c.PropertyPagesExcluded != nil {
		config.PropertyPagesExcludelist = copyStrings(c.PropertyPagesExcluded)
	}
}

// indexOptions are the options of the index for the fields that the config of
// the graph does not have.
func (c Config) indexOptions() []indexing.Option {
	opts := make([]indexing.Option, 0, 2)
	if c.TaskPages != nil {
		opts = append(opts, indexing.WithTaskPages(*c.TaskPages))
	}

	if c.NamespaceParentRefs != nil {
		opts = append(opts, indexing.WithNamespaceParentRefs(*c.NamespaceParentRefs))
	}

	return opts
}

// copyStrings copies a list, so that a change to the list of the caller does
// not change the settings of an open graph.
func copyStrings(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	return result
}
