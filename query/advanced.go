package query

import (
	"errors"
	"fmt"
	"regexp"
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
}

// ParseOption is an option for ParseAdvanced and ParseNode.
type ParseOption func(*parseOptions)

type parseOptions struct {
	currentPage  string
	queryPage    string
	currentBlock string
	parentBlock  string
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
func WithCurrentBlock(id string) ParseOption {
	return func(o *parseOptions) {
		o.currentBlock = id
	}
}

// WithParentBlock gives the UUID of the block that the input `:parent-block`
// of an advanced query stands for, which is the parent of the block the query
// is written in. The UUID is the value of the `id::` property of the parent,
// the same as for WithCurrentBlock.
func WithParentBlock(id string) ParseOption {
	return func(o *parseOptions) {
		o.parentBlock = id
	}
}

var inputOptionNames = map[string]string{
	"current-page":  "WithCurrentPage",
	"query-page":    "WithQueryPage",
	"current-block": "WithCurrentBlock",
	"parent-block":  "WithParentBlock",
}

// ParseNode reads the query in a node of a page: a simple query written as
// `{{query ...}}`, which is a *content.Query, or an advanced query written
// between `#+BEGIN_QUERY` and `#+END_QUERY`, which is a *content.QueryCommand.
// Nodes such as these are found with content.IsQuery, and the block that
// holds one with content.BlockOf.
//
// A simple query is read with Parse, and returned without a title. An
// advanced query is read with ParseAdvanced, which the options are given to.
// Run the query with Graph.Query in the logseq package.
func ParseNode(node content.Node, opts ...ParseOption) (*AdvancedQuery, error) {
	switch n := node.(type) {
	case *content.Query:
		q, err := Parse(n.Query)
		if err != nil {
			return nil, err
		}
		return &AdvancedQuery{Query: q}, nil
	case *content.QueryCommand:
		return ParseAdvanced(n.Query, opts...)
	}

	return nil, fmt.Errorf("query: %T is not a query", node)
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
//   - :find with one variable or `(pull ?b [*])`, for blocks or pages
//   - :in and :inputs with date inputs such as `:today` or `:-7d`, page names
//     and numbers
//   - the attributes :block/marker, :block/priority, :block/scheduled,
//     :block/deadline, :block/content, :block/page, :block/parent,
//     :block/refs, :block/path-refs, :block/properties, :block/name, :block/original-name,
//     :block/journal-day, :block/journal?, :block/tags, :block/namespace and
//     :page/properties
//   - the conditions =, not=, <, <=, >, >=, contains?, get and
//     clojure.string/includes?, and `[(get ?props :key) ?value]`
//   - the title of a page with a known title as a value, such as ?title in
//     `[?p :block/name ?page] [?p :block/original-name ?title]`
//   - and, or, or-join, not and not-join
//   - the rules of simple queries, such as `(task ?b #{"TODO"})`
//   - a :query that is a simple query, such as `(and (task TODO) [[page]])`
//   - a :result-transform that sorts with sort-by, and reverse
//
// The inputs `:current-page` and `:query-page` depend on where the query is,
// and are supported when the page is given with WithCurrentPage and
// WithQueryPage. The inputs `:current-block` and `:parent-block` are supported
// when the UUID of the block is given with WithCurrentBlock and
// WithParentBlock, as the parent of the blocks in `[?b :block/parent ?block]`.
// :rules are not supported. Keys that only change how Logseq shows the
// results, such as :view and :table-view?, are ignored. Errors in the EDN itself are
// returned as *ParseError.
func ParseAdvanced(text string, opts ...ParseOption) (*AdvancedQuery, error) {
	options := &parseOptions{}
	for _, opt := range opts {
		opt(options)
	}

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
		blocks: map[string]string{
			"current-block": options.currentBlock,
			"parent-block":  options.parentBlock,
		},
		blockInputs: make(map[string]string),
	}

	result := &AdvancedQuery{}
	var queryNode, inputsNode, transformNode *ednNode
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
		case "query":
			queryNode = value
		case "inputs":
			inputsNode = value
		case "result-transform":
			transformNode = value
		case "rules":
			return nil, t.unsupported(value, "custom rules are not supported")
		}
	}

	if queryNode == nil {
		return nil, errorAt(root.pos, "the query has no :query")
	}

	var q *Query
	switch queryNode.kind {
	case ednString:
		q, err = Parse(queryNode.text)
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
		sorts, err := t.resultTransform(transformNode)
		if err != nil {
			return nil, err
		}
		q.Sort = append(q.Sort, sorts...)
	}

	result.Query = q
	return result, nil
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
)

// source is where a variable gets its value from, such as the marker of a
// block or a property of a page.
type source struct {
	kind sourceKind

	// entity is the variable of the block or page the value belongs to.
	entity string

	// key is the name of the property for sourceProperty.
	key string
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

	// blocks are the UUIDs of the blocks that the inputs :current-block and
	// :parent-block stand for, empty if they are not known.
	blocks map[string]string

	// blockInputs maps the variables in :in that are blocks to their UUIDs.
	// They are kept out of inputs, as they can not be read as values.
	blockInputs map[string]string

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

	where := sections["where"]

	t.kinds = make(map[string]entityKind)
	t.pageOf = make(map[string]string)
	t.names = make(map[string]string)
	t.titles = make(map[string]*ednNode)
	t.props = make(map[string]string)
	t.sources = make(map[string]*source)
	for _, clause := range where {
		if err := t.collect(clause); err != nil {
			return nil, err
		}
	}

	t.readTitles()

	if err := t.chooseEntities(find[0]); err != nil {
		return nil, err
	}

	filter, err := t.conjunction(where, nil)
	if err != nil {
		return nil, err
	}

	q := New(filter)
	if q.FindsBlocks() != (t.kinds[t.result] == entityBlock) {
		if q.FindsBlocks() {
			return nil, t.unsupported(find[0], "finding pages by the blocks on them is not supported")
		}
		return nil, t.unsupported(find[0], "finding blocks with only conditions on their page is not supported")
	}

	return q, nil
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

		if id, isBlock := t.blocks[input.text]; isBlock && input.kind == ednKeyword {
			if id == "" {
				return t.unsupported(input, "the input :%s needs the UUID of the block, given with %s", input.text, inputOptionNames[input.text])
			}

			t.blockInputs[v.text] = id
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

	if _, isBlock := t.blockInputs[node.text]; isBlock && isVar(node) {
		return t.unsupported(node, "a block from :in is only supported as the parent in [?b :block/parent %s]", node.text)
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
func (t *translator) collect(clause *ednNode) error {
	switch clause.kind {
	case ednList:
		head := clause.head()
		switch head {
		case "and", "or", "not":
			for _, item := range clause.items[1:] {
				if err := t.collect(item); err != nil {
					return err
				}
			}
		case "or-join", "not-join":
			if len(clause.items) > 1 {
				for _, item := range clause.items[2:] {
					if err := t.collect(item); err != nil {
						return err
					}
				}
			}
		default:
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
		case "block/page", "block/refs", "block/path-refs", "block/tags", "block/namespace":
			if isVar(value) {
				if err := t.setKind(value, entityPage); err != nil {
					return err
				}
			}
		}

		if attribute == "block/page" && isVar(value) {
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

// blockID returns the UUID of the block that a value of :block/parent points
// at: a variable for :current-block or :parent-block, or a lookup such as
// `[:block/uuid #uuid "..."]`.
func (t *translator) blockID(node *ednNode) (string, bool) {
	if isVar(node) {
		id, ok := t.blockInputs[node.text]
		return id, ok
	}

	if node.kind != ednVector || len(node.items) != 2 || !(node.items[0].kind == ednKeyword && node.items[0].text == "block/uuid") {
		return "", false
	}

	value := node.items[1]
	if value.kind == ednTagged && value.text == "uuid" && len(value.items) == 1 {
		value = value.items[0]
	}

	if value.kind != ednString || value.text == "" {
		return "", false
	}
	return value.text, true
}

// conjunction translates clauses that must all match. The conditions in
// pending come from an outer scope and are applied where their variables get
// their values.
func (t *translator) conjunction(clauses []*ednNode, pending []*condition) (Filter, error) {
	var local []*condition
	var bindings, lists []*ednNode

	for _, clause := range clauses {
		switch {
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

	var filters []Filter
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

	// Conditions on variables that got their values in an outer scope, such
	// as in `(not [(= ?m "DONE")])`, apply to that value here.
	for _, c := range local {
		if c.used {
			continue
		}

		src, ok := t.sources[c.subject]
		if !ok {
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
			// This only names the page of the block. A title for it is
			// translated with its :block/name.
			return nil, nil
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
			// The clause only gives the title a variable.
			if isVar(value) && len(take(conditions, value.text)) > 0 {
				return nil, t.unsupported(clause, "conditions on the title of a page are not supported")
			}
			return nil, nil
		}

		if entity.text == t.result {
			return nil, t.unsupported(clause, "finding pages by their title is not supported")
		}

		if entity.text == t.page {
			return Page(name), nil
		}

		// The variable is a page that another clause refers to, such as a
		// page in :block/path-refs.
		return nil, nil
	case "block/journal?":
		if err := t.checkPageOfBlock(clause, entity); err != nil {
			return nil, err
		}

		if !value.isSymbol("true") {
			return nil, t.unsupported(clause, "expected true")
		}
		return OnJournal(), nil
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

		id, ok := t.blockID(value)
		if !ok {
			return nil, t.unsupported(clause, "expected a block, such as :current-block or [:block/uuid #uuid \"...\"]")
		}
		return Parent(id), nil
	case "block/refs", "block/path-refs":
		if err := t.checkBlock(clause, entity); err != nil {
			return nil, err
		}

		name, ok := t.pageRefName(value)
		if !ok {
			return nil, t.unsupported(clause, "expected a page, such as [:block/name \"page\"]")
		}

		if attribute == "block/refs" {
			return Refs(name), nil
		}
		return PageRef(name), nil
	case "block/tags", "block/namespace":
		if entity.text != t.page {
			return nil, t.unsupported(clause, "%s of other pages is not supported", attribute)
		}

		name, ok := t.pageRefName(value)
		if !ok {
			return nil, t.unsupported(clause, "expected a page, such as [:block/name \"page\"]")
		}

		if attribute == "block/tags" {
			return PageTags(name), nil
		}
		return Namespace(name), nil
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
	if entity.text != t.page || t.result != t.block {
		return t.unsupported(clause, "only the journal of the blocks the query finds is supported")
	}

	return nil
}

// functionBinding translates `[(get ?props :type) ?t]`, which gives ?t the
// value of a property.
func (t *translator) functionBinding(clause *ednNode, conditions []*condition) (Filter, error) {
	call := clause.items[0]
	out := clause.items[1]

	var entity, key string
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
	case call.head() == "get-in" && len(call.items) >= 3 && isVar(call.items[1]):
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
	default:
		return nil, t.unsupported(clause, "only getting a property with get is supported")
	}

	if entity != t.block && entity != t.page {
		return nil, t.unsupported(clause, "properties of other blocks or pages are not supported")
	}

	if entity == t.block && t.result != t.block {
		return nil, t.unsupported(clause, "finding pages by the blocks on them is not supported")
	}

	return t.bind(out, &source{kind: sourceProperty, entity: entity, key: key}, conditions)
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

	filter, err := f()
	t.sources = saved
	return filter, err
}

// rule translates a rule from simple queries, such as `(task ?b #{"TODO"})`.
func (t *translator) rule(clause *ednNode) (Filter, error) {
	head := clause.head()
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
	case sourceContent:
		var filters []Filter
		for _, c := range conditions {
			if c.fn != "clojure.string/includes?" || len(c.args) != 2 || !c.args[0].isSymbol(c.subject) {
				return nil, t.unsupported(c.node, "only clojure.string/includes? is supported for the content")
			}

			text, ok := t.scalar(c.args[1])
			if !ok {
				return nil, t.unsupported(c.node, "expected the text to search for")
			}
			filters = append(filters, FullText(text))
		}
		return andOf(filters), nil
	case sourceProperty:
		isPage := src.entity == t.page

		var filters []Filter
		for _, c := range conditions {
			value, err := t.comparedValue(c, "get", "contains?", "=")
			if err != nil {
				return nil, err
			}

			if isPage {
				filters = append(filters, PageProperty(src.key, value))
			} else {
				filters = append(filters, Property(src.key, value))
			}
		}

		if len(filters) == 0 {
			if isPage {
				return HasPageProperty(src.key), nil
			}
			return HasProperty(src.key), nil
		}
		return andOf(filters), nil
	}

	return nil, fmt.Errorf("query: unknown source %d", src.kind)
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
// as `(fn [result] (sort-by (fn [b] (get b :block/priority "Z")) result))`.
func (t *translator) resultTransform(node *ednNode) ([]Sort, error) {
	if node.head() != "fn" || len(node.items) != 3 || node.items[1].kind != ednVector ||
		len(node.items[1].items) != 1 || node.items[1].items[0].kind != ednSymbol {
		return nil, t.unsupported(node, "only a :result-transform that sorts with sort-by is supported")
	}

	return t.sortBody(node.items[2], node.items[1].items[0].text)
}

func (t *translator) sortBody(body *ednNode, results string) ([]Sort, error) {
	switch body.head() {
	case "reverse":
		if len(body.items) != 2 {
			break
		}

		sorts, err := t.sortBody(body.items[1], results)
		if err != nil {
			return nil, err
		}

		for i := range sorts {
			if sorts[i].Order == Asc {
				sorts[i].Order = Desc
			} else {
				sorts[i].Order = Asc
			}
		}
		return sorts, nil
	case "sort-by":
		if len(body.items) != 3 || !body.items[2].isSymbol(results) {
			break
		}

		sort, err := t.sortKey(body.items[1])
		if err != nil {
			return nil, err
		}
		return []Sort{sort}, nil
	}

	return nil, t.unsupported(body, "only a :result-transform that sorts with sort-by is supported")
}

// sortKey reads the key that sort-by sorts on: a keyword such as
// :block/priority, or a function that gets it from a result.
func (t *translator) sortKey(node *ednNode) (Sort, error) {
	var path []*ednNode
	switch {
	case node.kind == ednKeyword:
		path = []*ednNode{node}
	case node.head() == "fn" && len(node.items) == 3 && node.items[1].kind == ednVector && len(node.items[1].items) == 1:
		path = keyPath(node.items[2], node.items[1].items[0].text)
	case node.kind == ednTagged && node.text == "fn" && len(node.items) == 1:
		path = keyPath(node.items[0], "%")
	}

	if sort, ok := sortForPath(path); ok {
		return sort, nil
	}

	return Sort{}, t.unsupported(node, "only sorting by priority, scheduled, deadline, journal day, page title or a property is supported")
}

// keyPath returns the keys that an expression such as `(get b :k "default")`,
// `(:k b)` or `(get-in b [:a :b])` looks up in the result named param.
func keyPath(expr *ednNode, param string) []*ednNode {
	if expr.kind != ednList || len(expr.items) < 2 {
		return nil
	}

	first := expr.items[0]
	switch {
	case first.isSymbol("get") && len(expr.items) >= 3 && expr.items[1].isSymbol(param) && expr.items[2].kind == ednKeyword:
		return []*ednNode{expr.items[2]}
	case first.isSymbol("get-in") && len(expr.items) >= 3 && expr.items[1].isSymbol(param) && expr.items[2].kind == ednVector:
		return expr.items[2].items
	case first.kind == ednKeyword && expr.items[1].isSymbol(param):
		return []*ednNode{first}
	}

	return nil
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
