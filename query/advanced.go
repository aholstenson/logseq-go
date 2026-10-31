package query

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/utils"
)

// AdvancedQuery is an advanced query of Logseq, the EDN map that is written
// between `#+BEGIN_QUERY` and `#+END_QUERY`, translated into a Query.
type AdvancedQuery struct {
	// Title is the text of the title, without Hiccup markup such as
	// `[:b ...]`. Empty if the query has no title.
	Title string

	// Query is the query that the advanced query was translated into.
	Query *Query

	// Collapsed is true if Logseq shows the results of the query collapsed.
	Collapsed bool

	// GroupByPage is true if Logseq shows the blocks that the query finds
	// grouped below the page they are on. The value of `:group-by-page?` is
	// used if the query has it. If not, it is false when the query has a
	// `:result-transform` or `(by-page false)`, and true for other queries.
	//
	// GroupByPage is always false when TableView is true, as Logseq does not
	// group the results in a table.
	GroupByPage bool

	// RemoveBlockChildren is true if Logseq leaves a block out of the results
	// when the block directly above it is also in the results. It is the
	// value of `:remove-block-children?`, and true if the query does not have
	// it. Give it to logseq.RemoveBlockChildren when the query is run.
	RemoveBlockChildren bool

	// TableView is true if Logseq shows the results in a table. This is the
	// case if the query has `:table-view? true`, if the query is text that
	// ends with `table`, such as `{{query (task TODO) table}}`, or if the
	// block with the query has the property `query-table:: true`. Only
	// ParseNode can read the property, as ParseAdvanced does not get the
	// block.
	TableView bool

	// TableProperties are the names of the columns of the table, from the
	// property `query-properties::` of the block with the query, such as
	// `[:block :page :rating]`. Nil if the block does not have the property,
	// in which case Logseq shows a column for each property in the results.
	TableProperties []string

	// TableSortBy is the column that Logseq sorts the table by: the name of a
	// property, or `page`, `block`, `created-at` or `updated-at`. It is the
	// value of the property `query-sort-by::` of the block with the query. If
	// the block does not have the property, it is `updated-at`, or empty if
	// the query has a `(sort-by ...)` of its own, which keeps the order of
	// the query.
	TableSortBy string

	// TableSortDesc is true if the table is sorted in descending order. It is
	// false if the block with the query has `query-sort-desc:: false`.
	TableSortDesc bool

	// BreadcrumbShow is true if Logseq shows the parent blocks above each
	// block that the query finds. It is the value of `:breadcrumb-show?`, and
	// true if the query does not have it.
	BreadcrumbShow bool
}

// ParseOption is an option for ParseAdvanced and ParseNode.
type ParseOption func(*parseOptions)

type parseOptions struct {
	currentPage  string
	queryPage    string
	currentBlock blockRef
	parentBlock  blockRef
}

// blockRef names a block by its UUID, or by its place on a page when it has
// no UUID. See ParentFilter.
type blockRef struct {
	id       string
	page     string
	location []int
}

func (r blockRef) known() bool {
	return r.id != "" || r.page != ""
}

// parent returns the filter for the blocks directly below the block.
func (r blockRef) parent() Filter {
	if r.id != "" {
		return Parent(r.id)
	}
	return ParentAt(r.page, r.location...)
}

// ancestor returns the filter for the blocks below the block at any depth.
func (r blockRef) ancestor() Filter {
	if r.id != "" {
		return Ancestor(r.id)
	}
	return AncestorAt(r.page, r.location...)
}

// refOf returns the name of a block in a page: its UUID, or else its location
// on the page. Returns false if the block has no UUID and the page is not
// known, or if the block is not on a page.
func refOf(block *content.Block, page string) (blockRef, bool) {
	if id := block.ID(); id != "" {
		return blockRef{id: id}, true
	}

	if page == "" {
		return blockRef{}, false
	}

	location, ok := utils.LocationOf(block)
	if !ok {
		return blockRef{}, false
	}
	return blockRef{page: page, location: location}, true
}

// WithCurrentPage gives the title of the page that the input `:current-page`
// of an advanced query stands for. In Logseq that is the page that is shown,
// which is the page with the query, unless the query is shown in an embed or
// in the references of another page.
func WithCurrentPage(title string) ParseOption {
	return func(o *parseOptions) {
		o.currentPage = title
	}
}

// WithQueryPage gives the title of the page that the input `:query-page` of
// an advanced query stands for, which is the page the query is written on.
func WithQueryPage(title string) ParseOption {
	return func(o *parseOptions) {
		o.queryPage = title
	}
}

// WithCurrentBlock gives the UUID of the block that the input `:current-block`
// of an advanced query stands for, which is the block the query is written in.
// The UUID is the value of the `id::` property of the block, see
// content.Block.WithID. The block must have the property in the index for the
// query to find the blocks below it.
//
// ParseNode finds the block itself, also when it has no UUID, so this option
// is only needed for ParseAdvanced.
func WithCurrentBlock(id string) ParseOption {
	return func(o *parseOptions) {
		o.currentBlock = blockRef{id: id}
	}
}

// WithParentBlock gives the UUID of the block that the input `:parent-block`
// of an advanced query stands for, which is the parent of the block the query
// is written in. The UUID is the value of the `id::` property of the parent,
// the same as for WithCurrentBlock. ParseNode finds the parent itself, the
// same as for WithCurrentBlock.
func WithParentBlock(id string) ParseOption {
	return func(o *parseOptions) {
		o.parentBlock = blockRef{id: id}
	}
}

// WithCurrentBlockAt gives the block that the input `:current-block` of an
// advanced query stands for by its place on the page: the title of the page
// and the location of the block on it, see ParentFilter. Use it for a block
// without an `id::` property, when the query is read with ParseAdvanced. The
// location is the position of the block among the blocks next to it at each
// level, from the top of the page, so the location 8 is the ninth block at
// the top of the page and 0 1 is the second child of the first block.
func WithCurrentBlockAt(page string, location ...int) ParseOption {
	return func(o *parseOptions) {
		o.currentBlock = blockRef{page: page, location: location}
	}
}

// WithParentBlockAt gives the block that the input `:parent-block` of an
// advanced query stands for by its place on the page, the same as for
// WithCurrentBlockAt.
//
// A block at the top of a page has no parent block. Do not give this option
// for a query in such a block: without a location the option names the page,
// and the query then finds the blocks at the top of the page.
func WithParentBlockAt(page string, location ...int) ParseOption {
	return func(o *parseOptions) {
		o.parentBlock = blockRef{page: page, location: location}
	}
}

var inputOptionNames = map[string]string{
	"current-page":  "WithCurrentPage",
	"query-page":    "WithQueryPage",
	"current-block": "WithCurrentBlock or WithCurrentBlockAt",
	"parent-block":  "WithParentBlock or WithParentBlockAt",
}

// ParseNode reads the query in a node of a page: a simple query written as
// `{{query ...}}`, which is a *content.Query, or an advanced query written
// between `#+BEGIN_QUERY` and `#+END_QUERY`, which is a *content.QueryCommand.
// Nodes such as these are found with content.IsQuery, and the block that
// holds one with content.BlockOf.
//
// A simple query is read with Parse, and returned without a title. An
// advanced query is read with ParseAdvanced, which the options are given to.
//
// For an advanced query, the blocks that the inputs `:current-block` and
// `:parent-block` stand for are the block that holds the node and its parent,
// unless they are given with WithCurrentBlock, WithCurrentBlockAt,
// WithParentBlock or WithParentBlockAt. A block
// with an `id::` property is named by its UUID. A block without one is named
// by its location on the page given with WithQueryPage, see ParentFilter, so
// give that option for queries in blocks without the property.
//
// The properties `query-table::`, `query-properties::`, `query-sort-by::` and
// `query-sort-desc::` of the block with the query are read into the fields
// for the table. Run the query with Graph.Query in the logseq package.
//
// Logseq shows the results in a table when TableView is true, and always
// when the query finds pages. In a table, Logseq sorts the results by
// TableSortBy. ParseNode then adds this sort to the start of Query.Sort, so
// that Graph.Query returns the results in the same order. This is only done
// for a property or `page`, as the index can not sort by the other columns,
// and only for a query without a limit, as Logseq sorts the table after the
// results are limited.
func ParseNode(node content.Node, opts ...ParseOption) (*AdvancedQuery, error) {
	var result *AdvancedQuery
	switch n := node.(type) {
	case *content.Query:
		text, table := cutTable(n.Query)
		q, err := Parse(text)
		if err != nil {
			return nil, err
		}
		result = &AdvancedQuery{
			Query:               q,
			GroupByPage:         q.ByPage == nil || *q.ByPage,
			RemoveBlockChildren: true,
			TableView:           table,
			TableSortBy:         defaultTableSortBy(n.Query),
			TableSortDesc:       true,
			BreadcrumbShow:      true,
		}
	case *content.QueryCommand:
		options := newParseOptions(opts)

		if block := content.BlockOf(n); block != nil {
			if !options.currentBlock.known() {
				options.currentBlock, _ = refOf(block, options.queryPage)
			}

			// The root of a page is not a block that Logseq knows of, so a
			// block at the top of a page has no parent block.
			parent, ok := block.Parent().(*content.Block)
			if ok && parent.Parent() != nil && !options.parentBlock.known() {
				options.parentBlock, _ = refOf(parent, options.queryPage)
			}
		}

		var err error
		result, err = parseAdvanced(n.Query, options)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("query: %T is not a query", node)
	}

	if block := content.BlockOf(node); block != nil {
		if isTruthyProperty(block, "query-table") {
			result.TableView = true
		}
		if columns, ok := propertyText(block, "query-properties"); ok {
			result.TableProperties = tableColumns(columns)
		}
		if column, ok := propertyText(block, "query-sort-by"); ok && column != "" {
			result.TableSortBy = strings.TrimPrefix(column, ":")
		}
		if desc, ok := propertyText(block, "query-sort-desc"); ok && desc != "" {
			result.TableSortDesc = desc != "false"
		}
	}
	if result.TableView {
		result.GroupByPage = false
	}

	if (result.TableView || !result.Query.FindsBlocks()) && result.Query.Limit == 0 {
		if sort, ok := tableSort(result.TableSortBy, result.TableSortDesc); ok {
			result.Query.Sort = append([]Sort{sort}, result.Query.Sort...)
		}
	}

	return result, nil
}

// defaultTableSortBy returns the column that Logseq sorts a table by when the
// block with the query does not have the property `query-sort-by::`. Logseq
// keeps the order of the query if the text has `(sort-by`, also in a
// :result-transform, and else sorts by the time the blocks were changed.
func defaultTableSortBy(text string) string {
	if strings.Contains(text, "(sort-by") {
		return ""
	}

	return "updated-at"
}

// tableSort returns the sort for a column of a table, and false for a column
// that the index can not sort by.
func tableSort(column string, desc bool) (Sort, bool) {
	order := Asc
	if desc {
		order = Desc
	}

	switch column {
	case "", "block", "created-at", "updated-at":
		return Sort{}, false
	case "page":
		return Sort{Field: SortTitle, Order: order}, true
	}

	return Sort{Field: SortProperty, Property: column, Order: order}, true
}

// tableColumns reads the value of `query-properties::`, which is a vector of
// keywords such as `[:block :page :rating]`. The same as in Logseq, a value
// that can not be read gives no columns.
func tableColumns(text string) []string {
	r := &ednReader{input: text}
	node, err := r.read()
	if err != nil || (node.kind != ednVector && node.kind != ednList) {
		return nil
	}

	columns := make([]string, 0, len(node.items))
	for _, item := range node.items {
		if item.kind == ednKeyword && !slices.Contains(columns, item.text) {
			columns = append(columns, item.text)
		}
	}
	return columns
}

// cutTable returns the text of a simple query without the word `table` at the
// end, and if the text ends with `table`. Logseq shows the results of a query
// in a table if its text ends with `table`, and it reads only the first form
// of the text, so `(task TODO) table` is the query `(task TODO)`. The same as
// in Logseq, a text such as `portable` also ends with `table`.
func cutTable(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	rest, ok := strings.CutSuffix(trimmed, "table")
	if !ok {
		return text, false
	}

	// Only a separate word after the query is removed, so that a search such
	// as `portable` stays the same.
	if rest != strings.TrimRight(rest, " \t\n") && strings.TrimSpace(rest) != "" {
		return strings.TrimSpace(rest), true
	}

	return text, true
}

// isTruthyProperty returns true if Logseq treats the value of the property of
// the block as true: the property has a value, and the value is not `false`.
func isTruthyProperty(block *content.Block, name string) bool {
	v, _ := propertyText(block, name)
	return v != "" && v != "false"
}

// propertyText returns the text of the value of the property of the block,
// and false if the block does not have the property.
func propertyText(block *content.Block, name string) (string, bool) {
	properties := block.FindProperties()
	if properties == nil {
		return "", false
	}

	property := properties.GetAsNode(name)
	if property == nil {
		return "", false
	}

	var value strings.Builder
	for _, node := range property.Children() {
		if text, ok := node.(*content.Text); ok {
			value.WriteString(text.Value)
		}
	}

	return strings.TrimSpace(value.String()), true
}

func newParseOptions(opts []ParseOption) *parseOptions {
	options := &parseOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// ErrUnsupported is wrapped by the errors of ParseAdvanced for queries that
// can not be translated into a Query.
var ErrUnsupported = errors.New("not supported")

// UnsupportedError is a part of an advanced query that can not be translated
// into a Query. It wraps ErrUnsupported.
type UnsupportedError struct {
	// Offset is the byte offset in the text where the part starts.
	Offset int

	// Form is the text of the part, such as a clause of the query.
	Form string

	// Message describes why the part can not be translated.
	Message string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("query: %s: %s at offset %d", e.Message, e.Form, e.Offset)
}

func (e *UnsupportedError) Unwrap() error {
	return ErrUnsupported
}

var queryFence = regexp.MustCompile(`(?im)^\s*#\+(BEGIN|END)_QUERY\s*$`)

// ParseAdvanced reads an advanced query and translates it into a Query. The
// text is the EDN map, with or without the `#+BEGIN_QUERY` and `#+END_QUERY`
// lines around it.
//
// Datalog can express much more than a Query, so ParseAdvanced translates the
// forms that queries are commonly written in, and returns an
// *UnsupportedError for everything else. A condition such as
// `[(< ?d ?today)]` is applied where its variable gets its value, also in each
// branch of an `or-join`. These parts are translated:
//
//   - :find with one variable or `(pull ?b [*])`, for blocks or pages; a
//     query for pages with clauses about the blocks on them, such as
//     `[?b :block/page ?p] [?b :block/marker "TODO"]`, finds the pages with
//     one or more of these blocks, see HasBlockFilter; with only
//     `[?b :block/page ?p]` it finds the pages with one or more blocks
//   - :in and :inputs with date inputs such as `:today` or `:-7d`, page names
//     and numbers; inputs with a time of day, such as `:-7d-ms` or
//     `:right-now-ms`, are compared with :block/created-at and
//     :block/updated-at, see TimestampFilter
//   - the attributes :block/marker, :block/priority, :block/scheduled,
//     :block/deadline, :block/content, :block/created-at, :block/updated-at,
//     :block/page, :block/parent,
//     :block/refs, :block/path-refs, :block/properties, :block/name, :block/original-name,
//     :block/journal-day, :block/journal?, :block/tags, :block/namespace,
//     :block/alias and :page/properties
//   - the conditions =, not=, <, <=, >, >=, contains?, get and
//     clojure.string/includes?, and `[(get ?props :key) ?value]` with or
//     without a default value
//   - the content of a block compared with =, clojure.string/starts-with?,
//     ends-with?, includes?, re-find and re-matches, with regard for case,
//     see ContentFilter
//   - a regular expression as `#"bank"`, or from `[(re-pattern "bank") ?re]`;
//     the syntax is that of the regexp package, see TitleMatches
//   - a value in a set, such as `[(contains? #{"book" "film"} ?value)]`, for
//     values and for the titles of pages that blocks are on or refer to
//   - a property that is set, such as `[(get ?props :key)]`
//   - a property compared as a number or a date with <, <=, > and >=, or
//     with = and a date input, such as `[(< ?due ?today)]`, see
//     PropertyCompareFilter; with a number as the default value, such as
//     `[(get ?props :rating 0) ?r]`, the blocks or pages without the property
//     match when the default passes the comparison
//   - `[(missing? $ ?b :block/scheduled)]`, also for :block/marker,
//     :block/priority, :block/deadline and the :block/journal-day of the page
//   - conditions on a page that a block refers to, such as
//     `[?b :block/refs ?p] [?p :block/tags ?t] [?t :block/name "project"]`,
//     see ReferencesFilter
//   - conditions on the block above a block, such as
//     `[?b :block/parent ?p] [?p :block/marker "TODO"]`, or on a block below
//     it, such as `[?c :block/parent ?b] [?c :block/marker "TODO"]`, also at
//     any depth with the rule below, see BelowFilter and HasChildFilter
//   - a query for blocks with only conditions on their page, see OnPageFilter
//   - a page with a title as the parent, such as
//     `[?b :block/parent ?p] [?p :block/name "x"]`, for the blocks at the top
//     of the page, see ParentFilter
//   - conditions on the blocks on the page in a `not` or a branch of an
//     `or` in a query for pages, such as
//     `(not [?b :block/page ?p] [?b :block/marker "TODO"])`
//   - the title of a page compared with =, contains?,
//     clojure.string/starts-with?, ends-with?, includes?, re-find and
//     re-matches, see TitleFilter
//   - the title of a page with a known title as a value, such as ?title in
//     `[?p :block/name ?page] [?p :block/original-name ?title]`
//   - and, or, or-join, not and not-join
//   - the rules of simple queries, such as `(task ?b #{"TODO"})`
//   - a :query that is a simple query, such as `(and (task TODO) [[page]])`
//   - a :result-transform that sorts with sort-by and reverse, keeps
//     results with filter or remove, and takes the first results with take
//     as the last step, also with ->>, see Query.Limit; the function of filter or
//     remove can use and, or, not, not=, nil?, some? and the conditions of
//     :where on values such as `(:block/marker b)` or
//     `(get-in b [:block/properties :type])`
//
// The inputs `:current-page` and `:query-page` depend on where the query is,
// and are supported when the page is given with WithCurrentPage and
// WithQueryPage. The inputs `:current-block` and `:parent-block` are supported
// when the block is given with WithCurrentBlock and WithParentBlock by its
// UUID, with WithCurrentBlockAt and WithParentBlockAt by its place on the
// page, or found by ParseNode, as the parent of the blocks in
// `[?b :block/parent ?block]` or as the block above them in a rule such as the
// one below.
//
// The uses of the :rules are replaced with their clauses, with an `or-join`
// for a rule with more than one definition. Rules that use themselves are not
// supported, except the rule that finds the blocks below a block at any
// depth, which is translated into an AncestorFilter:
//
//	[(below ?b ?parent) [?b :block/parent ?parent]]
//	[(below ?b ?parent) [?b :block/parent ?c] (below ?c ?parent)]
//
// and the same rule with :block/namespace, which finds the pages in a
// namespace at any depth, see UnderNamespace.
//
// The keys :group-by-page?, :table-view?, :remove-block-children? and
// :breadcrumb-show? are read into the fields of AdvancedQuery. Other keys that
// only change how Logseq shows the results, such as :view, are ignored. Errors
// in the EDN itself are returned as *ParseError.
func ParseAdvanced(text string, opts ...ParseOption) (*AdvancedQuery, error) {
	return parseAdvanced(text, newParseOptions(opts))
}

func parseAdvanced(text string, options *parseOptions) (*AdvancedQuery, error) {
	// The fences are replaced with spaces so that offsets stay the same.
	text = queryFence.ReplaceAllStringFunc(text, func(s string) string {
		return strings.Repeat(" ", len(s))
	})

	r := &ednReader{input: text}
	root, err := r.read()
	if err != nil {
		return nil, err
	}

	if err := r.skipSpace(); err != nil {
		return nil, err
	}
	if r.pos < len(r.input) {
		return nil, errorAt(r.pos, "unexpected text after the query")
	}

	if root.kind != ednMap {
		return nil, errorAt(root.pos, "expected a map such as {:query [...]}")
	}

	t := &translator{
		src:    text,
		inputs: make(map[string]*ednNode),
		pages: map[string]string{
			"current-page": options.currentPage,
			"query-page":   options.queryPage,
		},
		blocks: map[string]blockRef{
			"current-block": options.currentBlock,
			"parent-block":  options.parentBlock,
		},
		blockInputs: make(map[string]blockRef),
		rules:       make(map[string]*recursiveRule),
		inlineRules: make(map[string][]*ruleDef),
	}

	result := &AdvancedQuery{
		RemoveBlockChildren: true,
		TableSortBy:         defaultTableSortBy(text),
		TableSortDesc:       true,
		BreadcrumbShow:      true,
	}
	var tableViewNode *ednNode
	var queryNode, inputsNode, transformNode, rulesNode, groupByPageNode *ednNode
	for i := 0; i < len(root.items); i += 2 {
		key, value := root.items[i], root.items[i+1]
		if key.kind != ednKeyword {
			return nil, t.unsupported(key, "keys of the map must be keywords")
		}

		switch key.text {
		case "title":
			result.Title = strings.TrimSpace(hiccupText(value))
		case "collapsed?":
			result.Collapsed = value.isSymbol("true")
		case "group-by-page?":
			groupByPageNode = value
		case "table-view?":
			tableViewNode = value
		case "remove-block-children?":
			result.RemoveBlockChildren = isTruthy(value)
		case "breadcrumb-show?":
			// Logseq shows the breadcrumbs unless the value is false.
			result.BreadcrumbShow = !value.isSymbol("false")
		case "query":
			queryNode = value
		case "inputs":
			inputsNode = value
		case "result-transform":
			transformNode = value
		case "rules":
			rulesNode = value
		}
	}

	if queryNode == nil {
		return nil, errorAt(root.pos, "the query has no :query")
	}

	if rulesNode != nil {
		if err := t.readRules(rulesNode); err != nil {
			return nil, err
		}
	}

	var q *Query
	switch queryNode.kind {
	case ednString:
		var queryText string
		queryText, result.TableView = cutTable(queryNode.text)
		q, err = Parse(queryText)
	case ednList:
		q, err = Parse(text[queryNode.pos:queryNode.end])
		if err != nil {
			var parseErr *ParseError
			if errors.As(err, &parseErr) {
				parseErr.Offset += queryNode.pos
			}
		}
	case ednVector:
		q, err = t.translate(queryNode, inputsNode)
	default:
		return nil, t.unsupported(queryNode, "expected a Datalog query or a simple query")
	}
	if err != nil {
		return nil, err
	}

	if transformNode != nil {
		transform, err := t.resultTransform(transformNode, q.FindsBlocks())
		if err != nil {
			return nil, err
		}

		// The transform runs on the results of the query, so a filter in it
		// can not run before a sample or limit of the query.
		if len(transform.filters) > 0 && (q.Sample > 0 || q.Limit > 0) {
			return nil, t.unsupported(transformNode, "filter and remove are not supported for a query with sample or limit")
		}
		if len(transform.sorts) > 0 && q.Limit > 0 {
			return nil, t.unsupported(transformNode, "sort-by is not supported for a query with limit")
		}

		q.Sort = append(q.Sort, transform.sorts...)
		q.Filter = withFilter(q.Filter, andOf(transform.filters))
		if transform.limit > 0 && (q.Limit == 0 || transform.limit < q.Limit) {
			q.Limit = transform.limit
		}
	}

	if tableViewNode != nil && isTruthy(tableViewNode) {
		result.TableView = true
	}

	switch {
	case result.TableView:
		result.GroupByPage = false
	case groupByPageNode != nil:
		result.GroupByPage = isTruthy(groupByPageNode)
	case transformNode != nil:
		result.GroupByPage = false
	default:
		result.GroupByPage = q.ByPage == nil || *q.ByPage
	}

	result.Query = q
	return result, nil
}

// isTruthy returns true if Logseq treats the value as true, which is all
// values except false and nil.
func isTruthy(node *ednNode) bool {
	return !node.isSymbol("false") && !node.isSymbol("nil")
}

// hiccupText returns the text in a title, which is either a string or Hiccup
// such as `[:b "Title"]`.
func hiccupText(node *ednNode) string {
	switch node.kind {
	case ednString:
		return node.text
	case ednVector, ednList:
		var b strings.Builder
		for _, item := range node.items {
			b.WriteString(hiccupText(item))
		}
		return b.String()
	}

	return ""
}

type entityKind int

const (
	entityUnknown entityKind = iota
	entityBlock
	entityPage
)

// sourceKind is what a variable holds the value of.
type sourceKind int

const (
	sourceMarker sourceKind = iota
	sourcePriority
	sourceScheduled
	sourceDeadline
	sourceJournalDay
	sourceContent
	sourceProperty
	sourceTimestamp
)

// source is where a variable gets its value from, such as the marker of a
// block or a property of a page.
type source struct {
	kind sourceKind

	// entity is the variable of the block or page the value belongs to.
	entity string

	// key is the name of the property for sourceProperty.
	key string

	// def is the value of a property that is not set, for sourceProperty
	// with a default value, such as "none" in `(get ?props :type "none")`.
	def string

	// hasDefault is true if def is set.
	hasDefault bool

	// timestamp is the time for sourceTimestamp.
	timestamp TimestampField
}

// condition is a clause such as `[(< ?d ?today)]` that compares the value of
// a variable.
type condition struct {
	node *ednNode
	fn   string
	args []*ednNode

	// subject is the variable the condition is about.
	subject string

	// used is set when the condition has been applied to a filter.
	used bool
}

type translator struct {
	src string

	// inputs are the values of the variables in :in.
	inputs map[string]*ednNode

	// pages are the titles of the pages that the inputs :current-page and
	// :query-page stand for, empty if they are not known.
	pages map[string]string

	// blocks are the blocks that the inputs :current-block and :parent-block
	// stand for, empty if they are not known.
	blocks map[string]blockRef

	// blockInputs maps the variables in :in that are blocks to the blocks.
	// They are kept out of inputs, as they can not be read as values.
	blockInputs map[string]blockRef

	// rules are the rules from :rules that find the blocks below a block, by
	// name.
	rules map[string]*recursiveRule

	// inlineRules are the other rules from :rules, by name. Their uses are
	// replaced with their clauses before the query is translated, see
	// expandRules.
	inlineRules map[string][]*ruleDef

	// expansions counts the uses of rules that have been replaced, to give
	// the variables of each use their own names.
	expansions int

	kinds map[string]entityKind
	// pageOf maps the variable of a page to the variable of a block on it,
	// from `[?b :block/page ?p]`.
	pageOf map[string]string
	// names maps the variable of a page to its title, from
	// `[?p :block/name "title"]`.
	names map[string]string
	// titles maps a variable for the title of a page to the clause that
	// gives it, such as `[?p :block/original-name ?title]`.
	titles map[string]*ednNode
	// props maps the variable of properties to the variable of the block or
	// page they belong to, from `[?p :block/properties ?props]`.
	props map[string]string
	// sources are the variables that hold values, in the current scope.
	sources map[string]*source

	result string
	// findsPages is true if the query finds pages. A query for pages with
	// clauses about the blocks on them, such as `[?b :block/page ?p]`, is
	// translated as a query for these blocks, with result set to the block,
	// and then finds the pages that have one or more of them.
	findsPages bool
	// block is the variable of the blocks the query is about.
	block string
	// page is the variable of the page of block, or of the pages the query
	// finds.
	page string
}

func (t *translator) unsupported(node *ednNode, format string, args ...any) error {
	form := strings.Join(strings.Fields(t.src[node.pos:node.end]), " ")
	return &UnsupportedError{
		Offset:  node.pos,
		Form:    form,
		Message: fmt.Sprintf(format, args...),
	}
}

func isVar(node *ednNode) bool {
	return node.kind == ednSymbol && strings.HasPrefix(node.text, "?")
}

func (t *translator) translate(queryNode *ednNode, inputsNode *ednNode) (*Query, error) {
	sections := make(map[string][]*ednNode)
	current := ""
	for _, item := range queryNode.items {
		if item.kind == ednKeyword {
			switch item.text {
			case "find", "in", "where":
				current = item.text
				continue
			default:
				return nil, t.unsupported(item, "the :%s section is not supported", item.text)
			}
		}

		if current == "" {
			return nil, t.unsupported(item, "expected :find, :in or :where")
		}
		sections[current] = append(sections[current], item)
	}

	if err := t.readInputs(queryNode, sections["in"], inputsNode); err != nil {
		return nil, err
	}

	where, err := t.expandRules(sections["where"])
	if err != nil {
		return nil, err
	}

	// The variables for the page of a block are renamed in :find as well,
	// as the query can find that page.
	renames := t.pageRenames(where)
	where = renameVars(where, renames)
	sections["where"] = where
	sections["find"] = renameVars(sections["find"], renames)

	for _, clause := range sections["where"] {
		if err := t.checkBlockInputs(clause); err != nil {
			return nil, err
		}
	}

	find := sections["find"]
	if len(find) != 1 {
		return nil, t.unsupported(queryNode, "expected :find with one variable or pull")
	}

	switch {
	case isVar(find[0]):
		t.result = find[0].text
	case find[0].head() == "pull" && len(find[0].items) == 3 && isVar(find[0].items[1]):
		t.result = find[0].items[1].text
	default:
		return nil, t.unsupported(find[0], "expected :find with one variable or pull")
	}

	t.kinds = make(map[string]entityKind)
	t.pageOf = make(map[string]string)
	t.names = make(map[string]string)
	t.titles = make(map[string]*ednNode)
	t.props = make(map[string]string)
	t.sources = make(map[string]*source)
	for _, clause := range where {
		if err := t.collect(clause, false); err != nil {
			return nil, err
		}
	}

	t.readTitles()

	if err := t.chooseEntities(find[0]); err != nil {
		return nil, err
	}

	t.findsPages = t.kinds[t.result] == entityPage
	if t.findsPages && t.block != "" {
		t.result = t.block
	}

	filter, err := t.conjunction(where, nil)
	if err != nil {
		return nil, err
	}

	if t.findsPages {
		filter = pagesWithBlocks(filter, t.block != "")
	} else if filter != nil && !findsBlocks(filter) {
		// A query for blocks with only conditions on their page.
		filter = OnPage(filter)
	}

	q := New(filter)
	if q.FindsBlocks() == t.findsPages {
		if q.FindsBlocks() {
			return nil, t.unsupported(find[0], "finding pages by the blocks on them is not supported")
		}
		return nil, t.unsupported(find[0], "finding blocks with only conditions on their page is not supported")
	}

	return q, nil
}

// pagesWithBlocks turns a filter for the blocks on the pages a query finds
// into a filter for these pages, with a HasBlockFilter for the parts that
// match blocks. In an `and`, the parts that only match pages are kept out of
// it, so that `(and (title = "x") (task TODO))` becomes
// `(and (title = "x") (has-block (task TODO)))`. If hasBlock is true the
// query has a block on the page, such as ?b in `[?b :block/page ?p]`, so a
// filter without conditions on blocks gets `(has-block)` for the pages with
// one or more blocks.
func pagesWithBlocks(filter Filter, hasBlock bool) Filter {
	if !findsBlocks(filter) {
		if hasBlock {
			return withFilter(filter, HasBlock(nil))
		}
		return filter
	}

	and, ok := filter.(*AndFilter)
	if !ok {
		return HasBlock(filter)
	}

	var pageFilters, blockFilters []Filter
	for _, f := range and.Filters {
		if findsBlocks(f) {
			blockFilters = append(blockFilters, f)
		} else {
			pageFilters = append(pageFilters, f)
		}
	}

	return andOf(append(pageFilters, HasBlock(andOf(blockFilters))))
}

func (t *translator) readInputs(queryNode *ednNode, in []*ednNode, inputsNode *ednNode) error {
	var vars []*ednNode
	for _, item := range in {
		switch {
		case item.isSymbol("$"), item.isSymbol("%"):
		case isVar(item):
			vars = append(vars, item)
		default:
			return t.unsupported(item, "expected a variable in :in")
		}
	}

	var inputs []*ednNode
	if inputsNode != nil {
		if inputsNode.kind != ednVector {
			return t.unsupported(inputsNode, "expected a vector of inputs")
		}
		inputs = inputsNode.items
	}

	if len(vars) != len(inputs) {
		return t.unsupported(queryNode, "the query has %d variables in :in but %d :inputs", len(vars), len(inputs))
	}

	for i, v := range vars {
		input := inputs[i]
		if title, isPage := t.pages[input.text]; isPage && input.kind == ednKeyword {
			if title == "" {
				return t.unsupported(input, "the input :%s needs the title of the page, given with %s", input.text, inputOptionNames[input.text])
			}

			// The input is read as the title of the page, the same as an
			// input such as "[[page]]".
			t.inputs[v.text] = &ednNode{kind: ednString, text: title, pos: input.pos, end: input.end}
			continue
		}

		if ref, isBlock := t.blocks[input.text]; isBlock && input.kind == ednKeyword {
			if !ref.known() {
				return t.unsupported(input, "the input :%s needs the block, given with %s, or found by ParseNode with the page given with WithQueryPage", input.text, inputOptionNames[input.text])
			}

			t.blockInputs[v.text] = ref
			continue
		}

		if input.kind == ednKeyword {
			if _, ok := utils.ParseQueryDateInput(input.text, time.Now()); !ok {
				return t.unsupported(input, "the input :%s is not supported", input.text)
			}
		}

		t.inputs[v.text] = input
	}

	return nil
}

// checkBlockInputs returns an error if a variable for a block from :in, such
// as :current-block, is used in other ways than as the parent in
// `[?b :block/parent ?block]`.
func (t *translator) checkBlockInputs(node *ednNode) error {
	if isDataPattern(node) && len(node.items) == 3 && node.items[1].text == "block/parent" {
		return nil
	}

	if _, isRule := t.rules[node.head()]; isRule && node.kind == ednList {
		// The rule checks its arguments when it is translated.
		return nil
	}

	if _, isBlock := t.blockInputs[node.text]; isBlock && isVar(node) {
		return t.unsupported(node, "a block from :in is only supported as the parent in [?b :block/parent %s] or in a rule for the blocks below it", node.text)
	}

	for _, item := range node.items {
		if err := t.checkBlockInputs(item); err != nil {
			return err
		}
	}

	return nil
}

var (
	blockAttributes = map[string]bool{
		"block/marker":    true,
		"block/priority":  true,
		"block/scheduled": true,
		"block/deadline":  true,
		"block/content":   true,
		"block/refs":      true,
		"block/path-refs": true,
		"block/page":      true,
		"block/parent":    true,
		"block/left":      true,
	}

	pageAttributes = map[string]bool{
		"block/name":          true,
		"block/original-name": true,
		"block/journal-day":   true,
		"block/journal?":      true,
		"block/tags":          true,
		"block/namespace":     true,
		"block/alias":         true,
		"page/properties":     true,
	}

	blockRules = map[string]bool{
		"task":     true,
		"todo":     true,
		"priority": true,
		"page-ref": true,
		"page":     true,
		"property": true,
		"between":  true,
	}

	pageRules = map[string]bool{
		"page-property": true,
		"page-tags":     true,
		"all-page-tags": true,
		"namespace":     true,
	}
)

// collect goes through a clause and its nested clauses to learn which
// variables are blocks and which are pages, before the clauses are translated.
// nested is true for clauses in an `or` or a `not`, which only apply in their
// own scope.
func (t *translator) collect(clause *ednNode, nested bool) error {
	switch clause.kind {
	case ednList:
		head := clause.head()
		switch head {
		case "and", "or", "not":
			for _, item := range clause.items[1:] {
				if err := t.collect(item, nested || head != "and"); err != nil {
					return err
				}
			}
		case "or-join", "not-join":
			if len(clause.items) > 1 {
				for _, item := range clause.items[2:] {
					if err := t.collect(item, true); err != nil {
						return err
					}
				}
			}
		default:
			if rule, ok := t.rules[head]; ok {
				if len(clause.items) != 3 {
					return nil
				}

				if rule.attribute == "block/namespace" {
					for _, arg := range clause.items[1:] {
						if isVar(arg) {
							if err := t.setKind(arg, entityPage); err != nil {
								return err
							}
						}
					}
					return nil
				}

				if isVar(clause.items[1+rule.child]) {
					return t.setKind(clause.items[1+rule.child], entityBlock)
				}
				return nil
			}

			if len(clause.items) > 1 && isVar(clause.items[1]) {
				if blockRules[head] {
					return t.setKind(clause.items[1], entityBlock)
				}
				if pageRules[head] {
					return t.setKind(clause.items[1], entityPage)
				}
			}
		}
	case ednVector:
		if isRegexBinding(clause) {
			return t.readRegex(clause)
		}

		if !isDataPattern(clause) {
			return nil
		}

		entity, attribute := clause.items[0], clause.items[1].text
		if !isVar(entity) {
			return nil
		}

		switch {
		case blockAttributes[attribute]:
			if err := t.setKind(entity, entityBlock); err != nil {
				return err
			}
		case pageAttributes[attribute]:
			if err := t.setKind(entity, entityPage); err != nil {
				return err
			}
		}

		if len(clause.items) < 3 {
			return nil
		}

		value := clause.items[2]
		switch attribute {
		case "block/page", "block/refs", "block/path-refs", "block/tags", "block/namespace", "block/alias":
			if isVar(value) {
				if err := t.setKind(value, entityPage); err != nil {
					return err
				}
			}
		}

		// A page of a block in an `or` or a `not` is only the page of the
		// block in that scope, see conjunction.
		if attribute == "block/page" && isVar(value) && !nested {
			t.pageOf[value.text] = entity.text
		}

		if attribute == "block/name" || attribute == "block/original-name" {
			if name, ok := t.name(value); ok {
				if previous, exists := t.names[entity.text]; exists && !strings.EqualFold(previous, name) {
					return t.unsupported(clause, "a variable for two different pages is not supported")
				}
				t.names[entity.text] = name
			} else if isVar(value) {
				t.titles[value.text] = clause
			}
		}
	}

	return nil
}

// readTitles reads a variable for the title of a page with a known title as
// that title, the same as an input. This is the ?title in
// `[?p :block/name ?page] [?p :block/original-name ?title]`, which can then
// be compared with a value, such as in `[(get ?values ?title)]`.
func (t *translator) readTitles() {
	for variable, clause := range t.titles {
		name, ok := t.names[clause.items[0].text]
		if !ok {
			continue
		}

		value := clause.items[2]
		t.inputs[variable] = &ednNode{kind: ednString, text: name, pos: value.pos, end: value.end}
	}
}

func (t *translator) setKind(node *ednNode, kind entityKind) error {
	if previous := t.kinds[node.text]; previous != entityUnknown && previous != kind {
		return t.unsupported(node, "%s is used as both a block and a page", node.text)
	}

	t.kinds[node.text] = kind
	return nil
}

// pageRenames returns new names for the variables that name the page of a
// block that another variable already names, such as ?q in
// `[?b :block/page ?p] (or ... [?b :block/page ?q])`. A block is on one page,
// so both variables are the same page, and the translator can only follow
// one page of the block. A page with a title keeps its variable, as the
// translator reads it as the page with that title, also in a `not`.
func (t *translator) pageRenames(clauses []*ednNode) map[string]string {
	named := make(map[string]bool)
	var findNamed func(node *ednNode)
	findNamed = func(node *ednNode) {
		if isDataPattern(node) && len(node.items) == 3 &&
			(node.items[1].text == "block/name" || node.items[1].text == "block/original-name") {
			if _, ok := t.name(node.items[2]); ok {
				named[node.items[0].text] = true
			}
			return
		}

		for _, item := range node.items {
			findNamed(item)
		}
	}
	for _, clause := range clauses {
		findNamed(clause)
	}

	pageOf := make(map[string]string)
	renames := make(map[string]string)

	var visit func(node *ednNode)
	visit = func(node *ednNode) {
		if isDataPattern(node) && len(node.items) == 3 && node.items[1].text == "block/page" {
			block, page := node.items[0], node.items[2]
			_, isInput := t.inputs[page.text]
			if isVar(block) && isVar(page) && !isInput && !named[page.text] {
				if first, ok := pageOf[block.text]; !ok {
					pageOf[block.text] = page.text
				} else if first != page.text {
					renames[page.text] = first
				}
			}
			return
		}

		for _, item := range node.items {
			visit(item)
		}
	}
	for _, clause := range clauses {
		visit(clause)
	}

	// A rename can lead to a variable that is renamed as well, such as when
	// two blocks share a page.
	for from := range renames {
		to := renames[from]
		for seen := 0; seen < len(renames); seen++ {
			next, ok := renames[to]
			if !ok || next == from {
				break
			}
			to = next
		}
		renames[from] = to
	}

	return renames
}

// renameVars returns the nodes with the variables renamed. Nodes without a
// renamed variable are kept as they are.
func renameVars(nodes []*ednNode, renames map[string]string) []*ednNode {
	if len(renames) == 0 {
		return nodes
	}

	result := make([]*ednNode, len(nodes))
	for i, node := range nodes {
		result[i] = renameVar(node, renames)
	}
	return result
}

func renameVar(node *ednNode, renames map[string]string) *ednNode {
	if isVar(node) {
		if to, ok := renames[node.text]; ok {
			renamed := *node
			renamed.text = to
			return &renamed
		}
		return node
	}

	if len(node.items) == 0 {
		return node
	}

	renamed := *node
	renamed.items = renameVars(node.items, renames)
	return &renamed
}

// isDataPattern returns true for a clause such as `[?b :block/marker ?m]`.
func isDataPattern(node *ednNode) bool {
	return node.kind == ednVector &&
		(len(node.items) == 2 || len(node.items) == 3) &&
		node.items[1].kind == ednKeyword
}

// isCondition returns true for a clause such as `[(< ?d ?today)]`.
func isCondition(node *ednNode) bool {
	return node.kind == ednVector && len(node.items) == 1 && node.items[0].kind == ednList
}

// isBinding returns true for a clause such as `[(get ?props :type) ?t]`.
func isBinding(node *ednNode) bool {
	return node.kind == ednVector && len(node.items) == 2 && node.items[0].kind == ednList && isVar(node.items[1])
}

// chooseEntities picks the variables of the block and the page that the
// query is about.
func (t *translator) chooseEntities(find *ednNode) error {
	kind := t.kinds[t.result]
	if kind == entityUnknown {
		kind = entityBlock
		if _, ok := t.pageOf[t.result]; ok {
			kind = entityPage
		}
		t.kinds[t.result] = kind
	}

	if kind == entityPage {
		t.page = t.result
		t.block = t.pageOf[t.result]
		return nil
	}

	t.block = t.result
	for page, block := range t.pageOf {
		if block != t.result {
			continue
		}

		if t.page != "" && t.page != page {
			return t.unsupported(find, "more than one page variable for the same block is not supported")
		}
		t.page = page
	}

	return nil
}

// name returns the page title that a value stands for, from a string, a
// keyword or an input.
func (t *translator) name(node *ednNode) (string, bool) {
	node = t.resolve(node)

	switch node.kind {
	case ednString:
		return pageNameOf(node.text), true
	case ednKeyword:
		return node.text, true
	}

	return "", false
}

// pageNameOf returns the title in `[[title]]`, or the text as it is.
func pageNameOf(text string) string {
	if strings.HasPrefix(text, "[[") && strings.HasSuffix(text, "]]") && len(text) > 4 {
		return text[2 : len(text)-2]
	}

	return text
}

// resolve returns the input for a variable in :in, or the node itself.
func (t *translator) resolve(node *ednNode) *ednNode {
	if isVar(node) {
		if input, ok := t.inputs[node.text]; ok {
			return input
		}
	}

	return node
}

// pageRefName returns the title of the page a value of :block/path-refs or a
// similar attribute points at: a lookup such as `[:block/name "title"]`, a
// variable with a title, or an input.
func (t *translator) pageRefName(node *ednNode) (string, bool) {
	if node.kind == ednVector && len(node.items) == 2 && node.items[0].kind == ednKeyword {
		switch node.items[0].text {
		case "block/name", "block/original-name":
			return t.name(node.items[1])
		}
		return "", false
	}

	if isVar(node) {
		if name, ok := t.names[node.text]; ok {
			return name, true
		}
	}

	return t.name(node)
}

// blockRef returns the block that a value of :block/parent points at: a
// variable for :current-block or :parent-block, or a lookup such as
// `[:block/uuid #uuid "..."]`.
func (t *translator) blockRef(node *ednNode) (blockRef, bool) {
	if isVar(node) {
		ref, ok := t.blockInputs[node.text]
		return ref, ok
	}

	if node.kind != ednVector || len(node.items) != 2 || !(node.items[0].kind == ednKeyword && node.items[0].text == "block/uuid") {
		return blockRef{}, false
	}

	value := node.items[1]
	if value.kind == ednTagged && value.text == "uuid" && len(value.items) == 1 {
		value = value.items[0]
	}

	if value.kind != ednString || value.text == "" {
		return blockRef{}, false
	}
	return blockRef{id: value.text}, true
}

// conjunction translates clauses that must all match. The conditions in
// pending come from an outer scope and are applied where their variables get
// their values.
func (t *translator) conjunction(clauses []*ednNode, pending []*condition) (Filter, error) {
	// In a query for pages, a scope with its own block on the page, such as
	// `(not [?b :block/page ?p] [?b :block/marker "TODO"])`, is about the
	// blocks on the page, and matches the pages with one or more of them.
	if block := t.blockOfPageIn(clauses); block != "" {
		savedBlock, savedResult := t.block, t.result
		t.block, t.result = block, block
		filter, err := t.translateConjunction(clauses, pending)
		t.block, t.result = savedBlock, savedResult
		if err != nil {
			return nil, err
		}
		return pagesWithBlocks(filter, true), nil
	}

	return t.translateConjunction(clauses, pending)
}

// joins translates the clauses about another page or block that the block of
// the query is related to into one filter for the block. It returns the
// clauses that are left and the filters. These are the relations:
//
//   - a page the block refers to, such as
//     `[?b :block/refs ?p] [?p :block/tags ?t] [?t :block/name "x"]`, into a
//     ReferencesFilter
//   - the block above, such as `[?b :block/parent ?p] [?p :block/marker ?m]`,
//     or a block above at any depth with a rule for the blocks below a block,
//     into a BelowFilter
//   - a block below, such as `[?c :block/parent ?b] [?c :block/marker ?m]`,
//     or at any depth with a rule for the blocks below a block, into a
//     HasChildFilter
//
// A page that only has its title compared with = or contains? is left to
// the clause itself, which then becomes a filter for each title, such as
// `(refs [[x]])`.
func (t *translator) joins(clauses []*ednNode, pending []*condition) ([]*ednNode, []Filter, error) {
	if t.block == "" || t.result != t.block {
		return clauses, nil, nil
	}

	var filters []Filter
	for i := 0; i < len(clauses); i++ {
		clause := clauses[i]
		j, ok := t.joinOf(clause)
		if !ok {
			continue
		}

		others := append(append([]*ednNode{}, clauses[:i]...), clauses[i+1:]...)
		about, rest, ok := t.clausesAbout(j.other, others)
		if !ok || (j.isPage && t.onlyTitles(j.other, about)) {
			continue
		}

		var filter Filter
		var err error
		if j.isPage {
			filter, err = t.pageFilter(j.other, about, pending)
		} else {
			filter, err = t.blockFilter(j.other, about, pending)
		}
		if err != nil {
			return nil, nil, err
		}
		if filter == nil {
			return nil, nil, t.unsupported(clause, "expected conditions on %s", j.other)
		}

		filters = append(filters, j.filter(filter))
		clauses = rest
		i = -1
	}

	return clauses, filters, nil
}

// join is a relation between the block of the query and another page or
// block, see joins.
type join struct {
	// other is the variable of the other page or block.
	other string

	// isPage is true if other is a page.
	isPage bool

	// filter returns the filter for the block of the query from the filter
	// for other.
	filter func(Filter) Filter
}

// joinOf returns the join in a clause, if it has one.
func (t *translator) joinOf(clause *ednNode) (join, bool) {
	// isOther returns true for a variable that can be translated on its own.
	isOther := func(node *ednNode) bool {
		_, isInput := t.inputs[node.text]
		_, isBlockInput := t.blockInputs[node.text]
		return isVar(node) && !isInput && !isBlockInput && node.text != t.block && node.text != t.page
	}
	isBlock := func(node *ednNode) bool {
		return isOther(node) && t.kinds[node.text] != entityPage
	}

	if rule, ok := t.rules[clause.head()]; ok && clause.kind == ednList && len(clause.items) == 3 {
		if rule.attribute != "block/parent" {
			return join{}, false
		}

		child, ancestor := clause.items[1+rule.child], clause.items[1+rule.ancestor]
		switch {
		case child.text == t.block && isBlock(ancestor):
			return join{other: ancestor.text, filter: func(f Filter) Filter { return DescendantOf(f) }}, true
		case ancestor.text == t.block && isBlock(child):
			return join{other: child.text, filter: func(f Filter) Filter { return HasDescendant(f) }}, true
		}
		return join{}, false
	}

	if !isDataPattern(clause) || len(clause.items) != 3 {
		return join{}, false
	}

	entity, attribute, value := clause.items[0], clause.items[1].text, clause.items[2]
	switch {
	case attribute == "block/refs" && entity.text == t.block && isOther(value):
		if _, named := t.names[value.text]; !named {
			return join{other: value.text, isPage: true, filter: func(f Filter) Filter { return RefsTo(f) }}, true
		}
	case attribute == "block/path-refs" && entity.text == t.block && isOther(value):
		if _, named := t.names[value.text]; !named {
			return join{other: value.text, isPage: true, filter: func(f Filter) Filter { return PageRefTo(f) }}, true
		}
	case attribute == "block/parent" && entity.text == t.block && isBlock(value):
		return join{other: value.text, filter: func(f Filter) Filter { return ChildOf(f) }}, true
	case attribute == "block/parent" && value.text == t.block && isBlock(entity):
		return join{other: entity.text, filter: func(f Filter) Filter { return HasChild(f) }}, true
	}

	return join{}, false
}

// blockFilter translates the clauses about another block into a filter for
// blocks, the same as for the blocks a query finds. Conditions that are only
// on the page of the block are matched with an OnPageFilter.
func (t *translator) blockFilter(block string, clauses []*ednNode, pending []*condition) (Filter, error) {
	savedPage, savedBlock, savedResult, savedFindsPages := t.page, t.block, t.result, t.findsPages
	t.page, t.block, t.result, t.findsPages = "", block, block, false

	filter, err := t.scoped(func() (Filter, error) {
		return t.conjunction(clauses, pending)
	})

	t.page, t.block, t.result, t.findsPages = savedPage, savedBlock, savedResult, savedFindsPages
	if err != nil || filter == nil || findsBlocks(filter) {
		return filter, err
	}
	return OnPage(filter), nil
}

// clausesAbout returns the clauses that are about another page or block,
// directly or through the variables they give values, and the clauses that
// are not. Returns false if one of them is also about the block or the page
// the query is about, as the other can then not be translated on its own.
func (t *translator) clausesAbout(other string, clauses []*ednNode) ([]*ednNode, []*ednNode, bool) {
	vars := map[string]bool{other: true}
	taken := make([]bool, len(clauses))

	for changed := true; changed; {
		changed = false
		for i, clause := range clauses {
			if taken[i] {
				continue
			}

			clauseVars := varsOf(clause, nil)
			related := false
			for _, v := range clauseVars {
				if vars[v] {
					related = true
				}
			}
			if !related {
				continue
			}

			taken[i], changed = true, true
			for _, v := range clauseVars {
				if v == t.block || (t.page != "" && v == t.page) {
					return nil, nil, false
				}

				_, isInput := t.inputs[v]
				_, isBlockInput := t.blockInputs[v]
				if !isInput && !isBlockInput {
					vars[v] = true
				}
			}
		}
	}

	var about, rest []*ednNode
	for i, clause := range clauses {
		if taken[i] {
			about = append(about, clause)
		} else {
			rest = append(rest, clause)
		}
	}
	return about, rest, len(about) > 0
}

// varsOf appends the variables in a clause and the clauses in it to vars.
func varsOf(node *ednNode, vars []string) []string {
	if isVar(node) {
		return append(vars, node.text)
	}

	for _, item := range node.items {
		vars = varsOf(item, vars)
	}
	return vars
}

// onlyTitles returns true if the clauses only give the title of a page a
// variable and compare it with = or contains?, such as
// `[?p :block/name ?n] [(contains? #{"a" "b"} ?n)]`.
func (t *translator) onlyTitles(page string, clauses []*ednNode) bool {
	for _, clause := range clauses {
		if isDataPattern(clause) && len(clause.items) == 3 && clause.items[0].text == page &&
			(clause.items[1].text == "block/name" || clause.items[1].text == "block/original-name") {
			continue
		}

		if !isCondition(clause) {
			return false
		}

		c, err := t.condition(clause)
		if err != nil {
			return false
		}

		if title, isTitle := t.titles[c.subject]; !isTitle || title.items[0].text != page {
			return false
		}

		isSet := c.fn == "contains?" && len(c.args) == 2 && c.args[1].isSymbol(c.subject)
		if c.fn != "=" && !isSet {
			return false
		}
	}

	return true
}

// pageFilter translates the clauses about a page that a block refers to into
// a filter for pages, the same as for the pages a query finds.
func (t *translator) pageFilter(page string, clauses []*ednNode, pending []*condition) (Filter, error) {
	savedPage, savedBlock, savedResult, savedFindsPages := t.page, t.block, t.result, t.findsPages
	t.page, t.block, t.result, t.findsPages = page, "", page, true

	filter, err := t.scoped(func() (Filter, error) {
		return t.conjunction(clauses, pending)
	})

	t.page, t.block, t.result, t.findsPages = savedPage, savedBlock, savedResult, savedFindsPages
	if err != nil {
		return nil, err
	}

	if findsBlocks(filter) {
		return nil, t.unsupported(clauses[0], "expected conditions on the page %s, not on blocks", page)
	}
	return filter, nil
}

// blockOfPageIn returns the variable of a block on the page that a query for
// pages finds, from a clause such as `[?b :block/page ?p]` in the clauses,
// when it is not the block the query is already about. Returns an empty
// string if there is no such block.
func (t *translator) blockOfPageIn(clauses []*ednNode) string {
	if !t.findsPages {
		return ""
	}

	for _, clause := range clauses {
		if !isDataPattern(clause) || len(clause.items) != 3 || clause.items[1].text != "block/page" {
			continue
		}

		entity, value := clause.items[0], clause.items[2]
		if isVar(entity) && entity.text != t.block && value.text == t.page {
			return entity.text
		}
	}

	return ""
}

func (t *translator) translateConjunction(clauses []*ednNode, pending []*condition) (Filter, error) {
	t.choosePageInScope(clauses)

	clauses, joins, err := t.joins(clauses, pending)
	if err != nil {
		return nil, err
	}

	var local []*condition
	var bindings, lists, missing []*ednNode

	for _, clause := range clauses {
		switch {
		case isCondition(clause) && clause.items[0].head() == "missing?":
			missing = append(missing, clause)
		case isCondition(clause):
			c, err := t.condition(clause)
			if err != nil {
				return nil, err
			}
			local = append(local, c)
		case clause.kind == ednList:
			lists = append(lists, clause)
		case clause.kind == ednVector:
			bindings = append(bindings, clause)
		default:
			return nil, t.unsupported(clause, "expected a clause")
		}
	}

	available := append(append([]*condition{}, pending...), local...)

	filters := joins
	for _, clause := range bindings {
		filter, err := t.binding(clause, available)
		if err != nil {
			return nil, err
		}
		filters = appendFilter(filters, filter)
	}

	for _, clause := range lists {
		filter, err := t.list(clause, unused(available))
		if err != nil {
			return nil, err
		}
		filters = appendFilter(filters, filter)
	}

	for _, clause := range missing {
		filter, err := t.missingFilter(clause)
		if err != nil {
			return nil, err
		}
		filters = appendFilter(filters, filter)
	}

	// Conditions on variables that got their values in an outer scope, such
	// as in `(not [(= ?m "DONE")])`, apply to that value here.
	for _, c := range local {
		if c.used {
			continue
		}

		if _, isProps := t.props[c.subject]; isProps {
			filter, err := t.propertyCondition(c)
			if err != nil {
				return nil, err
			}
			filters = appendFilter(filters, filter)
			continue
		}

		src, ok := t.sources[c.subject]
		if !ok {
			if clause, isTitle := t.titles[c.subject]; isTitle && clause.items[0].text == t.page {
				filter, err := t.pageTitleFilter([]*condition{c})
				if err != nil {
					return nil, err
				}
				filters = appendFilter(filters, filter)
				continue
			}

			if _, isTitle := t.titles[c.subject]; isTitle {
				return nil, t.unsupported(c.node, "conditions on the title of a page are only supported for a page that a block is on or refers to, such as in [?b :block/refs ?p]")
			}
			return nil, t.unsupported(c.node, "%s does not get a value in the query", c.subject)
		}

		filter, err := t.sourceFilter(src, []*condition{c})
		if err != nil {
			return nil, err
		}
		filters = appendFilter(filters, filter)
	}

	return andOf(filters), nil
}

// choosePageInScope makes ?p the page of the block for the clauses, from a
// clause such as `[?b :block/page ?p]` in a branch of an `or` or in a `not`.
// The scope keeps it until it ends, see scoped. A page with a title is not
// chosen, as its title can come from a clause outside the scope, and the
// clause is translated as a filter for the page with that title instead.
func (t *translator) choosePageInScope(clauses []*ednNode) {
	if t.page != "" || t.block == "" {
		return
	}

	for _, clause := range clauses {
		if !isDataPattern(clause) || len(clause.items) != 3 || clause.items[1].text != "block/page" || clause.items[0].text != t.block {
			continue
		}

		value := clause.items[2]
		_, isInput := t.inputs[value.text]
		_, named := t.names[value.text]
		if isVar(value) && !isInput && !named {
			t.page = value.text
			return
		}
	}
}

func unused(conditions []*condition) []*condition {
	var result []*condition
	for _, c := range conditions {
		if !c.used {
			result = append(result, c)
		}
	}
	return result
}

func appendFilter(filters []Filter, filter Filter) []Filter {
	if filter == nil {
		return filters
	}
	return append(filters, filter)
}

// andOf combines filters that must all match. Returns nil for no filters, as
// that matches everything.
func andOf(filters []Filter) Filter {
	switch len(filters) {
	case 0:
		return nil
	case 1:
		return filters[0]
	}

	return And(filters...)
}

func (t *translator) condition(clause *ednNode) (*condition, error) {
	call := clause.items[0]
	if call.head() == "" {
		return nil, t.unsupported(clause, "expected a function in the condition")
	}

	c := &condition{
		node: clause,
		fn:   call.head(),
		args: call.items[1:],
	}

	for _, arg := range c.args {
		if !isVar(arg) {
			continue
		}

		if _, isInput := t.inputs[arg.text]; isInput {
			continue
		}

		if c.subject != "" {
			return nil, t.unsupported(clause, "comparing two variables is not supported")
		}
		c.subject = arg.text
	}

	if c.subject == "" {
		return nil, t.unsupported(clause, "the condition has no variable")
	}

	return c, nil
}

// equalTo returns a condition that the value of a clause is equal to the
// value of the node, for values written in the clause itself, such as the
// "TODO" in `[?b :block/marker "TODO"]`.
func equalTo(value *ednNode) *condition {
	const subject = "?_value"
	return &condition{
		node:    value,
		fn:      "=",
		args:    []*ednNode{{kind: ednSymbol, text: subject, pos: value.pos, end: value.end}, value},
		subject: subject,
		used:    true,
	}
}

// take marks the conditions about a variable as used and returns them.
func take(conditions []*condition, variable string) []*condition {
	var result []*condition
	for _, c := range conditions {
		if c.subject == variable {
			c.used = true
			result = append(result, c)
		}
	}
	return result
}

// bind gives a variable its value from a source, and returns the filter for
// the conditions on it.
func (t *translator) bind(variable *ednNode, src *source, conditions []*condition) (Filter, error) {
	if variable.isSymbol("_") {
		return t.sourceFilter(src, nil)
	}

	if !isVar(variable) {
		return nil, t.unsupported(variable, "expected a variable")
	}

	if _, isInput := t.inputs[variable.text]; isInput {
		return t.sourceFilter(src, []*condition{equalTo(variable)})
	}

	t.sources[variable.text] = src
	return t.sourceFilter(src, take(conditions, variable.text))
}

func (t *translator) binding(clause *ednNode, conditions []*condition) (Filter, error) {
	if isRegexBinding(clause) {
		// The variable is an input, see readRegex.
		return nil, nil
	}

	if isBinding(clause) {
		return t.functionBinding(clause, conditions)
	}

	if !isDataPattern(clause) {
		return nil, t.unsupported(clause, "expected a clause such as [?b :block/marker ?m]")
	}

	entity := clause.items[0]
	attribute := clause.items[1].text
	if !isVar(entity) {
		return nil, t.unsupported(clause, "expected a variable as the first value")
	}

	var value *ednNode
	if len(clause.items) == 3 {
		value = clause.items[2]
	} else {
		value = &ednNode{kind: ednSymbol, text: "_", pos: clause.pos, end: clause.end}
	}

	switch attribute {
	case "block/page":
		if _, isInput := t.inputs[value.text]; isVar(value) && !isInput {
			if value.text == t.page {
				// This names the page of the block. A title for it is
				// translated with its :block/name, and conditions on the
				// title here.
				return t.pageTitleFilter(conditions)
			}

			// A page with a title from another clause, such as in
			// `(or [?b :block/refs ?p] [?b :block/page ?p])`.
			name, named := t.names[value.text]
			if !named {
				return nil, t.unsupported(clause, "expected the page of the blocks the query finds, or a page with a title")
			}

			if err := t.checkBlock(clause, entity); err != nil {
				return nil, err
			}
			return Page(name), nil
		}

		if err := t.checkBlock(clause, entity); err != nil {
			return nil, err
		}

		name, ok := t.pageRefName(value)
		if !ok {
			return nil, t.unsupported(clause, "expected a page")
		}
		return Page(name), nil
	case "block/name", "block/original-name":
		name, named := t.name(value)
		if !named {
			// The clause only gives the title a variable. Conditions on it
			// are translated with the clause that uses the page, see
			// titleFilter.
			return nil, nil
		}

		if entity.text == t.page {
			if t.findsPages {
				return Title(TitleEquals, name), nil
			}
			return Page(name), nil
		}

		// The variable is a page that another clause refers to, such as a
		// page in :block/path-refs.
		return nil, nil
	case "block/journal?":
		if err := t.checkPageOfBlock(clause, entity); err != nil {
			return nil, err
		}

		var journal Filter = OnJournal()
		if t.findsPages {
			journal = IsJournalPage()
		}

		switch {
		case value.isSymbol("true"):
			return journal, nil
		case value.isSymbol("false"):
			return Not(journal), nil
		}
		return nil, t.unsupported(clause, "expected true or false")
	case "block/journal-day":
		if err := t.checkPageOfBlock(clause, entity); err != nil {
			return nil, err
		}
		return t.bind(value, &source{kind: sourceJournalDay, entity: entity.text}, conditions)
	case "block/marker", "block/priority", "block/scheduled", "block/deadline", "block/content":
		if err := t.checkBlock(clause, entity); err != nil {
			return nil, err
		}

		kinds := map[string]sourceKind{
			"block/marker":    sourceMarker,
			"block/priority":  sourcePriority,
			"block/scheduled": sourceScheduled,
			"block/deadline":  sourceDeadline,
			"block/content":   sourceContent,
		}
		src := &source{kind: kinds[attribute], entity: entity.text}

		if !isVar(value) && !value.isSymbol("_") {
			// A value such as "TODO" in [?b :block/marker "TODO"].
			return t.sourceFilter(src, []*condition{equalTo(value)})
		}
		return t.bind(value, src, conditions)
	case "block/created-at", "block/updated-at":
		if entity.text != t.block && entity.text != t.page {
			return nil, t.unsupported(clause, "times of other blocks or pages are not supported")
		}
		if entity.text == t.block && t.result != t.block {
			return nil, t.unsupported(clause, "finding pages by the blocks on them is not supported")
		}

		src := &source{kind: sourceTimestamp, entity: entity.text, timestamp: CreatedAt}
		if attribute == "block/updated-at" {
			src.timestamp = UpdatedAt
		}

		if !isVar(value) && !value.isSymbol("_") {
			return t.sourceFilter(src, []*condition{equalTo(value)})
		}
		return t.bind(value, src, conditions)
	case "block/properties", "page/properties":
		if entity.text != t.block && entity.text != t.page {
			return nil, t.unsupported(clause, "properties of other blocks or pages are not supported")
		}

		if !isVar(value) {
			return nil, t.unsupported(clause, "expected a variable for the properties")
		}

		t.props[value.text] = entity.text
		return nil, nil
	case "block/parent":
		if err := t.checkBlock(clause, entity); err != nil {
			return nil, err
		}

		ref, ok := t.blockRef(value)
		if ok {
			return ref.parent(), nil
		}

		// The parent of a block at the top of a page is the page.
		if t.kinds[value.text] == entityPage || value.text == t.page || value.kind == ednVector {
			if name, ok := t.pageRefName(value); ok {
				return ParentAt(name), nil
			}

			// Conditions on the page of the block are translated with
			// the clauses about it.
			if value.text == t.page && entity.text == t.block {
				return TopLevel(), nil
			}
		}
		return nil, t.unsupported(clause, "expected a block, such as :current-block or [:block/uuid #uuid \"...\"], or a page with a title")
	case "block/refs", "block/path-refs", "block/tags", "block/namespace", "block/alias":
		if attribute == "block/tags" || attribute == "block/namespace" || attribute == "block/alias" {
			if entity.text != t.page {
				return nil, t.unsupported(clause, "%s of other pages is not supported", attribute)
			}
		} else if err := t.checkBlock(clause, entity); err != nil {
			return nil, err
		}

		pageFilter := func(names []string) Filter {
			switch attribute {
			case "block/refs":
				return orOf(names, func(name string) Filter { return Refs(name) })
			case "block/path-refs":
				return orOf(names, func(name string) Filter { return PageRef(name) })
			case "block/tags":
				return PageTags(names...)
			case "block/alias":
				return orOf(names, func(name string) Filter { return Alias(name) })
			}
			return orOf(names, func(name string) Filter { return Namespace(name) })
		}

		if name, ok := t.pageRefName(value); ok {
			return pageFilter([]string{name}), nil
		}

		if isVar(value) {
			filter, err := t.titleFilter(value.text, conditions, pageFilter)
			if err != nil || filter != nil {
				return filter, err
			}
		}
		return nil, t.unsupported(clause, "expected a page, such as [:block/name \"page\"]")
	}

	return nil, t.unsupported(clause, "the attribute :%s is not supported", attribute)
}

// checkBlock returns an error if a variable is not the block the query finds.
func (t *translator) checkBlock(clause *ednNode, entity *ednNode) error {
	if entity.text != t.block {
		return t.unsupported(clause, "conditions on other blocks are not supported")
	}

	if t.result != t.block {
		return t.unsupported(clause, "finding pages by the blocks on them is not supported")
	}

	return nil
}

// checkPageOfBlock returns an error if a variable is not the page of the
// block the query finds.
func (t *translator) checkPageOfBlock(clause *ednNode, entity *ednNode) error {
	if entity.text != t.page {
		return t.unsupported(clause, "only the journal of the pages the query finds, or of the page of the blocks, is supported")
	}

	return nil
}

// functionBinding translates `[(get ?props :type) ?t]`, which gives ?t the
// value of a property.
func (t *translator) functionBinding(clause *ednNode, conditions []*condition) (Filter, error) {
	call := clause.items[0]
	out := clause.items[1]

	var entity, key string
	var def *ednNode
	switch {
	case call.head() == "get" && (len(call.items) == 3 || len(call.items) == 4) && isVar(call.items[1]):
		owner, ok := t.props[call.items[1].text]
		if !ok {
			return nil, t.unsupported(clause, "expected the properties of a block or page")
		}

		name, ok := keyName(call.items[2])
		if !ok {
			return nil, t.unsupported(clause, "expected the name of a property")
		}
		entity, key = owner, name
		if len(call.items) == 4 {
			def = call.items[3]
		}
	case call.head() == "get-in" && (len(call.items) == 3 || len(call.items) == 4) && isVar(call.items[1]):
		path := call.items[2]
		if path.kind != ednVector || len(path.items) != 2 || path.items[0].kind != ednKeyword ||
			(path.items[0].text != "block/properties" && path.items[0].text != "page/properties") {
			return nil, t.unsupported(clause, "expected a path such as [:block/properties :type]")
		}

		name, ok := keyName(path.items[1])
		if !ok {
			return nil, t.unsupported(clause, "expected the name of a property")
		}
		entity, key = call.items[1].text, name
		if len(call.items) == 4 {
			def = call.items[3]
		}
	default:
		return nil, t.unsupported(clause, "only getting a property with get is supported")
	}

	src := &source{kind: sourceProperty, entity: entity, key: key}
	if def != nil && !def.isSymbol("nil") {
		value, ok := t.scalar(def)
		if !ok {
			return nil, t.unsupported(def, "expected a value as the default")
		}
		src.def, src.hasDefault = value, true
	}

	if entity != t.block && entity != t.page {
		return nil, t.unsupported(clause, "properties of other blocks or pages are not supported")
	}

	if entity == t.block && t.result != t.block {
		return nil, t.unsupported(clause, "finding pages by the blocks on them is not supported")
	}

	return t.bind(out, src, conditions)
}

// orOf returns a filter that matches one of the names.
func orOf(names []string, filter func(name string) Filter) Filter {
	filters := make([]Filter, 0, len(names))
	for _, name := range names {
		filters = append(filters, filter(name))
	}

	if len(filters) == 1 {
		return filters[0]
	}
	return Or(filters...)
}

// titleFilter translates conditions on the title of the page in a variable,
// such as `[(contains? #{"a" "b"} ?n)]` with `[?p :block/name ?n]`, with
// filter for the titles that a condition allows. Returns nil if there are no
// such conditions.
func (t *translator) titleFilter(page string, conditions []*condition, filter func(names []string) Filter) (Filter, error) {
	matches, err := t.titleMatches(page, conditions)
	if err != nil {
		return nil, err
	}

	filters := make([]Filter, 0, len(matches))
	for _, m := range matches {
		if m.op != TitleEquals {
			return nil, t.unsupported(m.node, "only = and contains? are supported for the title of a page that a block refers to")
		}
		filters = append(filters, filter(m.values))
	}

	return andOf(filters), nil
}

// pageTitleFilter translates conditions on the title of the page the query is
// about: the page of the blocks, or the pages the query finds. Returns nil if
// there are no such conditions.
func (t *translator) pageTitleFilter(conditions []*condition) (Filter, error) {
	matches, err := t.titleMatches(t.page, conditions)
	if err != nil {
		return nil, err
	}

	filters := make([]Filter, 0, len(matches))
	for _, m := range matches {
		filters = append(filters, orOf(m.values, func(value string) Filter {
			if m.op == TitleEquals && !t.findsPages {
				return Page(value)
			}
			return Title(m.op, value)
		}))
	}

	return andOf(filters), nil
}

// titleMatch is a condition on the title of a page, which is one of values.
type titleMatch struct {
	node   *ednNode
	op     TitleOp
	values []string
}

var titleFunctions = map[string]TitleOp{
	"clojure.string/starts-with?": TitleStartsWith,
	"clojure.string/ends-with?":   TitleEndsWith,
	"clojure.string/includes?":    TitleIncludes,
}

// titleMatches reads the conditions on the title of the page in a variable,
// such as `[(contains? #{"a" "b"} ?n)]` or
// `[(clojure.string/starts-with? ?n "projects/")]` with `[?p :block/name ?n]`,
// and marks them as used.
func (t *translator) titleMatches(page string, conditions []*condition) ([]titleMatch, error) {
	var matches []titleMatch
	for _, c := range conditions {
		clause, isTitle := t.titles[c.subject]
		if !isTitle || clause.items[0].text != page {
			continue
		}
		c.used = true

		m := titleMatch{node: c.node, op: TitleEquals}
		switch op, isFunction := titleFunctions[c.fn]; {
		case c.fn == "contains?" && len(c.args) == 2 && c.args[1].isSymbol(c.subject):
			values, ok := t.strings(c.args[0])
			if !ok || len(values) == 0 {
				return nil, t.unsupported(c.node, "expected a set of titles")
			}
			m.values = values
		case c.fn == "=":
			value, err := t.comparedValue(c, "=")
			if err != nil {
				return nil, err
			}
			m.values = []string{value}
		case isFunction && len(c.args) == 2 && c.args[0].isSymbol(c.subject):
			value, ok := t.scalar(c.args[1])
			if !ok {
				return nil, t.unsupported(c.node, "expected the text to compare the title with")
			}
			m.op, m.values = op, []string{value}
		case regexFunctions[c.fn] && len(c.args) == 2 && c.args[1].isSymbol(c.subject):
			pattern, err := t.regex(c)
			if err != nil {
				return nil, err
			}
			m.op, m.values = TitleMatches, []string{pattern}
		default:
			return nil, t.unsupported(c.node, "only =, contains?, clojure.string/starts-with?, ends-with?, includes?, re-find and re-matches are supported for the title of a page")
		}

		matches = append(matches, m)
	}

	return matches, nil
}

// contentFilter translates a condition on the content of a block, such as
// `[(clojure.string/starts-with? ?c "TODO")]` with `[?b :block/content ?c]`.
// The text is used as it is, so `"[[page]]"` is a reference in the text and
// not the title of the page.
func (t *translator) contentFilter(c *condition) (Filter, error) {
	if len(c.args) == 2 {
		switch {
		case c.fn == "=":
			other := c.args[1]
			if other.isSymbol(c.subject) {
				other = c.args[0]
			}

			text, ok := t.text(other)
			if !ok {
				return nil, t.unsupported(c.node, "expected the text to compare the content with")
			}
			return Content(TitleEquals, text), nil
		case regexFunctions[c.fn] && c.args[1].isSymbol(c.subject):
			pattern, err := t.regex(c)
			if err != nil {
				return nil, err
			}
			return Content(TitleMatches, pattern), nil
		}
	}

	op, isFunction := titleFunctions[c.fn]
	if !isFunction || len(c.args) != 2 || !c.args[0].isSymbol(c.subject) {
		return nil, t.unsupported(c.node, "only =, clojure.string/starts-with?, ends-with?, includes?, re-find and re-matches are supported for the content")
	}

	text, ok := t.text(c.args[1])
	if !ok {
		return nil, t.unsupported(c.node, "expected the text to compare the content with")
	}
	return Content(op, text), nil
}

// regexFunctions are the functions that match a regular expression, with the
// expression as the first argument.
var regexFunctions = map[string]bool{
	"re-find":    true,
	"re-matches": true,
}

// regex returns the regular expression of a condition such as
// `[(re-find #"bank\d" ?c)]`. For re-matches, which matches all of the value,
// the expression is put between `^` and `$`.
func (t *translator) regex(c *condition) (string, error) {
	node := t.resolve(c.args[0])
	if node.kind != ednRegex {
		return "", t.unsupported(c.node, "expected a regular expression, such as #\"bank\" or (re-pattern \"bank\")")
	}

	if _, err := syntax.Parse(node.text, syntax.Perl); err != nil {
		return "", t.unsupported(c.node, "the regular expression is not supported: %v", err)
	}

	if c.fn == "re-matches" {
		return "^(?:" + node.text + ")$", nil
	}
	return node.text, nil
}

// isRegexBinding returns true for a clause such as
// `[(re-pattern "bank") ?re]`.
func isRegexBinding(node *ednNode) bool {
	return isBinding(node) && node.items[0].head() == "re-pattern"
}

// readRegex reads the variable of a clause such as `[(re-pattern "bank") ?re]`
// as an input with the regular expression, so that conditions such as
// `[(re-find ?re ?c)]` can use it.
func (t *translator) readRegex(clause *ednNode) error {
	call := clause.items[0]
	if len(call.items) != 2 {
		return t.unsupported(clause, "re-pattern takes one text")
	}

	value := t.resolve(call.items[1])
	if value.kind != ednString && value.kind != ednRegex {
		return t.unsupported(clause, "expected the text of a regular expression")
	}

	out := clause.items[1]
	t.inputs[out.text] = &ednNode{kind: ednRegex, text: value.text, pos: value.pos, end: value.end}
	return nil
}

// propertyCondition translates a condition on the properties of a block or
// page itself, such as `[(get ?props :type)]` or `[(contains? ?props :type)]`,
// which match when the property is set.
func (t *translator) propertyCondition(c *condition) (Filter, error) {
	if (c.fn != "get" && c.fn != "contains?") || len(c.args) < 2 || !c.args[0].isSymbol(c.subject) ||
		(c.fn == "contains?" && len(c.args) != 2) || len(c.args) > 3 {
		return nil, t.unsupported(c.node, "expected (get %s :key) for the properties", c.subject)
	}

	key, ok := keyName(c.args[1])
	if !ok {
		return nil, t.unsupported(c.node, "expected the name of a property")
	}

	if len(c.args) == 3 && !c.args[2].isSymbol("nil") && !c.args[2].isSymbol("false") {
		// With a default, every block or page has a value.
		return nil, nil
	}

	owner := t.props[c.subject]
	switch {
	case owner == t.block && t.result == t.block:
		return HasProperty(key), nil
	case owner == t.page:
		return HasPageProperty(key), nil
	case owner == t.block:
		return nil, t.unsupported(c.node, "finding pages by the blocks on them is not supported")
	}
	return nil, t.unsupported(c.node, "properties of other blocks or pages are not supported")
}

// missingFilter translates `[(missing? $ ?b :block/scheduled)]`, which matches
// the blocks without a value for the attribute.
func (t *translator) missingFilter(clause *ednNode) (Filter, error) {
	args := clause.items[0].items[1:]
	if len(args) != 3 || !args[0].isSymbol("$") || !isVar(args[1]) || args[2].kind != ednKeyword {
		return nil, t.unsupported(clause, "expected (missing? $ ?b :block/scheduled)")
	}

	entity, attribute := args[1], args[2].text
	kinds := map[string]sourceKind{
		"block/marker":    sourceMarker,
		"block/priority":  sourcePriority,
		"block/scheduled": sourceScheduled,
		"block/deadline":  sourceDeadline,
	}

	var src *source
	if kind, ok := kinds[attribute]; ok {
		if err := t.checkBlock(clause, entity); err != nil {
			return nil, err
		}
		src = &source{kind: kind, entity: entity.text}
	} else if attribute == "block/journal-day" {
		if err := t.checkPageOfBlock(clause, entity); err != nil {
			return nil, err
		}
		src = &source{kind: sourceJournalDay, entity: entity.text}
	} else {
		return nil, t.unsupported(clause, "missing? is only supported for :block/marker, :block/priority, :block/scheduled, :block/deadline and :block/journal-day")
	}

	exists, err := t.sourceFilter(src, nil)
	if err != nil {
		return nil, err
	}
	return Not(exists), nil
}

// keyName returns the name of a property from a keyword or a string.
func keyName(node *ednNode) (string, bool) {
	switch node.kind {
	case ednKeyword, ednString:
		return node.text, node.text != ""
	}

	return "", false
}

func (t *translator) list(clause *ednNode, conditions []*condition) (Filter, error) {
	head := clause.head()
	if head == "" {
		return nil, t.unsupported(clause, "expected a clause such as (not ...) or a rule")
	}
	args := clause.items[1:]

	switch head {
	case "and":
		return t.conjunction(args, conditions)
	case "or", "or-join":
		branches := args
		if head == "or-join" {
			if len(args) == 0 || args[0].kind != ednVector {
				return nil, t.unsupported(clause, "expected the variables to join")
			}
			branches = args[1:]
		}

		if len(branches) == 0 {
			return nil, t.unsupported(clause, "expected one or more branches")
		}

		filters := make([]Filter, 0, len(branches))
		for _, branch := range branches {
			clauses := []*ednNode{branch}
			if branch.head() == "and" {
				clauses = branch.items[1:]
			}

			filter, err := t.scoped(func() (Filter, error) {
				return t.conjunction(clauses, conditions)
			})
			if err != nil {
				return nil, err
			}

			if filter == nil {
				// A branch that matches everything makes the whole `or` match
				// everything.
				return nil, nil
			}
			filters = append(filters, filter)
		}

		if len(filters) == 1 {
			return filters[0], nil
		}
		return Or(filters...), nil
	case "not", "not-join":
		clauses := args
		if head == "not-join" {
			if len(args) == 0 || args[0].kind != ednVector {
				return nil, t.unsupported(clause, "expected the variables to join")
			}
			clauses = args[1:]
		}

		filter, err := t.scoped(func() (Filter, error) {
			return t.conjunction(clauses, nil)
		})
		if err != nil {
			return nil, err
		}

		if filter == nil {
			return nil, t.unsupported(clause, "a `not` that matches everything is not supported")
		}
		return Not(filter), nil
	}

	return t.rule(clause)
}

// scoped runs f with the variables that get values inside it kept out of the
// scope outside it.
func (t *translator) scoped(f func() (Filter, error)) (Filter, error) {
	saved := make(map[string]*source, len(t.sources))
	for k, v := range t.sources {
		saved[k] = v
	}
	savedPage := t.page

	filter, err := f()
	t.sources = saved
	t.page = savedPage
	return filter, err
}

// rule translates a rule from simple queries, such as `(task ?b #{"TODO"})`.
func (t *translator) rule(clause *ednNode) (Filter, error) {
	head := clause.head()
	if rule, ok := t.rules[head]; ok {
		return t.ancestorFilter(clause, rule)
	}

	if head == "" || len(clause.items) < 2 || !isVar(clause.items[1]) {
		return nil, t.unsupported(clause, "expected a rule such as (task ?b #{\"TODO\"})")
	}

	entity := clause.items[1]
	args := clause.items[2:]

	switch {
	case blockRules[head]:
		if err := t.checkBlock(clause, entity); err != nil {
			return nil, err
		}
	case pageRules[head]:
		if entity.text != t.page {
			return nil, t.unsupported(clause, "conditions on other pages are not supported")
		}
	default:
		return nil, t.unsupported(clause, "the rule %s is not supported", head)
	}

	names := func() ([]string, error) {
		var result []string
		for _, arg := range args {
			values, ok := t.strings(arg)
			if !ok {
				return nil, t.unsupported(arg, "expected a name or a set of names")
			}
			result = append(result, values...)
		}

		if len(result) == 0 {
			return nil, t.unsupported(clause, "expected one or more names")
		}
		return result, nil
	}

	switch head {
	case "task", "todo":
		values, err := names()
		if err != nil {
			return nil, err
		}
		return t.taskFilter(clause, values)
	case "priority":
		values, err := names()
		if err != nil {
			return nil, err
		}
		return t.priorityFilter(clause, values)
	case "page-ref", "page", "namespace":
		values, err := names()
		if err != nil {
			return nil, err
		}
		if len(values) != 1 {
			return nil, t.unsupported(clause, "expected one page")
		}

		switch head {
		case "page-ref":
			return PageRef(values[0]), nil
		case "page":
			return Page(values[0]), nil
		}
		return Namespace(values[0]), nil
	case "page-tags":
		values, err := names()
		if err != nil {
			return nil, err
		}
		return PageTags(values...), nil
	case "all-page-tags":
		return AllPageTags(), nil
	case "property", "page-property":
		if len(args) == 0 || len(args) > 2 {
			return nil, t.unsupported(clause, "expected a property and a value")
		}

		key, ok := keyName(t.resolve(args[0]))
		if !ok {
			return nil, t.unsupported(args[0], "expected the name of a property")
		}

		value := ""
		if len(args) == 2 {
			v, ok := t.scalar(args[1])
			if !ok {
				return nil, t.unsupported(args[1], "expected a value")
			}
			value = v
		}

		if head == "property" {
			return Property(key, value), nil
		}
		return PageProperty(key, value), nil
	case "between":
		if len(args) != 2 {
			return nil, t.unsupported(clause, "expected two dates")
		}

		start, err := t.date(args[0])
		if err != nil {
			return nil, err
		}

		end, err := t.date(args[1])
		if err != nil {
			return nil, err
		}
		return Between(start, end), nil
	}

	return nil, t.unsupported(clause, "the rule %s is not supported", head)
}

// recursiveRule is a rule from :rules that uses itself to step along an
// attribute at any depth. With :block/parent it finds the blocks below a
// block:
//
//	[(below ?b ?parent) [?b :block/parent ?parent]]
//	[(below ?b ?parent) [?b :block/parent ?c] (below ?c ?parent)]
//
// With :block/namespace it finds the pages in a namespace at any depth:
//
//	[(in-ns ?p ?ns) [?p :block/namespace ?ns]]
//	[(in-ns ?p ?ns) [?p :block/namespace ?mid] (in-ns ?mid ?ns)]
type recursiveRule struct {
	// attribute is the attribute that the rule steps along,
	// "block/parent" or "block/namespace".
	attribute string

	// child and ancestor are the positions of the arguments for the block
	// or page below, and for the one above it.
	child, ancestor int
}

// recursiveAttributes are the attributes that a recursive rule can step
// along, see recursiveRule.
var recursiveAttributes = []string{"block/parent", "block/namespace"}

// readRules reads :rules, which is a vector of rule definitions such as
// `[(name ?a ?b) clause ...]`. The definitions with the same name make one
// rule, which matches when one of them matches.
func (t *translator) readRules(node *ednNode) error {
	if node.kind != ednVector {
		return t.unsupported(node, "expected a vector of rules")
	}

	var names []string
	definitions := make(map[string][]*ednNode)
	for _, def := range node.items {
		if def.kind != ednVector || len(def.items) < 2 || def.items[0].head() == "" {
			return t.unsupported(def, "expected a rule such as [(name ?a ?b) clause ...]")
		}

		name := def.items[0].head()
		if _, seen := definitions[name]; !seen {
			names = append(names, name)
		}
		definitions[name] = append(definitions[name], def)
	}

	for _, name := range names {
		defs := definitions[name]
		if rule, ok := readRecursiveRule(name, defs); ok {
			t.rules[name] = rule
			continue
		}

		for _, def := range defs {
			ruleDef, err := t.readRuleDef(def)
			if err != nil {
				return err
			}

			if len(t.inlineRules[name]) > 0 && len(ruleDef.params) != len(t.inlineRules[name][0].params) {
				return t.unsupported(def, "the definitions of the rule %s take different numbers of arguments", name)
			}
			t.inlineRules[name] = append(t.inlineRules[name], ruleDef)
		}
	}

	// Rules that use themselves can not be replaced with their clauses, as
	// that never ends.
	for _, name := range names {
		if t.usesRule(name, name, make(map[string]bool)) {
			return t.unsupported(definitions[name][0], "rules that use themselves are only supported to find the blocks below a block with :block/parent, or the pages in a namespace with :block/namespace")
		}
	}

	return nil
}

// ruleDef is one definition of a rule from :rules, such as
// `[(tagged ?b ?tag) [?b :block/refs ?p] [?p :block/name ?tag]]`.
type ruleDef struct {
	// params are the variables of the arguments, such as ?b and ?tag.
	params []string

	// clauses are the clauses that match when the rule matches.
	clauses []*ednNode
}

// readRuleDef reads one definition of a rule. The variables that a rule
// requires to have values can be written in a vector first, such as
// `(name [?b] ?tag)`, and are read the same as the others.
func (t *translator) readRuleDef(def *ednNode) (*ruleDef, error) {
	head := def.items[0]

	var params []string
	for _, item := range head.items[1:] {
		vars := []*ednNode{item}
		if item.kind == ednVector {
			vars = item.items
		}

		for _, v := range vars {
			if !isVar(v) {
				return nil, t.unsupported(v, "expected a variable as the argument of the rule")
			}
			params = append(params, v.text)
		}
	}

	return &ruleDef{params: params, clauses: def.items[1:]}, nil
}

// usesRule returns true if the rule name uses the rule target, directly or
// through other rules.
func (t *translator) usesRule(name string, target string, seen map[string]bool) bool {
	if seen[name] {
		return false
	}
	seen[name] = true

	for _, def := range t.inlineRules[name] {
		for _, clause := range def.clauses {
			for _, used := range ruleUses(clause) {
				if used == target || t.usesRule(used, target, seen) {
					return true
				}
			}
		}
	}

	return false
}

// ruleUses returns the names of the rules that a clause uses, also in the
// clauses of an and, or or not.
func ruleUses(clause *ednNode) []string {
	if clause.kind != ednList {
		return nil
	}

	switch head := clause.head(); head {
	case "and", "or", "not", "or-join", "not-join":
		var names []string
		for _, item := range clause.items[1:] {
			names = append(names, ruleUses(item)...)
		}
		return names
	case "":
		return nil
	default:
		return []string{head}
	}
}

// expandRules replaces the uses of the rules in inlineRules with their
// clauses, also in the clauses of an and, or or not. A rule with one
// definition is replaced with its clauses, and a rule with more than one with
// an `or-join` that has the clauses of each definition as a branch.
func (t *translator) expandRules(clauses []*ednNode) ([]*ednNode, error) {
	var result []*ednNode
	for _, clause := range clauses {
		expanded, err := t.expandRule(clause)
		if err != nil {
			return nil, err
		}
		result = append(result, expanded...)
	}
	return result, nil
}

func (t *translator) expandRule(clause *ednNode) ([]*ednNode, error) {
	if clause.kind != ednList {
		return []*ednNode{clause}, nil
	}

	head := clause.head()
	switch head {
	case "and", "not":
		items, err := t.expandRules(clause.items[1:])
		if err != nil {
			return nil, err
		}
		return []*ednNode{withItems(clause, clause.items[:1], items)}, nil
	case "not-join":
		if len(clause.items) < 2 {
			return []*ednNode{clause}, nil
		}

		items, err := t.expandRules(clause.items[2:])
		if err != nil {
			return nil, err
		}
		return []*ednNode{withItems(clause, clause.items[:2], items)}, nil
	case "or", "or-join":
		start := 1
		if head == "or-join" {
			start = 2
		}
		if len(clause.items) < start {
			return []*ednNode{clause}, nil
		}

		branches := make([]*ednNode, 0, len(clause.items)-start)
		for _, branch := range clause.items[start:] {
			expanded, err := t.expandRule(branch)
			if err != nil {
				return nil, err
			}
			branches = append(branches, asBranch(branch, expanded))
		}
		return []*ednNode{withItems(clause, clause.items[:start], branches)}, nil
	}

	defs, isRule := t.inlineRules[head]
	if !isRule {
		return []*ednNode{clause}, nil
	}

	args := clause.items[1:]
	if len(args) != len(defs[0].params) {
		return nil, t.unsupported(clause, "the rule %s takes %d arguments", head, len(defs[0].params))
	}

	branches := make([][]*ednNode, 0, len(defs))
	for _, def := range defs {
		t.expansions++
		replacements := make(map[string]*ednNode, len(def.params))
		for i, param := range def.params {
			replacements[param] = args[i]
		}

		clauses := make([]*ednNode, 0, len(def.clauses))
		for _, c := range def.clauses {
			clauses = append(clauses, t.replaceVars(c, replacements))
		}

		expanded, err := t.expandRules(clauses)
		if err != nil {
			return nil, err
		}
		branches = append(branches, expanded)
	}

	if len(branches) == 1 {
		return branches[0], nil
	}

	// The branches join on the variables given as arguments, the other
	// variables of each definition are its own.
	join := &ednNode{kind: ednVector, pos: clause.pos, end: clause.end}
	for _, arg := range args {
		if isVar(arg) {
			join.items = append(join.items, arg)
		}
	}

	orJoin := &ednNode{kind: ednList, pos: clause.pos, end: clause.end, items: []*ednNode{
		{kind: ednSymbol, text: "or-join", pos: clause.pos, end: clause.pos},
		join,
	}}
	for _, branch := range branches {
		orJoin.items = append(orJoin.items, asBranch(clause, branch))
	}
	return []*ednNode{orJoin}, nil
}

// replaceVars returns a copy of a clause from a rule with the parameters
// replaced with the arguments, and the other variables given names of their
// own for this use of the rule.
func (t *translator) replaceVars(node *ednNode, replacements map[string]*ednNode) *ednNode {
	if isVar(node) {
		if arg, ok := replacements[node.text]; ok {
			return arg
		}

		own := &ednNode{kind: ednSymbol, text: fmt.Sprintf("%s__%d", node.text, t.expansions), pos: node.pos, end: node.end}
		replacements[node.text] = own
		return own
	}

	if len(node.items) == 0 {
		return node
	}

	items := make([]*ednNode, 0, len(node.items))
	for _, item := range node.items {
		items = append(items, t.replaceVars(item, replacements))
	}
	return withItems(node, nil, items)
}

// withItems returns a copy of a node with the items first followed by rest.
func withItems(node *ednNode, first []*ednNode, rest []*ednNode) *ednNode {
	copied := *node
	copied.items = append(append([]*ednNode{}, first...), rest...)
	return &copied
}

// asBranch returns the clauses of a branch of an `or` as one clause, in an
// `and` if there is more than one.
func asBranch(at *ednNode, clauses []*ednNode) *ednNode {
	if len(clauses) == 1 {
		return clauses[0]
	}

	return &ednNode{kind: ednList, pos: at.pos, end: at.end, items: append([]*ednNode{
		{kind: ednSymbol, text: "and", pos: at.pos, end: at.pos},
	}, clauses...)}
}

// readRecursiveRule reads the two definitions of a recursive rule, see
// recursiveRule: one where the block or page is directly below, and one that
// steps up and uses the rule again. The recursive definition can step from
// either end:
//
//	[(name ?b ?a) [?b :block/parent ?c] (name ?c ?a)]
//	[(name ?b ?a) [?c :block/parent ?a] (name ?b ?c)]
func readRecursiveRule(name string, defs []*ednNode) (*recursiveRule, bool) {
	if len(defs) != 2 {
		return nil, false
	}

	for _, attribute := range recursiveAttributes {
		if rule, ok := readRecursiveRuleFor(name, defs, attribute); ok {
			return rule, true
		}
	}
	return nil, false
}

func readRecursiveRuleFor(name string, defs []*ednNode, attribute string) (*recursiveRule, bool) {
	params := func(def *ednNode) ([]string, bool) {
		head := def.items[0]
		if len(head.items) != 3 || !isVar(head.items[1]) || !isVar(head.items[2]) || head.items[1].text == head.items[2].text {
			return nil, false
		}
		return []string{head.items[1].text, head.items[2].text}, true
	}

	// parentOf returns the variables of the block and of its parent in a
	// clause such as `[?b :block/parent ?c]`.
	parentOf := func(clause *ednNode) (string, string, bool) {
		if !isDataPattern(clause) || len(clause.items) != 3 || clause.items[1].text != attribute ||
			!isVar(clause.items[0]) || !isVar(clause.items[2]) {
			return "", "", false
		}
		return clause.items[0].text, clause.items[2].text, true
	}

	for i, base := range defs {
		recursive := defs[1-i]

		baseParams, ok := params(base)
		if !ok || len(base.items) != 2 {
			continue
		}

		child, parent, ok := parentOf(base.items[1])
		if !ok {
			continue
		}

		rule := &recursiveRule{attribute: attribute}
		switch {
		case child == baseParams[0] && parent == baseParams[1]:
			rule.child, rule.ancestor = 0, 1
		case child == baseParams[1] && parent == baseParams[0]:
			rule.child, rule.ancestor = 1, 0
		default:
			continue
		}

		if isRecursiveStep(name, recursive, rule, params, parentOf) {
			return rule, true
		}
	}

	return nil, false
}

// isRecursiveStep returns true if a definition is the recursive step of a
// recursive rule, see readRecursiveRule.
func isRecursiveStep(
	name string,
	def *ednNode,
	rule *recursiveRule,
	params func(*ednNode) ([]string, bool),
	parentOf func(*ednNode) (string, string, bool),
) bool {
	vars, ok := params(def)
	if !ok || len(def.items) != 3 {
		return false
	}
	below, above := vars[rule.child], vars[rule.ancestor]

	var call *ednNode
	var child, parent string
	for _, clause := range def.items[1:] {
		if c, p, ok := parentOf(clause); ok {
			child, parent = c, p
		} else if clause.kind == ednList && clause.head() == name && len(clause.items) == 3 {
			call = clause
		}
	}

	if call == nil || child == "" || !isVar(call.items[1]) || !isVar(call.items[2]) {
		return false
	}

	callBelow, callAbove := call.items[1+rule.child].text, call.items[1+rule.ancestor].text
	isParam := func(v string) bool { return v == below || v == above }

	// Step up from the block: its parent is below the block above.
	if child == below && !isParam(parent) && callBelow == parent && callAbove == above {
		return true
	}

	// Step down from the block above: the block is below one of its children.
	return parent == above && !isParam(child) && callBelow == below && callAbove == child
}

// ancestorFilter translates a use of an ancestor rule, such as
// `(below ?b ?parent)`, into an AncestorFilter.
func (t *translator) ancestorFilter(clause *ednNode, rule *recursiveRule) (Filter, error) {
	if len(clause.items) != 3 {
		return nil, t.unsupported(clause, "expected two arguments for the rule")
	}

	if rule.attribute == "block/namespace" {
		return t.namespaceFilter(clause, rule)
	}

	if err := t.checkBlock(clause, clause.items[1+rule.child]); err != nil {
		return nil, err
	}

	ref, ok := t.blockRef(clause.items[1+rule.ancestor])
	if !ok {
		return nil, t.unsupported(clause, "expected a block above, such as :current-block or [:block/uuid #uuid \"...\"]")
	}
	return ref.ancestor(), nil
}

// namespaceFilter translates a use of a rule that finds the pages in a
// namespace at any depth, such as `(in-ns ?p ?ns)`. With a title for the
// namespace it matches the pages in it, see UnderNamespace. With a title for
// the page it matches the namespaces the page is in, which its title names.
func (t *translator) namespaceFilter(clause *ednNode, rule *recursiveRule) (Filter, error) {
	child, ancestor := clause.items[1+rule.child], clause.items[1+rule.ancestor]

	byTitle := func(name string) Filter {
		if t.findsPages {
			return Title(TitleEquals, name)
		}
		return Page(name)
	}

	switch {
	case child.text == t.page:
		name, ok := t.pageRefName(ancestor)
		if !ok {
			return nil, t.unsupported(clause, "expected a namespace with a title, such as [:block/name \"projects\"]")
		}
		return UnderNamespace(name), nil
	case ancestor.text == t.page:
		name, ok := t.pageRefName(child)
		if !ok {
			return nil, t.unsupported(clause, "expected a page with a title, such as [:block/name \"projects/a\"]")
		}

		namespaces := utils.NamespacesOf(name)
		if len(namespaces) == 0 {
			// The page is in no namespace, so nothing matches.
			return Or(), nil
		}
		return orOf(namespaces, byTitle), nil
	}

	return nil, t.unsupported(clause, "expected the pages the query finds, or the page of the blocks, as one of the arguments")
}

// strings returns the strings in a value, which is a string, a keyword, a
// symbol, a set or vector of them, or an input.
func (t *translator) strings(node *ednNode) ([]string, bool) {
	node = t.resolve(node)

	switch node.kind {
	case ednString:
		return []string{pageNameOf(node.text)}, true
	case ednKeyword, ednSymbol:
		if isVar(node) {
			return nil, false
		}
		return []string{node.text}, true
	case ednSet, ednVector:
		var result []string
		for _, item := range node.items {
			values, ok := t.strings(item)
			if !ok {
				return nil, false
			}
			result = append(result, values...)
		}
		return result, true
	}

	return nil, false
}

// text returns a single value as text, the same as scalar, but keeps a
// string as it is, also when it looks like a reference to a page.
func (t *translator) text(node *ednNode) (string, bool) {
	if resolved := t.resolve(node); resolved.kind == ednString {
		return resolved.text, true
	}
	return t.scalar(node)
}

// scalar returns a single value as text, from a string, keyword, number or
// input.
func (t *translator) scalar(node *ednNode) (string, bool) {
	node = t.resolve(node)

	switch node.kind {
	case ednString:
		return pageNameOf(node.text), true
	case ednKeyword, ednNumber:
		return node.text, true
	case ednSymbol:
		if isVar(node) {
			return "", false
		}
		return node.text, true
	}

	return "", false
}

var journalDayPattern = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`)

// date returns the date a value stands for: a date input such as `:today`,
// a journal day such as 20240115, or the title of a journal page.
func (t *translator) date(node *ednNode) (Date, error) {
	resolved := t.resolve(node)

	switch resolved.kind {
	case ednKeyword:
		input, ok := utils.ParseQueryDateInput(resolved.text, time.Now())
		if !ok {
			return Date{}, t.unsupported(node, "expected a date")
		}

		if input.HasTime {
			return Date{}, t.unsupported(node, "inputs with a time of day are not supported for dates")
		}
		return Relative(normalizeDateInput(resolved.text)), nil
	case ednNumber:
		if m := journalDayPattern.FindStringSubmatch(resolved.text); m != nil {
			year, _ := strconv.Atoi(m[1])
			month, _ := strconv.Atoi(m[2])
			day, _ := strconv.Atoi(m[3])
			date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
			if date.Year() == year && int(date.Month()) == month && date.Day() == day {
				return On(date), nil
			}
		}
	case ednString:
		return JournalDate(pageNameOf(resolved.text)), nil
	}

	return Date{}, t.unsupported(node, "expected a date")
}

var legacyDateInput = regexp.MustCompile(`^(\d+)d(-before|-after)?$`)

// normalizeDateInput writes the older date inputs, such as `3d-after`, in the
// form that Logseq uses now, such as `+3d`.
func normalizeDateInput(input string) string {
	if m := legacyDateInput.FindStringSubmatch(input); m != nil {
		if m[2] == "-after" {
			return "+" + m[1] + "d"
		}
		return "-" + m[1] + "d"
	}

	return input
}

func (t *translator) taskFilter(node *ednNode, values []string) (Filter, error) {
	statuses := make([]content.TaskStatus, 0, len(values))
	for _, value := range values {
		status, ok := taskStatuses[strings.ToUpper(value)]
		if !ok {
			return nil, t.unsupported(node, "unknown task status %q", value)
		}
		statuses = append(statuses, status)
	}
	return Task(statuses...), nil
}

func (t *translator) priorityFilter(node *ednNode, values []string) (Filter, error) {
	priorities := make([]content.Priority, 0, len(values))
	for _, value := range values {
		priority, ok := priorityValues[strings.ToUpper(value)]
		if !ok {
			return nil, t.unsupported(node, "unknown priority %q", value)
		}
		priorities = append(priorities, priority)
	}
	return Priority(priorities...), nil
}

// allTaskStatuses is every task status, which is what a block with any marker
// matches.
func allTaskStatuses() []content.TaskStatus {
	var statuses []content.TaskStatus
	for status := content.TaskStatusTodo; status.String() != ""; status++ {
		statuses = append(statuses, status)
	}
	return statuses
}

// sourceFilter returns the filter for the conditions on a value. Without
// conditions it is the filter for having the value at all.
func (t *translator) sourceFilter(src *source, conditions []*condition) (Filter, error) {
	switch src.kind {
	case sourceMarker, sourcePriority:
		return t.valueFilter(src, conditions)
	case sourceScheduled, sourceDeadline, sourceJournalDay:
		return t.dateFilter(src, conditions)
	case sourceTimestamp:
		return t.timestampFilter(src, conditions)
	case sourceContent:
		var filters []Filter
		for _, c := range conditions {
			filter, err := t.contentFilter(c)
			if err != nil {
				return nil, err
			}
			filters = append(filters, filter)
		}
		return andOf(filters), nil
	case sourceProperty:
		isPage := src.entity == t.page

		var has Filter = HasProperty(src.key)
		if isPage {
			has = HasPageProperty(src.key)
		}

		var filters []Filter
		for _, c := range conditions {
			compare, isComparison, err := t.propertyComparison(c, src, isPage)
			if err != nil {
				return nil, err
			}
			if isComparison {
				filters = append(filters, compare)
				continue
			}

			var values []string
			if c.fn == "contains?" && len(c.args) == 2 && c.args[1].isSymbol(c.subject) {
				// One of a set of values, such as (contains? #{"a" "b"} ?v).
				set, ok := t.strings(c.args[0])
				if !ok || len(set) == 0 {
					return nil, t.unsupported(c.node, "expected a set of values")
				}
				values = set
			} else {
				value, err := t.comparedValue(c, "get", "contains?", "=")
				if err != nil {
					return nil, err
				}
				values = []string{value}
			}

			filters = append(filters, orOf(values, func(value string) Filter {
				var filter Filter = Property(src.key, value)
				if isPage {
					filter = PageProperty(src.key, value)
				}

				// The default is the value for the blocks or pages without
				// the property.
				if src.hasDefault && value == src.def {
					filter = Or(filter, Not(has))
				}
				return filter
			}))
		}

		if len(filters) == 0 {
			if src.hasDefault {
				// Every block or page has a value, if only the default.
				return nil, nil
			}
			return has, nil
		}
		return andOf(filters), nil
	}

	return nil, fmt.Errorf("query: unknown source %d", src.kind)
}

// propertyComparison translates a condition that compares the value of a
// property as a number or a date, such as `[(> ?rating 7)]` or
// `[(< ?due ?today)]`, into a PropertyCompareFilter. Returns false for other
// conditions, which includes `=` with a value that is not a date input, as
// that compares the text of the value.
func (t *translator) propertyComparison(c *condition, src *source, isPage bool) (Filter, bool, error) {
	op, ok := ops[c.fn]
	if !ok || len(c.args) != 2 {
		return nil, false, nil
	}

	other := c.args[1]
	switch {
	case c.args[1].isSymbol(c.subject):
		op, other = flippedOps[op], c.args[0]
	case !c.args[0].isSymbol(c.subject):
		return nil, false, nil
	}

	resolved := t.resolve(other)
	isDateInput := resolved.kind == ednKeyword
	if isDateInput {
		_, isDateInput = utils.ParseQueryDateInput(resolved.text, time.Now())
	}
	if op == Equal && !isDateInput {
		return nil, false, nil
	}

	filter := &PropertyCompareFilter{Name: src.key, Op: op, Page: isPage}
	switch {
	case isDateInput:
		date, err := t.date(other)
		if err != nil {
			return nil, false, err
		}
		filter.Date = date
	case resolved.kind == ednNumber:
		number, err := strconv.ParseFloat(resolved.text, 64)
		if err != nil {
			return nil, false, t.unsupported(c.node, "expected a number to compare with")
		}
		filter.Number = number
	case resolved.kind == ednString:
		date, err := time.ParseInLocation("2006-01-02", resolved.text, time.Local)
		if err != nil {
			return nil, false, t.unsupported(c.node, "expected a number or a date such as \"2024-01-15\" to compare with")
		}
		filter.Date = On(date)
	default:
		return nil, false, t.unsupported(c.node, "expected a number or a date to compare with")
	}

	if !src.hasDefault {
		return filter, true, nil
	}

	passes, err := t.defaultPasses(c, src, filter)
	if err != nil || !passes {
		return filter, true, err
	}

	// The default is the value for the blocks or pages without the
	// property, so they match as well.
	var has Filter = HasProperty(src.key)
	if isPage {
		has = HasPageProperty(src.key)
	}
	return Or(filter, Not(has)), true, nil
}

// defaultPasses returns true if the default value of a property passes the
// comparison of the filter, such as 0 in `(get ?props :rating 0)` with
// `[(< ?rating 7)]`. Only a number is supported as the default, and only
// when it is compared with a number.
func (t *translator) defaultPasses(c *condition, src *source, filter *PropertyCompareFilter) (bool, error) {
	if !filter.Date.IsZero() {
		return false, t.unsupported(c.node, "comparing a date property that has a default value is not supported")
	}

	def, err := strconv.ParseFloat(src.def, 64)
	if err != nil {
		return false, t.unsupported(c.node, "expected a number as the default value of a property that is compared with a number")
	}

	switch filter.Op {
	case Equal:
		return def == filter.Number, nil
	case Less:
		return def < filter.Number, nil
	case LessOrEqual:
		return def <= filter.Number, nil
	case Greater:
		return def > filter.Number, nil
	case GreaterOrEqual:
		return def >= filter.Number, nil
	}
	return false, fmt.Errorf("query: unknown comparison %d", filter.Op)
}

// comparedValue returns the value a condition such as `[(= ?t "book")]`
// compares its variable with, for one of the given functions.
func (t *translator) comparedValue(c *condition, functions ...string) (string, error) {
	supported := false
	for _, fn := range functions {
		if c.fn == fn {
			supported = true
		}
	}

	if !supported || len(c.args) != 2 {
		return "", t.unsupported(c.node, "only %s are supported here", strings.Join(functions, ", "))
	}

	other := c.args[1]
	if c.args[1].isSymbol(c.subject) {
		if c.fn != "=" {
			return "", t.unsupported(c.node, "expected %s as the first argument", c.subject)
		}
		other = c.args[0]
	}

	value, ok := t.scalar(other)
	if !ok {
		return "", t.unsupported(c.node, "expected a value to compare with")
	}
	return value, nil
}

// valueFilter returns the filter for conditions on a marker or a priority.
func (t *translator) valueFilter(src *source, conditions []*condition) (Filter, error) {
	makeFilter := func(node *ednNode, values []string) (Filter, error) {
		if src.kind == sourceMarker {
			return t.taskFilter(node, values)
		}
		return t.priorityFilter(node, values)
	}

	var exists Filter
	if src.kind == sourceMarker {
		exists = Task(allTaskStatuses()...)
	} else {
		exists = Priority(content.PriorityA, content.PriorityB, content.PriorityC)
	}

	var filters []Filter
	negated := false
	for _, c := range conditions {
		switch c.fn {
		case "contains?":
			if len(c.args) != 2 || !c.args[1].isSymbol(c.subject) {
				return nil, t.unsupported(c.node, "expected (contains? #{...} %s)", c.subject)
			}

			values, ok := t.strings(c.args[0])
			if !ok || len(values) == 0 {
				return nil, t.unsupported(c.node, "expected a set of values")
			}

			filter, err := makeFilter(c.node, values)
			if err != nil {
				return nil, err
			}
			filters = append(filters, filter)
		case "=", "not=":
			if len(c.args) != 2 {
				return nil, t.unsupported(c.node, "expected two values to compare")
			}

			other := c.args[0]
			if other.isSymbol(c.subject) {
				other = c.args[1]
			}

			value, ok := t.scalar(other)
			if !ok {
				return nil, t.unsupported(c.node, "expected a value to compare with")
			}

			filter, err := makeFilter(c.node, []string{value})
			if err != nil {
				return nil, err
			}

			if c.fn == "not=" {
				negated = true
				filter = Not(filter)
			}
			filters = append(filters, filter)
		default:
			return nil, t.unsupported(c.node, "only =, not= and contains? are supported here")
		}
	}

	// A negated condition still requires the block to have the value.
	if len(filters) == 0 || negated {
		filters = append([]Filter{exists}, filters...)
	}

	return andOf(filters), nil
}

var flippedOps = map[Op]Op{
	Less:           Greater,
	LessOrEqual:    GreaterOrEqual,
	Greater:        Less,
	GreaterOrEqual: LessOrEqual,
	Equal:          Equal,
}

// dateFilter returns the filter for comparisons of a date.
func (t *translator) dateFilter(src *source, conditions []*condition) (Filter, error) {
	field := map[sourceKind]DateField{
		sourceScheduled:  FieldScheduled,
		sourceDeadline:   FieldDeadline,
		sourceJournalDay: FieldJournal,
	}[src.kind]
	if src.kind == sourceJournalDay && t.findsPages {
		field = FieldJournalPage
	}

	if len(conditions) == 0 {
		return &DateFilter{Field: field, Op: Exists}, nil
	}

	var filters []*DateFilter
	for _, c := range conditions {
		op, ok := ops[c.fn]
		if !ok || len(c.args) != 2 {
			return nil, t.unsupported(c.node, "only =, <, <=, > and >= are supported for dates")
		}

		other := c.args[1]
		if c.args[1].isSymbol(c.subject) {
			op = flippedOps[op]
			other = c.args[0]
		} else if !c.args[0].isSymbol(c.subject) {
			return nil, t.unsupported(c.node, "expected %s in the comparison", c.subject)
		}

		date, err := t.date(other)
		if err != nil {
			return nil, err
		}

		filters = append(filters, &DateFilter{Field: field, Op: op, Date: date})
	}

	if len(filters) == 2 {
		if rangeFilter, ok := asRange(filters[0], filters[1]); ok {
			return rangeFilter, nil
		}
	}

	result := make([]Filter, 0, len(filters))
	for _, f := range filters {
		result = append(result, f)
	}
	return andOf(result), nil
}

// timestampFilter returns the filter for conditions on the time a block or
// page was created or last changed, such as `[(> ?u ?since)]` with
// `:inputs [:-7d-ms]`. Logseq stores these times in milliseconds, so they are
// compared with inputs that have a time of day, or with numbers.
func (t *translator) timestampFilter(src *source, conditions []*condition) (Filter, error) {
	isPage := src.entity == t.page
	if len(conditions) == 0 {
		return &TimestampFilter{Field: src.timestamp, Op: Exists, Page: isPage}, nil
	}

	var filters []Filter
	for _, c := range conditions {
		op, ok := ops[c.fn]
		if !ok || len(c.args) != 2 {
			return nil, t.unsupported(c.node, "only =, <, <=, > and >= are supported for times")
		}

		other := c.args[1]
		if c.args[1].isSymbol(c.subject) {
			op = flippedOps[op]
			other = c.args[0]
		} else if !c.args[0].isSymbol(c.subject) {
			return nil, t.unsupported(c.node, "expected %s in the comparison", c.subject)
		}

		var at Time
		resolved := t.resolve(other)
		switch resolved.kind {
		case ednKeyword:
			input, ok := utils.ParseQueryDateInput(resolved.text, time.Now())
			if !ok {
				return nil, t.unsupported(other, "expected a time, such as :-7d-ms or :right-now-ms")
			}
			if !input.HasTime {
				return nil, t.unsupported(other, "times are only compared with inputs that have a time of day, such as :-7d-ms or :today-start, as Logseq stores them in milliseconds")
			}
			at = TimeInput(resolved.text)
		case ednNumber:
			ms, err := strconv.ParseInt(resolved.text, 10, 64)
			if err != nil {
				return nil, t.unsupported(other, "expected a time in milliseconds")
			}
			at = At(time.UnixMilli(ms))
		default:
			return nil, t.unsupported(other, "expected a time, such as :-7d-ms or :right-now-ms")
		}

		filters = append(filters, &TimestampFilter{Field: src.timestamp, Op: op, Time: at, Page: isPage})
	}

	return andOf(filters), nil
}

// asRange combines a lower and an upper bound into one range, such as
// `> today` and `<= +3d` into `+1d +3d`.
func asRange(a *DateFilter, b *DateFilter) (*DateFilter, bool) {
	lower, upper := a, b
	if lower.Op == Less || lower.Op == LessOrEqual {
		lower, upper = b, a
	}

	start, ok := inclusiveDate(lower, Greater, GreaterOrEqual, 1)
	if !ok {
		return nil, false
	}

	end, ok := inclusiveDate(upper, Less, LessOrEqual, -1)
	if !ok {
		return nil, false
	}

	return &DateFilter{Field: a.Field, Op: Range, Date: start, End: end}, true
}

// inclusiveDate returns the date of a bound as an included date. A bound that
// leaves out its date, such as `> today`, is moved by a day.
func inclusiveDate(bound *DateFilter, exclusive Op, inclusive Op, shift int) (Date, bool) {
	switch bound.Op {
	case inclusive:
		return bound.Date, true
	case exclusive:
		return shiftDate(bound.Date, shift)
	}

	return Date{}, false
}

var dayOffset = regexp.MustCompile(`^([+-]\d+)d$`)

// shiftDate moves a date by a number of days, for dates that stay readable.
func shiftDate(date Date, days int) (Date, bool) {
	switch {
	case date.Relative != "":
		offset := 0
		switch date.Relative {
		case "today":
		case "yesterday":
			offset = -1
		case "tomorrow":
			offset = 1
		default:
			m := dayOffset.FindStringSubmatch(date.Relative)
			if m == nil {
				return Date{}, false
			}
			offset, _ = strconv.Atoi(m[1])
		}
		return DaysFromToday(offset + days), true
	case !date.Fixed.IsZero():
		return On(date.Fixed.AddDate(0, 0, days)), true
	}

	return Date{}, false
}

// resultTransform translates a :result-transform that sorts the results, such
// as `(fn [result] (sort-by (fn [b] (get b :block/priority "Z")) result))`,
// or that keeps some of them, such as
// `(fn [result] (filter (fn [b] (= "TODO" (:block/marker b))) result))` or
// `(fn [result] (take 10 result))`. findsBlocks tells if the results are
// blocks or pages.
func (t *translator) resultTransform(node *ednNode, findsBlocks bool) (*transform, error) {
	if node.head() != "fn" || len(node.items) != 3 || node.items[1].kind != ednVector ||
		len(node.items[1].items) != 1 || node.items[1].items[0].kind != ednSymbol {
		return nil, t.unsupported(node, "only a :result-transform that sorts with sort-by, keeps results with filter or remove, or takes the first results with take is supported")
	}

	transform := &transform{t: t, results: node.items[1].items[0].text, findsBlocks: findsBlocks}
	if err := transform.body(node.items[2]); err != nil {
		return nil, err
	}
	return transform, nil
}

// transform holds what a :result-transform does to the results.
type transform struct {
	t *translator

	// results is the name of the parameter of the function, which is the
	// results of the query.
	results string

	// findsBlocks tells if the results are blocks or pages.
	findsBlocks bool

	sorts   []Sort
	filters []Filter

	// limit is the number of results that take keeps, or zero for all of
	// them.
	limit int
}

// checkAfterTake returns an error if take came before the function in body.
// The limit is applied after the sort and the filter, so take must be the
// last step.
func (tr *transform) checkAfterTake(body *ednNode) error {
	if tr.limit > 0 {
		return tr.t.unsupported(body, "take is only supported as the last step of a :result-transform")
	}
	return nil
}

// body reads the body of the function, which is the results or the results
// with functions applied to them.
func (tr *transform) body(body *ednNode) error {
	if body.isSymbol(tr.results) {
		return nil
	}

	switch body.head() {
	case "->>":
		if len(body.items) < 2 {
			break
		}
		return tr.body(threadLast(body))
	case "reverse":
		if len(body.items) != 2 {
			break
		}

		before := len(tr.sorts)
		if err := tr.body(body.items[1]); err != nil {
			return err
		}
		if err := tr.checkAfterTake(body); err != nil {
			return err
		}
		if len(tr.sorts) == before {
			return tr.t.unsupported(body, "reverse is only supported for the results of sort-by")
		}

		// sort-by puts its sort first, so the sorts inside reverse are the
		// first ones.
		for i := 0; i < len(tr.sorts)-before; i++ {
			if tr.sorts[i].Order == Asc {
				tr.sorts[i].Order = Desc
			} else {
				tr.sorts[i].Order = Asc
			}
		}
		return nil
	case "sort-by":
		if len(body.items) != 3 {
			break
		}

		if err := tr.body(body.items[2]); err != nil {
			return err
		}
		if err := tr.checkAfterTake(body); err != nil {
			return err
		}

		sort, err := tr.t.sortKey(body.items[1])
		if err != nil {
			return err
		}
		// The last sort decides the order, so it comes first.
		tr.sorts = append([]Sort{sort}, tr.sorts...)
		return nil
	case "filter", "remove":
		if len(body.items) != 3 {
			break
		}

		if err := tr.body(body.items[2]); err != nil {
			return err
		}
		if err := tr.checkAfterTake(body); err != nil {
			return err
		}

		filter, err := tr.predicate(body.items[1])
		if err != nil {
			return err
		}
		if body.head() == "remove" {
			filter = Not(filter)
		}
		tr.filters = append(tr.filters, filter)
		return nil
	case "take":
		if len(body.items) != 3 {
			break
		}

		if err := tr.body(body.items[2]); err != nil {
			return err
		}

		n, err := strconv.Atoi(body.items[1].text)
		if body.items[1].kind != ednNumber || err != nil || n <= 0 {
			return tr.t.unsupported(body.items[1], "expected a number above zero")
		}
		if tr.limit == 0 || n < tr.limit {
			tr.limit = n
		}
		return nil
	}

	return tr.t.unsupported(body, "only a :result-transform that sorts with sort-by, keeps results with filter or remove, or takes the first results with take is supported")
}

// threadLast rewrites `(->> r (filter f) (sort-by k))` as
// `(sort-by k (filter f r))`.
func threadLast(node *ednNode) *ednNode {
	value := node.items[1]
	for _, step := range node.items[2:] {
		next := &ednNode{kind: ednList, pos: step.pos, end: step.end}
		if step.kind == ednList {
			next.items = append(append(next.items, step.items...), value)
		} else {
			next.items = []*ednNode{step, value}
		}
		value = next
	}
	return value
}

// predicate translates the function of filter or remove, such as
// `(fn [b] (= "TODO" (:block/marker b)))`, `#(get-in % [:block/properties :type])`
// or `:block/marker`, into the filter for the results it returns true for.
func (tr *transform) predicate(node *ednNode) (Filter, error) {
	switch {
	case node.kind == ednKeyword:
		return tr.test(&ednNode{kind: ednList, items: []*ednNode{node, {kind: ednSymbol, text: "%"}}, pos: node.pos, end: node.end}, "%")
	case node.head() == "fn" && len(node.items) == 3 && node.items[1].kind == ednVector &&
		len(node.items[1].items) == 1 && node.items[1].items[0].kind == ednSymbol:
		return tr.test(node.items[2], node.items[1].items[0].text)
	case node.kind == ednTagged && node.text == "fn" && len(node.items) == 1:
		return tr.test(node.items[0], "%")
	}

	return nil, tr.t.unsupported(node, "expected a function such as (fn [b] (= \"TODO\" (:block/marker b)))")
}

// test translates the body of a predicate, where param is the result. The
// values of the result, such as `(:block/marker b)`, are given to the
// translator as clauses on a query for one result, so that the same
// conditions are supported as in :where. `and`, `or`, `not`, `not=` and
// `nil?` are translated here, as a value that is not set makes a condition
// false in :where but `not=` true in Clojure.
func (tr *transform) test(expr *ednNode, param string) (Filter, error) {
	switch expr.head() {
	case "and", "or":
		var filters []Filter
		for _, item := range expr.items[1:] {
			filter, err := tr.test(item, param)
			if err != nil {
				return nil, err
			}
			filters = append(filters, filter)
		}

		if expr.head() == "or" {
			return Or(filters...), nil
		}
		return And(filters...), nil
	case "not":
		if len(expr.items) != 2 {
			break
		}

		filter, err := tr.test(expr.items[1], param)
		if err != nil {
			return nil, err
		}
		return Not(filter), nil
	case "not=":
		equal := &ednNode{kind: ednList, pos: expr.pos, end: expr.end,
			items: append([]*ednNode{{kind: ednSymbol, text: "=", pos: expr.items[0].pos, end: expr.items[0].end}}, expr.items[1:]...)}
		filter, err := tr.test(equal, param)
		if err != nil {
			return nil, err
		}
		return Not(filter), nil
	case "nil?", "some?":
		if len(expr.items) != 2 {
			break
		}

		filter, err := tr.value(expr.items[1], param, nil, expr)
		if err != nil {
			return nil, err
		}
		if expr.head() == "nil?" {
			return Not(filter), nil
		}
		return filter, nil
	}

	if path, _ := keyPath(expr, param); len(path) > 0 {
		// A value on its own is true when it is set.
		return tr.value(expr, param, nil, expr)
	}

	if expr.kind != ednList || expr.head() == "" {
		return nil, tr.t.unsupported(expr, "expected a condition on a value of the result, such as (= \"TODO\" (:block/marker %s))", param)
	}

	var access *ednNode
	for _, arg := range expr.items[1:] {
		if path, _ := keyPath(arg, param); len(path) == 0 {
			continue
		}
		if access != nil {
			return nil, tr.t.unsupported(expr, "comparing two values of the result is not supported")
		}
		access = arg
	}
	if access == nil {
		return nil, tr.t.unsupported(expr, "expected a condition on a value of the result, such as (= \"TODO\" (:block/marker %s))", param)
	}

	return tr.value(access, param, expr, expr)
}

// value translates a condition on one value of the result, such as
// `(:block/marker b)` in `(= "TODO" (:block/marker b))`. Without a condition
// it matches the results that have the value, or for an attribute that is
// true or false, such as :block/journal?, the results where it is true.
func (tr *transform) value(access *ednNode, param string, condition *ednNode, node *ednNode) (Filter, error) {
	path, hasDefault := keyPath(access, param)
	if len(path) == 0 {
		return nil, tr.t.unsupported(access, "expected a value of the result, such as (:block/marker %s)", param)
	}
	if hasDefault {
		return nil, tr.t.unsupported(access, "default values are not supported in the function of filter or remove")
	}

	at := func(kind ednKind, text string, items ...*ednNode) *ednNode {
		return &ednNode{kind: kind, text: text, items: items, pos: node.pos, end: node.end}
	}
	variable := func(name string) *ednNode { return at(ednSymbol, name) }

	result := variable("?_result")
	value := variable("?_value")
	clauses := []*ednNode{at(ednKeyword, "find"), result, at(ednKeyword, "where")}
	if !tr.findsBlocks {
		clauses = append(clauses, at(ednVector, "", result, at(ednKeyword, "block/name")))
	}

	entity := result
	for i, key := range path {
		if key.kind != ednKeyword {
			return nil, tr.t.unsupported(access, "expected keywords in the path of the value")
		}

		last := i == len(path)-1
		if (key.text == "block/properties" || key.text == "page/properties") && i == len(path)-2 {
			props := variable(fmt.Sprintf("?_props%d", i))
			get := at(ednList, "", variable("get"), props, path[i+1])
			clauses = append(clauses, at(ednVector, "", entity, key, props), at(ednVector, "", get, value))
			break
		}

		next := value
		switch {
		case !last:
			next = variable(fmt.Sprintf("?_entity%d", i))
		case condition == nil && strings.HasSuffix(key.text, "?"):
			next = variable("true")
		}
		clauses = append(clauses, at(ednVector, "", entity, key, next))
		entity = next
	}

	if condition != nil {
		call := at(ednList, "")
		for _, item := range condition.items {
			if item == access {
				item = value
			}
			call.items = append(call.items, item)
		}
		clauses = append(clauses, at(ednVector, "", call))
	}

	q, err := tr.t.sub().translate(at(ednVector, "", clauses...), nil)
	if err != nil {
		return nil, err
	}
	if q.Filter == nil {
		// The condition is true for every result.
		return And(), nil
	}
	return q.Filter, nil
}

// sub returns a translator for another query in the same text, with the
// same options and rules but none of the inputs.
func (t *translator) sub() *translator {
	return &translator{
		src:         t.src,
		inputs:      make(map[string]*ednNode),
		pages:       t.pages,
		blocks:      t.blocks,
		blockInputs: make(map[string]blockRef),
		rules:       t.rules,
		inlineRules: t.inlineRules,
	}
}

// withFilter adds a filter that the results must match as well.
func withFilter(filter Filter, other Filter) Filter {
	switch {
	case other == nil:
		return filter
	case filter == nil:
		return other
	}

	var filters []Filter
	for _, f := range []Filter{filter, other} {
		if and, ok := f.(*AndFilter); ok {
			filters = append(filters, and.Filters...)
		} else {
			filters = append(filters, f)
		}
	}
	return And(filters...)
}

// sortKey reads the key that sort-by sorts on: a keyword such as
// :block/priority, or a function that gets it from a result.
func (t *translator) sortKey(node *ednNode) (Sort, error) {
	var path []*ednNode
	switch {
	case node.kind == ednKeyword:
		path = []*ednNode{node}
	case node.head() == "fn" && len(node.items) == 3 && node.items[1].kind == ednVector && len(node.items[1].items) == 1:
		path, _ = keyPath(node.items[2], node.items[1].items[0].text)
	case node.kind == ednTagged && node.text == "fn" && len(node.items) == 1:
		path, _ = keyPath(node.items[0], "%")
	}

	if sort, ok := sortForPath(path); ok {
		return sort, nil
	}

	return Sort{}, t.unsupported(node, "only sorting by priority, scheduled, deadline, journal day, page title, the time of the last change or a property is supported")
}

// keyPath returns the keys that an expression such as `(get b :k "default")`,
// `(:k b)`, `(get-in b [:a :b])` or `(:b (:a b))` looks up in the result
// named param, and true if a lookup has a default value. The path is empty
// for the result itself, and nil for other expressions.
func keyPath(expr *ednNode, param string) ([]*ednNode, bool) {
	if expr.isSymbol(param) {
		return []*ednNode{}, false
	}
	if expr.kind != ednList || len(expr.items) < 2 {
		return nil, false
	}

	first := expr.items[0]
	var from *ednNode
	var keys []*ednNode
	var withDefault bool
	switch {
	case first.isSymbol("get") && (len(expr.items) == 3 || len(expr.items) == 4) && expr.items[2].kind == ednKeyword:
		from, keys, withDefault = expr.items[1], expr.items[2:3], len(expr.items) == 4
	case first.isSymbol("get-in") && (len(expr.items) == 3 || len(expr.items) == 4) && expr.items[2].kind == ednVector:
		from, keys, withDefault = expr.items[1], expr.items[2].items, len(expr.items) == 4
	case first.kind == ednKeyword && (len(expr.items) == 2 || len(expr.items) == 3):
		from, keys, withDefault = expr.items[1], expr.items[:1], len(expr.items) == 3
	default:
		return nil, false
	}

	base, baseDefault := keyPath(from, param)
	if base == nil {
		return nil, false
	}
	return append(base, keys...), withDefault || baseDefault
}

func sortForPath(path []*ednNode) (Sort, bool) {
	for _, key := range path {
		if key.kind != ednKeyword {
			return Sort{}, false
		}
	}

	switch len(path) {
	case 1:
		switch path[0].text {
		case "block/priority":
			return Sort{Field: SortPriority}, true
		case "block/scheduled":
			return Sort{Field: SortScheduled}, true
		case "block/deadline":
			return Sort{Field: SortDeadline}, true
		case "block/updated-at":
			return Sort{Field: SortUpdatedAt}, true
		}
	case 2:
		switch {
		case path[0].text == "block/page" && path[1].text == "block/journal-day":
			return Sort{Field: SortJournalDay}, true
		case path[0].text == "block/page" && (path[1].text == "block/original-name" || path[1].text == "block/name"):
			return Sort{Field: SortTitle}, true
		case path[0].text == "block/properties":
			return Sort{Field: SortProperty, Property: path[1].text}, true
		}
	}

	return Sort{}, false
}
