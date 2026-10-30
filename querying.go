package logseq

import (
	"context"
	"fmt"
	"math"
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
	case *query.ReferencesFilter:
		return r.referencesQuery(f)
	case *query.OnPageFilter:
		if f.Pages == nil || query.New(f.Pages).FindsBlocks() {
			return nil, fmt.Errorf("query: page takes a filter for pages, such as (title = \"x\")")
		}

		// In a query for blocks, the filters for pages match the page of
		// each block.
		return r.compile(f.Pages, true)
	case *query.BelowFilter:
		return r.belowQuery(f)
	case *query.HasChildFilter:
		return r.hasChildQuery(f)
	case *query.FullTextFilter:
		return indexing.ContentMatches(f.Text), nil
	case *query.PageFilter:
		return indexing.PageTitleIs(f.Page), nil
	case *query.ParentFilter:
		return blockQuery("parent", f.Block, f.Page, f.Location, indexing.ParentIs, indexing.ParentAt)
	case *query.AncestorFilter:
		return blockQuery("ancestor", f.Block, f.Page, f.Location, indexing.AncestorIs, indexing.AncestorAt)
	case *query.PropertyFilter:
		return propertyQuery(f.Name, f.Value), nil
	case *query.PropertyCompareFilter:
		compareQuery, err := r.propertyCompareQuery(f)
		if err != nil || !f.Page || !blocks {
			return compareQuery, err
		}
		return r.blocksOnPages(compareQuery)
	case *query.DateFilter:
		return r.dateQuery(f)
	case *query.TitleFilter:
		// Blocks have the title of their page, so this matches the blocks on
		// the pages as well.
		return titleQuery(f)
	case *query.HasBlockFilter:
		subPaths, err := r.pagesWithBlocks(f.Filter)
		if err != nil {
			return nil, err
		}

		if blocks {
			return indexing.OnPages(subPaths...), nil
		}
		return indexing.PagesAt(subPaths...), nil
	case *query.PagePropertyFilter, *query.PageTagsFilter, *query.NamespaceFilter, *query.AllPageTagsFilter,
		*query.AliasFilter:
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
	case *query.AliasFilter:
		return indexing.HasAlias(f.Alias), nil
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

// pagesWithBlocks returns the sub paths of the pages with one or more blocks
// that match the filter.
func (r *queryRunner) pagesWithBlocks(filter query.Filter) ([]string, error) {
	seen := make(map[string]bool)
	var subPaths []string
	err := r.eachBlock("has-block", filter, func(block *indexing.Block) {
		if !seen[block.PageSubPath] {
			seen[block.PageSubPath] = true
			subPaths = append(subPaths, block.PageSubPath)
		}
	})
	return subPaths, err
}

// eachBlock calls f for each block that the filter matches. The index has no
// joins, so filters that relate blocks to other blocks or pages find these
// first. name is the name of the filter that needs the blocks, for errors.
func (r *queryRunner) eachBlock(name string, filter query.Filter, f func(block *indexing.Block)) error {
	if filter == nil {
		return fmt.Errorf("query: %s has no filter", name)
	}

	blockQuery, err := r.compile(filter, true)
	if err != nil {
		return err
	}

	return eachResult(func(opts indexing.SearchOptions) (indexing.SearchResults[*indexing.Block], error) {
		return r.graph.index.SearchBlocks(r.ctx, blockQuery, opts)
	}, f)
}

// belowQuery matches the blocks below the blocks that the filter of f
// matches, by the location of these blocks on their pages.
func (r *queryRunner) belowQuery(f *query.BelowFilter) (indexing.Query, error) {
	name, below := "parent", indexing.ParentAt
	if f.AnyDepth {
		name, below = "ancestor", indexing.AncestorAt
	}

	var clauses []indexing.Query
	err := r.eachBlock(name, f.Filter, func(block *indexing.Block) {
		if len(block.Location) == 0 {
			return
		}
		clauses = append(clauses, indexing.And(indexing.OnPages(block.PageSubPath), below(block.Location)))
	})
	if err != nil {
		return nil, err
	}

	if len(clauses) == 0 {
		return indexing.None(), nil
	}
	return indexing.Or(clauses...), nil
}

// hasChildQuery matches the blocks above the blocks that the filter of f
// matches, by the location of these blocks on their pages.
func (r *queryRunner) hasChildQuery(f *query.HasChildFilter) (indexing.Query, error) {
	name := "has-child"
	if f.AnyDepth {
		name = "has-descendant"
	}

	seen := make(map[string]bool)
	var clauses []indexing.Query
	add := func(subPath string, location []int) {
		key := fmt.Sprint(subPath, location)
		if !seen[key] {
			seen[key] = true
			clauses = append(clauses, indexing.BlockAt(subPath, location))
		}
	}

	err := r.eachBlock(name, f.Filter, func(block *indexing.Block) {
		// A block at the top of a page has no block above it.
		if len(block.Location) < 2 {
			return
		}

		if !f.AnyDepth {
			add(block.PageSubPath, block.Location[:len(block.Location)-1])
			return
		}

		for depth := 1; depth < len(block.Location); depth++ {
			add(block.PageSubPath, block.Location[:depth])
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

// titleQuery matches the pages with a title that compares with the text of the
// filter, and the blocks on them.
func titleQuery(f *query.TitleFilter) (indexing.Query, error) {
	switch f.Op {
	case query.TitleEquals:
		return indexing.PageTitleIs(f.Text), nil
	}

	match, err := textMatch(f.Op)
	if err != nil {
		return nil, err
	}
	return indexing.PageTitleMatches(match, f.Text), nil
}

// textMatch returns how the index compares a part of a title, for the
// comparisons of a TitleFilter other than TitleEquals.
func textMatch(op query.TitleOp) (indexing.TextMatch, error) {
	switch op {
	case query.TitleStartsWith:
		return indexing.MatchPrefix, nil
	case query.TitleEndsWith:
		return indexing.MatchSuffix, nil
	case query.TitleIncludes:
		return indexing.MatchIncludes, nil
	}

	return 0, fmt.Errorf("query: unknown title comparison %d", op)
}

// referencesQuery matches the blocks that reference one or more of the pages
// that the filter of f matches. A filter on the title is matched on the
// references themselves, so that pages without a file match as well. Other
// filters are run on the pages first, as the index has no joins, so only
// pages with a file match them.
func (r *queryRunner) referencesQuery(f *query.ReferencesFilter) (indexing.Query, error) {
	if f.Pages == nil {
		return nil, fmt.Errorf("query: the references have no filter for the pages")
	}

	byTitle := indexing.References
	byMatch := indexing.ReferencesMatching
	if f.Path {
		byTitle = indexing.PathReferences
		byMatch = indexing.PathReferencesMatching
	}

	if title, ok := f.Pages.(*query.TitleFilter); ok {
		if title.Op == query.TitleEquals {
			return byTitle(title.Text), nil
		}

		match, err := textMatch(title.Op)
		if err != nil {
			return nil, err
		}
		return byMatch(match, title.Text), nil
	}

	pageQuery, err := r.compile(f.Pages, false)
	if err != nil {
		return nil, err
	}

	var clauses []indexing.Query
	err = eachResult(func(opts indexing.SearchOptions) (indexing.SearchResults[*indexing.Page], error) {
		return r.graph.index.SearchPages(r.ctx, pageQuery, opts)
	}, func(page *indexing.Page) {
		clauses = append(clauses, byTitle(r.graph.newPageResult(page, r.graph).Title()))
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

// blockQuery matches the blocks below a block that is named by its UUID, or by
// its page and location when it has none.
func blockQuery(
	name string,
	block string,
	page string,
	location []int,
	byID func(string) indexing.Query,
	byLocation func([]int) indexing.Query,
) (indexing.Query, error) {
	switch {
	case block != "":
		return byID(block), nil
	case page != "" && len(location) > 0:
		// The location is the same on every page, so the page is a part of
		// the query.
		return indexing.And(indexing.PageTitleIs(page), byLocation(location)), nil
	case page != "" && name == "parent":
		// Without a location the page itself is the parent, which it is
		// for the blocks at the top of the page.
		return indexing.And(indexing.PageTitleIs(page), indexing.TopLevel()), nil
	case page != "":
		return indexing.PageTitleIs(page), nil
	}

	return nil, fmt.Errorf("query: the %s block has no UUID or location", name)
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
	case query.FieldJournalPage:
		// Journal pages and the blocks on them have the date in the same
		// field, so this matches both.
		between = indexing.JournalBetween
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

	switch f.Op {
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

	start, end, err := dayBounds(f.Op, date)
	if err != nil {
		return nil, err
	}
	return between(start, end), nil
}

// dayBounds returns the start, inclusive, and end, exclusive, of the times
// that compare with a date. Dates are matched as whole days, as a date can
// have a time of day, such as `SCHEDULED: <2024-01-15 Mon 09:00>`. A zero time
// leaves that end open.
func dayBounds(op query.Op, date time.Time) (time.Time, time.Time, error) {
	nextDay := date.AddDate(0, 0, 1)

	switch op {
	case query.Equal:
		return date, nextDay, nil
	case query.Less:
		return time.Time{}, date, nil
	case query.LessOrEqual:
		return time.Time{}, nextDay, nil
	case query.Greater:
		return nextDay, time.Time{}, nil
	case query.GreaterOrEqual:
		return date, time.Time{}, nil
	}

	return time.Time{}, time.Time{}, fmt.Errorf("query: unknown date operation %d", op)
}

// propertyCompareQuery matches the property when its value is a number or a
// date that compares with the value of the filter.
func (r *queryRunner) propertyCompareQuery(f *query.PropertyCompareFilter) (indexing.Query, error) {
	if !f.Date.IsZero() {
		date, err := r.resolveDate(f.Date)
		if err != nil {
			return nil, err
		}

		start, end, err := dayBounds(f.Op, date)
		if err != nil {
			return nil, err
		}
		return indexing.PropertyDateBetween(f.Name, start, end), nil
	}

	min, max := math.Inf(-1), math.Inf(1)
	minInclusive, maxInclusive := false, false
	switch f.Op {
	case query.Equal:
		min, max, minInclusive, maxInclusive = f.Number, f.Number, true, true
	case query.Less:
		max = f.Number
	case query.LessOrEqual:
		max, maxInclusive = f.Number, true
	case query.Greater:
		min = f.Number
	case query.GreaterOrEqual:
		min, minInclusive = f.Number, true
	default:
		return nil, fmt.Errorf("query: unknown comparison %d", f.Op)
	}

	return indexing.PropertyNumberBetween(f.Name, min, max, minInclusive, maxInclusive), nil
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
