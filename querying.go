package logseq

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/aholstenson/logseq-go/internal/indexing"
	"github.com/aholstenson/logseq-go/internal/utils"
	"github.com/aholstenson/logseq-go/query"
)

// QueryResultType tells if a query found blocks or pages.
type QueryResultType int

const (
	// QueryResultBlocks is the result of a query with one or more filters
	// that match blocks.
	QueryResultBlocks QueryResultType = iota
	// QueryResultPages is the result of a query with only filters that match
	// pages, or with no filters at all.
	QueryResultPages
)

// QueryResults are the results of running a query with Graph.Query.
type QueryResults struct {
	// Type tells if the query found blocks or pages.
	Type QueryResultType

	// Blocks are the blocks the query found, nil if it found pages.
	Blocks []BlockResult

	// Pages are the pages the query found, nil if it found blocks.
	Pages []PageResult
}

// QueryOption is an option for running a query with Graph.Query.
type QueryOption func(*queryOptions)

type queryOptions struct {
	now time.Time
}

// QueryAt runs a query as if it is the given time, which decides the dates
// that relative dates such as `today` or `-7d` stand for. The default is the
// current time.
func QueryAt(now time.Time) QueryOption {
	return func(o *queryOptions) {
		o.now = now
	}
}

// Query runs a query on the graph and returns all of the results, in the
// order of the sort of the query. Without a sort the results are in the order
// of how well they match. When the results are sorted by a property, values
// that are numbers sort by their value and before values that are text, and
// text sorts without regard for case. Results without a value come last in
// both orders.
//
// The same as in Logseq, a query finds blocks if one or more of its filters
// match blocks, such as `(task TODO)` or `[[page]]`, and pages if all of its
// filters match pages, such as `(page-property type book)`. When a query that
// finds blocks has filters that match pages, these filters match the page a
// block is on.
//
// Dates are calendar days in the local time zone. Errors in the query, such as
// a date that is not valid, are returned as errors.
//
// Queries are run on the index, so this requires the graph to have been opened
// with an index.
func (g *Graph) Query(ctx context.Context, q *query.Query, opts ...QueryOption) (*QueryResults, error) {
	return g.runQuery(ctx, q, opts, g)
}

func (g *Graph) runQuery(ctx context.Context, q *query.Query, opts []QueryOption, source pageSource) (*QueryResults, error) {
	if g.index == nil {
		return nil, fmt.Errorf("indexing is not enabled")
	}

	options := &queryOptions{
		now: time.Now(),
	}
	for _, opt := range opts {
		opt(options)
	}

	sortBy, err := querySortFields(q.Sort)
	if err != nil {
		return nil, err
	}

	runner := &queryRunner{
		ctx:   ctx,
		graph: g,
		now:   options.now.Local(),
	}

	results := &QueryResults{}
	if q.FindsBlocks() {
		results.Type = QueryResultBlocks

		indexQuery, err := runner.compile(q.Filter, true)
		if err != nil {
			return nil, err
		}

		err = eachResult(func(opts indexing.SearchOptions) (indexing.SearchResults[*indexing.Block], error) {
			opts.SortBy = sortBy
			return g.index.SearchBlocks(ctx, indexQuery, opts)
		}, func(block *indexing.Block) {
			results.Blocks = append(results.Blocks, g.newBlockResult(block, source))
		})
		if err != nil {
			return nil, err
		}

		results.Blocks = sample(results.Blocks, q.Sample)
	} else {
		results.Type = QueryResultPages

		indexQuery, err := runner.compile(q.Filter, false)
		if err != nil {
			return nil, err
		}

		err = eachResult(func(opts indexing.SearchOptions) (indexing.SearchResults[*indexing.Page], error) {
			opts.SortBy = sortBy
			return g.index.SearchPages(ctx, indexQuery, opts)
		}, func(page *indexing.Page) {
			results.Pages = append(results.Pages, g.newPageResult(page, source))
		})
		if err != nil {
			return nil, err
		}

		results.Pages = sample(results.Pages, q.Sample)
	}

	return results, nil
}

func querySortFields(sorts []query.Sort) ([]indexing.SortField, error) {
	fields := make([]indexing.SortField, 0, len(sorts))
	for _, s := range sorts {
		var name string
		switch s.Field {
		case query.SortPriority:
			name = indexing.SortFieldPriority
		case query.SortScheduled:
			name = indexing.SortFieldScheduled
		case query.SortDeadline:
			name = indexing.SortFieldDeadline
		case query.SortJournalDay:
			name = indexing.SortFieldJournalDate
		case query.SortTitle:
			name = indexing.SortFieldTitle
		case query.SortProperty:
			if s.Property == "" {
				return nil, fmt.Errorf("query: the property to sort by has no name")
			}
			name = indexing.SortFieldProperty(s.Property)
		default:
			return nil, fmt.Errorf("query: unknown sort field %d", s.Field)
		}

		fields = append(fields, indexing.SortField{
			Field: name,
			Asc:   s.Order == query.Asc,
		})
	}

	return fields, nil
}

// queryRunner turns the filters of a query into queries on the index.
type queryRunner struct {
	ctx   context.Context
	graph *Graph

	// now is the time that relative dates are relative to, in the local
	// time zone.
	now time.Time
}

// compile turns a filter into a query on the index. If blocks is true the
// query is run on blocks, and the filters that match pages match the page a
// block is on.
func (r *queryRunner) compile(filter query.Filter, blocks bool) (indexing.Query, error) {
	switch f := filter.(type) {
	case nil:
		return indexing.All(), nil
	case *query.AndFilter:
		if len(f.Filters) == 0 {
			return indexing.All(), nil
		}

		clauses, err := r.compileAll(f.Filters, blocks)
		if err != nil {
			return nil, err
		}
		return indexing.And(clauses...), nil
	case *query.OrFilter:
		if len(f.Filters) == 0 {
			return indexing.None(), nil
		}

		clauses, err := r.compileAll(f.Filters, blocks)
		if err != nil {
			return nil, err
		}
		return indexing.Or(clauses...), nil
	case *query.NotFilter:
		if f.Filter == nil {
			return nil, fmt.Errorf("query: `not` has no filter")
		}

		clause, err := r.compile(f.Filter, blocks)
		if err != nil {
			return nil, err
		}
		return indexing.Not(clause), nil
	case *query.TaskFilter:
		return indexing.TaskStatusIs(f.Statuses...), nil
	case *query.PriorityFilter:
		clauses := make([]indexing.Query, 0, len(f.Priorities))
		for _, priority := range f.Priorities {
			clauses = append(clauses, indexing.PriorityIs(priority))
		}

		if len(clauses) == 0 {
			return indexing.None(), nil
		}
		return indexing.Or(clauses...), nil
	case *query.PageRefFilter:
		return indexing.PathReferences(f.Page), nil
	case *query.RefsFilter:
		return indexing.References(f.Page), nil
	case *query.FullTextFilter:
		return indexing.ContentMatches(f.Text), nil
	case *query.PageFilter:
		return indexing.PageTitleIs(f.Page), nil
	case *query.ParentFilter:
		if f.Block == "" {
			return nil, fmt.Errorf("query: the parent block has no UUID")
		}
		return indexing.ParentIs(f.Block), nil
	case *query.AncestorFilter:
		if f.Block == "" {
			return nil, fmt.Errorf("query: the ancestor block has no UUID")
		}
		return indexing.AncestorIs(f.Block), nil
	case *query.PropertyFilter:
		return propertyQuery(f.Name, f.Value), nil
	case *query.DateFilter:
		return r.dateQuery(f)
	case *query.PagePropertyFilter, *query.PageTagsFilter, *query.NamespaceFilter, *query.AllPageTagsFilter:
		pageQuery, err := r.compilePageFilter(filter)
		if err != nil {
			return nil, err
		}

		if !blocks {
			return pageQuery, nil
		}

		// The index has no joins, so the pages are found first and the blocks
		// are then matched by the page they are on.
		return r.blocksOnPages(pageQuery)
	}

	return nil, fmt.Errorf("query: unknown filter %T", filter)
}

func (r *queryRunner) compileAll(filters []query.Filter, blocks bool) ([]indexing.Query, error) {
	clauses := make([]indexing.Query, 0, len(filters))
	for _, filter := range filters {
		clause, err := r.compile(filter, blocks)
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, clause)
	}

	return clauses, nil
}

// compilePageFilter turns a filter that matches pages into a query on the
// index that is run on pages.
func (r *queryRunner) compilePageFilter(filter query.Filter) (indexing.Query, error) {
	switch f := filter.(type) {
	case *query.PagePropertyFilter:
		return propertyQuery(f.Name, f.Value), nil
	case *query.PageTagsFilter:
		clauses := make([]indexing.Query, 0, len(f.Tags))
		for _, tag := range f.Tags {
			clauses = append(clauses, indexing.PropertyReferences("tags", tag))
		}

		if len(clauses) == 0 {
			return indexing.None(), nil
		}
		return indexing.Or(clauses...), nil
	case *query.NamespaceFilter:
		return indexing.InNamespace(f.Namespace), nil
	case *query.AllPageTagsFilter:
		return r.allPageTags()
	}

	return nil, fmt.Errorf("query: unknown page filter %T", filter)
}

// allPageTags returns a query that matches the pages that other pages use as
// tags. The index has no joins, so the tags of all pages are found first. Tags
// for pages that do not have a file are not in the index, and do not match.
func (r *queryRunner) allPageTags() (indexing.Query, error) {
	seen := make(map[string]bool)
	var clauses []indexing.Query
	err := eachResult(func(opts indexing.SearchOptions) (indexing.SearchResults[*indexing.Page], error) {
		return r.graph.index.SearchPages(r.ctx, indexing.HasProperty("tags"), opts)
	}, func(page *indexing.Page) {
		for _, tag := range page.Tags {
			if !seen[tag] {
				seen[tag] = true
				clauses = append(clauses, indexing.PageTitleIs(tag))
			}
		}
	})
	if err != nil {
		return nil, err
	}

	if len(clauses) == 0 {
		return indexing.None(), nil
	}
	return indexing.Or(clauses...), nil
}

// blocksOnPages finds the pages that match a query, and returns a query that
// matches the blocks on these pages.
func (r *queryRunner) blocksOnPages(pageQuery indexing.Query) (indexing.Query, error) {
	var subPaths []string
	err := eachResult(func(opts indexing.SearchOptions) (indexing.SearchResults[*indexing.Page], error) {
		return r.graph.index.SearchPages(r.ctx, pageQuery, opts)
	}, func(page *indexing.Page) {
		subPaths = append(subPaths, page.SubPath)
	})
	if err != nil {
		return nil, err
	}

	return indexing.OnPages(subPaths...), nil
}

// propertyQuery matches a property with the value, either as the text of the
// property or as a reference to a page with the value as its title. An empty
// value matches the property whatever its value is.
func propertyQuery(name string, value string) indexing.Query {
	if value == "" {
		return indexing.HasProperty(name)
	}

	return indexing.Or(
		indexing.PropertyEquals(name, value),
		indexing.PropertyReferences(name, value),
	)
}

func (r *queryRunner) dateQuery(f *query.DateFilter) (indexing.Query, error) {
	var between func(start, end time.Time) indexing.Query
	switch f.Field {
	case query.FieldJournal:
		between = indexing.JournalBetween
	case query.FieldScheduled:
		between = indexing.ScheduledBetween
	case query.FieldDeadline:
		between = indexing.DeadlineBetween
	default:
		return nil, fmt.Errorf("query: unknown date field %d", f.Field)
	}

	if f.Op == query.Exists {
		return between(time.Time{}, time.Time{}), nil
	}

	date, err := r.resolveDate(f.Date)
	if err != nil {
		return nil, err
	}

	// Dates are matched as whole days, as a date can have a time of day, such
	// as `SCHEDULED: <2024-01-15 Mon 09:00>`.
	nextDay := date.AddDate(0, 0, 1)

	switch f.Op {
	case query.Equal:
		return between(date, nextDay), nil
	case query.Less:
		return between(time.Time{}, date), nil
	case query.LessOrEqual:
		return between(time.Time{}, nextDay), nil
	case query.Greater:
		return between(nextDay, time.Time{}), nil
	case query.GreaterOrEqual:
		return between(date, time.Time{}), nil
	case query.Range:
		end, err := r.resolveDate(f.End)
		if err != nil {
			return nil, err
		}

		// Logseq accepts the dates of a range in either order.
		if end.Before(date) {
			date, end = end, date
		}

		return between(date, end.AddDate(0, 0, 1)), nil
	}

	return nil, fmt.Errorf("query: unknown date operation %d", f.Op)
}

// resolveDate returns midnight of a date in the local time zone.
func (r *queryRunner) resolveDate(date query.Date) (time.Time, error) {
	switch {
	case date.Relative != "":
		input, ok := utils.ParseQueryDateInput(date.Relative, r.now)
		if !ok || input.HasTime {
			return time.Time{}, fmt.Errorf("query: %q is not a date", date.Relative)
		}
		return input.Time, nil
	case date.Journal != "":
		t, err := r.graph.journalTitleFormat.Parse(date.Journal)
		if err != nil {
			return time.Time{}, fmt.Errorf("query: %q is not the title of a journal: %w", date.Journal, err)
		}
		return t, nil
	case !date.Fixed.IsZero():
		return time.Date(date.Fixed.Year(), date.Fixed.Month(), date.Fixed.Day(), 0, 0, 0, 0, time.Local), nil
	}

	return time.Time{}, fmt.Errorf("query: a date is missing")
}

// sample picks n random items, in the order they have in items. All of the
// items are returned if n is zero or there are not more than n items.
func sample[T any](items []T, n int) []T {
	if n <= 0 || len(items) <= n {
		return items
	}

	picked := rand.Perm(len(items))[:n]
	sort.Ints(picked)

	result := make([]T, 0, n)
	for _, i := range picked {
		result = append(result, items[i])
	}
	return result
}
