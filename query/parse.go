package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/utils"
)

// ParseError is an error in the text of a query.
type ParseError struct {
	// Offset is the byte offset in the text where the error is.
	Offset int

	// Message describes the error.
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("query: %s at offset %d", e.Message, e.Offset)
}

func errorAt(offset int, format string, args ...any) *ParseError {
	return &ParseError{
		Offset:  offset,
		Message: fmt.Sprintf(format, args...),
	}
}

// Parse reads a query in the text form described in the package
// documentation, which is the part inside `{{query ...}}`. Empty text is a
// query that matches everything. Errors are returned as *ParseError.
func Parse(text string) (*Query, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return &Query{}, nil
	}

	switch trimmed[0] {
	case '(', '[', '"', '#':
	default:
		// Logseq searches for text that is not a query.
		return New(FullText(trimmed)), nil
	}

	r := &reader{input: text}
	e, err := r.read()
	if err != nil {
		return nil, err
	}

	r.skipSpace()
	if r.pos < len(r.input) {
		return nil, errorAt(r.pos, "unexpected text after the query, combine filters with `and` or `or`")
	}

	q := &Query{}

	if e.isList("sort-by") || e.isList("sample") {
		if err := q.readOption(e); err != nil {
			return nil, err
		}

		return q, nil
	}

	if !e.isList("and") {
		q.Filter, err = parseFilter(e)
		if err != nil {
			return nil, err
		}

		return q, nil
	}

	// The sort order and sample are read from the top-level `and`, which is
	// where Logseq reads them.
	filters := make([]Filter, 0, len(e.items)-1)
	hasOptions := false
	for _, item := range e.items[1:] {
		if item.isList("sort-by") || item.isList("sample") {
			if err := q.readOption(item); err != nil {
				return nil, err
			}

			hasOptions = true
			continue
		}

		filter, err := parseFilter(item)
		if err != nil {
			return nil, err
		}

		filters = append(filters, filter)
	}

	if len(filters) > 0 || !hasOptions {
		q.Filter = &AndFilter{Filters: filters}
	}

	return q, nil
}

func (q *Query) readOption(e *expr) error {
	args := e.items[1:]

	switch e.op() {
	case "sort-by":
		if len(args) == 0 || len(args) > 2 {
			return errorAt(e.pos, "sort-by takes a field and an optional order")
		}

		name, err := parseKey(args[0])
		if err != nil {
			return err
		}

		// Logseq sorts in descending order if no order is given.
		order := Desc
		if len(args) == 2 {
			if args[1].kind != exprSymbol {
				return errorAt(args[1].pos, "expected `asc` or `desc`")
			}

			switch strings.ToLower(strings.TrimPrefix(args[1].text, ":")) {
			case "asc":
				order = Asc
			case "desc":
				order = Desc
			default:
				return errorAt(args[1].pos, "expected `asc` or `desc`")
			}
		}

		for field, fieldName := range sortFieldNames {
			if strings.EqualFold(name, fieldName) {
				q.SortBy(field, order)
				return nil
			}
		}

		q.SortByProperty(name, order)
		return nil
	case "sample":
		if len(args) != 1 || args[0].kind != exprSymbol {
			return errorAt(e.pos, "sample takes a number")
		}

		n, err := strconv.Atoi(args[0].text)
		if err != nil || n <= 0 {
			return errorAt(args[0].pos, "sample takes a number above zero")
		}

		q.Sample = n
		return nil
	}

	return nil
}

func parseFilter(e *expr) (Filter, error) {
	switch e.kind {
	case exprPageRef:
		return PageRef(e.text), nil
	case exprString:
		return FullText(e.text), nil
	case exprList:
	default:
		return nil, errorAt(e.pos, "expected a filter, such as `(task TODO)`, `[[page]]` or \"text\"")
	}

	if len(e.items) == 0 || e.items[0].kind != exprSymbol {
		return nil, errorAt(e.pos, "expected the name of a filter after `(`")
	}

	op := e.op()
	args := e.items[1:]

	switch op {
	case "and", "or":
		filters := make([]Filter, 0, len(args))
		for _, arg := range args {
			filter, err := parseFilter(arg)
			if err != nil {
				return nil, err
			}
			filters = append(filters, filter)
		}

		if op == "and" {
			return &AndFilter{Filters: filters}, nil
		}
		return &OrFilter{Filters: filters}, nil
	case "not":
		if len(args) == 0 {
			return nil, errorAt(e.pos, "not takes one or more filters")
		}

		filters := make([]Filter, 0, len(args))
		for _, arg := range args {
			filter, err := parseFilter(arg)
			if err != nil {
				return nil, err
			}
			filters = append(filters, filter)
		}

		// Logseq reads `(not a b)` as a Datalog `not`, which matches when the
		// filters do not all match.
		if len(filters) == 1 {
			return Not(filters[0]), nil
		}
		return Not(And(filters...)), nil
	case "task", "todo":
		names, err := parseNames(args, e.pos, op)
		if err != nil {
			return nil, err
		}

		statuses := make([]content.TaskStatus, 0, len(names))
		for i, name := range names {
			status, ok := taskStatuses[strings.ToUpper(name)]
			if !ok {
				return nil, errorAt(namePos(args, i, e.pos), "unknown task status %q", name)
			}
			statuses = append(statuses, status)
		}

		return Task(statuses...), nil
	case "priority":
		names, err := parseNames(args, e.pos, op)
		if err != nil {
			return nil, err
		}

		priorities := make([]content.Priority, 0, len(names))
		for i, name := range names {
			priority, ok := priorityValues[strings.ToUpper(name)]
			if !ok {
				return nil, errorAt(namePos(args, i, e.pos), "unknown priority %q, expected A, B or C", name)
			}
			priorities = append(priorities, priority)
		}

		return Priority(priorities...), nil
	case "has-child", "has-descendant":
		if len(args) != 1 {
			return nil, errorAt(e.pos, "%s takes one filter", op)
		}

		filter, err := parseFilter(args[0])
		if err != nil {
			return nil, err
		}

		if op == "has-descendant" {
			return HasDescendant(filter), nil
		}
		return HasChild(filter), nil
	case "page":
		if len(args) == 1 && args[0].kind == exprList {
			pages, err := parseFilter(args[0])
			if err != nil {
				return nil, err
			}
			return OnPage(pages), nil
		}

		name, err := parseSingleName(e, args)
		if err != nil {
			return nil, err
		}
		return Page(name), nil
	case "page-ref", "refs":
		if len(args) == 1 && args[0].kind == exprList {
			pages, err := parseFilter(args[0])
			if err != nil {
				return nil, err
			}

			if op == "refs" {
				return RefsTo(pages), nil
			}
			return PageRefTo(pages), nil
		}

		name, err := parseSingleName(e, args)
		if err != nil {
			return nil, err
		}

		if op == "refs" {
			return Refs(name), nil
		}
		return PageRef(name), nil
	case "parent", "ancestor":
		if len(args) == 1 && args[0].kind == exprList {
			filter, err := parseFilter(args[0])
			if err != nil {
				return nil, err
			}

			if op == "ancestor" {
				return DescendantOf(filter), nil
			}
			return ChildOf(filter), nil
		}

		if len(args) > 1 && args[0].kind == exprPageRef {
			location, err := parseLocation(args[1:])
			if err != nil {
				return nil, err
			}

			if op == "ancestor" {
				return AncestorAt(args[0].text, location...), nil
			}
			return ParentAt(args[0].text, location...), nil
		}

		if len(args) != 1 {
			return nil, errorAt(e.pos, "%s takes one block, such as ((uuid)) or [[page]] 0 1", op)
		}

		switch args[0].kind {
		case exprBlockRef, exprString, exprSymbol:
			if op == "ancestor" {
				return Ancestor(args[0].text), nil
			}
			return Parent(args[0].text), nil
		}
		return nil, errorAt(args[0].pos, "expected a block, such as ((uuid)) or [[page]] 0 1")
	case "namespace":
		name, err := parseSingleName(e, args)
		if err != nil {
			return nil, err
		}
		return Namespace(name), nil
	case "property", "page-property":
		if len(args) == 0 || len(args) > 2 {
			return nil, errorAt(e.pos, "%s takes a name and an optional value", op)
		}

		name, err := parseKey(args[0])
		if err != nil {
			return nil, err
		}

		// Logseq writes property names with dashes, and reads underscores in
		// queries as dashes.
		name = strings.ReplaceAll(name, "_", "-")

		value := ""
		if len(args) == 2 {
			valueExpr := args[1]
			if valueExpr.kind == exprVector {
				// Logseq uses the first value of a collection.
				if len(valueExpr.items) == 0 {
					return nil, errorAt(valueExpr.pos, "expected a value")
				}
				valueExpr = valueExpr.items[0]
			}

			value, err = parseName(valueExpr)
			if err != nil {
				return nil, err
			}
		}

		if op == "property" {
			return Property(name, value), nil
		}
		return PageProperty(name, value), nil
	case "page-tags":
		tags, err := parseNames(args, e.pos, op)
		if err != nil {
			return nil, err
		}
		return PageTags(tags...), nil
	case "all-page-tags":
		if len(args) > 0 {
			return nil, errorAt(e.pos, "all-page-tags takes no arguments")
		}
		return AllPageTags(), nil
	case "between":
		if len(args) == 3 {
			return nil, errorAt(e.pos, "between with a timestamp property is not supported")
		}

		if len(args) != 2 {
			return nil, errorAt(e.pos, "between takes two dates")
		}

		start, err := parseDate(args[0])
		if err != nil {
			return nil, err
		}

		end, err := parseDate(args[1])
		if err != nil {
			return nil, err
		}

		return Between(start, end), nil
	case "journal", "scheduled", "deadline", "journal-page":
		return parseDateFilter(e, op, args)
	case "title":
		if len(args) != 2 || args[0].kind != exprSymbol {
			return nil, errorAt(e.pos, "title takes a comparison and a text, such as (title starts-with \"projects/\")")
		}

		titleOp, ok := titleOps[strings.ToLower(args[0].text)]
		if !ok {
			return nil, errorAt(args[0].pos, "expected =, starts-with, ends-with or includes")
		}

		text, err := parseName(args[1])
		if err != nil {
			return nil, err
		}
		return Title(titleOp, text), nil
	case "alias":
		name, err := parseSingleName(e, args)
		if err != nil {
			return nil, err
		}
		return Alias(name), nil
	case "has-block":
		if len(args) != 1 {
			return nil, errorAt(e.pos, "has-block takes one filter")
		}

		filter, err := parseFilter(args[0])
		if err != nil {
			return nil, err
		}
		return HasBlock(filter), nil
	case "sort-by", "sample":
		return nil, errorAt(e.pos, "%s can only be used at the top of a query or in its top-level `and`", op)
	}

	return nil, errorAt(e.items[0].pos, "unknown filter %q", e.items[0].text)
}

var dateFields = map[string]DateField{
	"journal":      FieldJournal,
	"scheduled":    FieldScheduled,
	"deadline":     FieldDeadline,
	"journal-page": FieldJournalPage,
}

var titleOps = map[string]TitleOp{
	"=":           TitleEquals,
	"starts-with": TitleStartsWith,
	"ends-with":   TitleEndsWith,
	"includes":    TitleIncludes,
}

var ops = map[string]Op{
	"=":  Equal,
	"<":  Less,
	"<=": LessOrEqual,
	">":  Greater,
	">=": GreaterOrEqual,
}

func parseDateFilter(e *expr, name string, args []*expr) (Filter, error) {
	filter := &DateFilter{Field: dateFields[name]}

	switch len(args) {
	case 0:
		filter.Op = Exists
		return filter, nil
	case 1:
		date, err := parseDate(args[0])
		if err != nil {
			return nil, err
		}

		filter.Op = Equal
		filter.Date = date
		return filter, nil
	case 2:
		if args[0].kind == exprSymbol {
			if op, ok := ops[args[0].text]; ok {
				date, err := parseDate(args[1])
				if err != nil {
					return nil, err
				}

				filter.Op = op
				filter.Date = date
				return filter, nil
			}
		}

		start, err := parseDate(args[0])
		if err != nil {
			return nil, err
		}

		end, err := parseDate(args[1])
		if err != nil {
			return nil, err
		}

		filter.Op = Range
		filter.Date = start
		filter.End = end
		return filter, nil
	}

	return nil, errorAt(e.pos, "%s takes no date, a date, an operator and a date, or two dates", name)
}

var (
	fixedDate     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	offsetNoUnits = regexp.MustCompile(`^[+-]\d+$`)
)

func parseDate(e *expr) (Date, error) {
	switch e.kind {
	case exprPageRef, exprString:
		// A title with brackets is written as a string, see formatPageRef.
		return JournalDate(e.text), nil
	case exprSymbol:
	default:
		return Date{}, errorAt(e.pos, "expected a date, such as `today`, `-7d` or `2024-01-15`")
	}

	text := strings.ToLower(strings.TrimPrefix(e.text, ":"))

	if fixedDate.MatchString(text) {
		date, err := time.ParseInLocation("2006-01-02", text, time.Local)
		if err != nil {
			return Date{}, errorAt(e.pos, "%q is not a valid date", e.text)
		}
		return On(date), nil
	}

	// Logseq reads an offset without a unit as days.
	if offsetNoUnits.MatchString(text) {
		text += "d"
	}

	if input, ok := utils.ParseQueryDateInput(text, time.Now()); ok && !input.HasTime {
		return Relative(text), nil
	}

	return Date{}, errorAt(e.pos, "%q is not a date, expected a date such as `today`, `-7d` or `2024-01-15`", e.text)
}

// parseNames reads the arguments of a filter that takes several names, which
// can also be given as a vector such as `(task [TODO LATER])`.
func parseNames(args []*expr, pos int, op string) ([]string, error) {
	if len(args) == 1 && args[0].kind == exprVector {
		args = args[0].items
	}

	if len(args) == 0 {
		return nil, errorAt(pos, "%s takes one or more values", op)
	}

	names := make([]string, 0, len(args))
	for _, arg := range args {
		name, err := parseName(arg)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}

	return names, nil
}

// namePos returns the position of a name read by parseNames.
func namePos(args []*expr, i int, fallback int) int {
	if len(args) == 1 && args[0].kind == exprVector {
		args = args[0].items
	}

	if i < len(args) {
		return args[i].pos
	}

	return fallback
}

func parseSingleName(e *expr, args []*expr) (string, error) {
	if len(args) != 1 {
		return "", errorAt(e.pos, "%s takes one name", e.op())
	}

	return parseName(args[0])
}

// parseKey reads the name of a property or a sort field. A symbol can be
// written as a keyword, such as `:type`.
func parseKey(e *expr) (string, error) {
	name, err := parseName(e)
	if err != nil {
		return "", err
	}

	if e.kind == exprSymbol {
		name = strings.TrimPrefix(name, ":")
	}

	if strings.TrimSpace(name) == "" {
		return "", errorAt(e.pos, "expected a name")
	}

	return name, nil
}

// parseName reads a name, which can be a symbol, a string or a page ref.
func parseName(e *expr) (string, error) {
	switch e.kind {
	case exprSymbol, exprString, exprPageRef:
		return e.text, nil
	}

	return "", errorAt(e.pos, "expected a name")
}

var taskStatuses = func() map[string]content.TaskStatus {
	statuses := make(map[string]content.TaskStatus)
	for status := content.TaskStatusTodo; status.String() != ""; status++ {
		statuses[status.String()] = status
	}
	return statuses
}()

var priorityValues = map[string]content.Priority{
	"A": content.PriorityA,
	"B": content.PriorityB,
	"C": content.PriorityC,
}

// parseLocation reads the location of a block on a page, such as the `0 1`
// in `(parent [[page]] 0 1)`.
func parseLocation(args []*expr) ([]int, error) {
	location := make([]int, 0, len(args))
	for _, arg := range args {
		idx, err := strconv.Atoi(arg.text)
		if arg.kind != exprSymbol || err != nil || idx < 0 {
			return nil, errorAt(arg.pos, "expected the position of a block, such as 0")
		}
		location = append(location, idx)
	}
	return location, nil
}

type exprKind int

const (
	exprList exprKind = iota
	exprVector
	exprSymbol
	exprString
	exprPageRef
	exprBlockRef
)

// expr is an expression read from the text of a query, before it is turned
// into filters.
type expr struct {
	kind exprKind
	// text is the value of a symbol, a string, the title of a page ref, or the
	// UUID of a block ref.
	text string
	// items are the expressions in a list or vector.
	items []*expr
	// pos is the byte offset where the expression starts.
	pos int
}

// op returns the name of a list, which is its first symbol in lower case.
func (e *expr) op() string {
	if e.kind != exprList || len(e.items) == 0 || e.items[0].kind != exprSymbol {
		return ""
	}

	return strings.ToLower(e.items[0].text)
}

func (e *expr) isList(op string) bool {
	return e.op() == op
}

// reader reads expressions in the EDN-like syntax of Logseq queries, with
// `[[page]]` and `#tag` read as page refs, and `((uuid))` as a block ref.
type reader struct {
	input string
	pos   int
}

func (r *reader) skipSpace() {
	for r.pos < len(r.input) {
		c, size := utf8.DecodeRuneInString(r.input[r.pos:])
		switch {
		case isSpace(c):
			r.pos += size
		case c == ';':
			// A comment runs to the end of the line.
			for r.pos < len(r.input) && r.input[r.pos] != '\n' {
				r.pos++
			}
		default:
			return
		}
	}
}

// isSpace returns true for the characters that separate expressions. Commas
// are whitespace in EDN.
func isSpace(c rune) bool {
	return c == ',' || unicode.IsSpace(c)
}

func (r *reader) read() (*expr, error) {
	r.skipSpace()
	if r.pos >= len(r.input) {
		return nil, errorAt(r.pos, "unexpected end of the query")
	}

	start := r.pos
	switch c := r.input[r.pos]; c {
	case '(':
		if strings.HasPrefix(r.input[r.pos:], "((") {
			id, err := r.readBlockRef()
			if err != nil {
				return nil, err
			}
			return &expr{kind: exprBlockRef, text: id, pos: start}, nil
		}

		r.pos++
		items, err := r.readUntil(')', start)
		if err != nil {
			return nil, err
		}
		return &expr{kind: exprList, items: items, pos: start}, nil
	case '[':
		if strings.HasPrefix(r.input[r.pos:], "[[") {
			title, err := r.readPageRef()
			if err != nil {
				return nil, err
			}
			return &expr{kind: exprPageRef, text: title, pos: start}, nil
		}

		r.pos++
		items, err := r.readUntil(']', start)
		if err != nil {
			return nil, err
		}
		return &expr{kind: exprVector, items: items, pos: start}, nil
	case ')', ']':
		return nil, errorAt(r.pos, "unexpected `%c`", c)
	case '"':
		value, err := r.readString()
		if err != nil {
			return nil, err
		}
		return &expr{kind: exprString, text: value, pos: start}, nil
	case '#':
		r.pos++
		if strings.HasPrefix(r.input[r.pos:], "[[") {
			title, err := r.readPageRef()
			if err != nil {
				return nil, err
			}
			return &expr{kind: exprPageRef, text: title, pos: start}, nil
		}

		tag := r.readSymbol()
		if tag == "" {
			return nil, errorAt(start, "expected a tag after `#`")
		}
		return &expr{kind: exprPageRef, text: tag, pos: start}, nil
	}

	return &expr{kind: exprSymbol, text: r.readSymbol(), pos: start}, nil
}

func (r *reader) readUntil(end byte, start int) ([]*expr, error) {
	var items []*expr
	for {
		r.skipSpace()
		if r.pos >= len(r.input) {
			return nil, errorAt(start, "missing `%c`", end)
		}

		if r.input[r.pos] == end {
			r.pos++
			return items, nil
		}

		item, err := r.read()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
}

// readPageRef reads `[[title]]` and returns the title. Page refs in the title,
// such as `[[a [[b]]]]`, are part of it.
func (r *reader) readPageRef() (string, error) {
	start := r.pos
	depth := 0
	for r.pos < len(r.input) {
		switch {
		case strings.HasPrefix(r.input[r.pos:], "[["):
			depth++
			r.pos += 2
		case strings.HasPrefix(r.input[r.pos:], "]]"):
			depth--
			r.pos += 2
			if depth == 0 {
				title := r.input[start+2 : r.pos-2]
				if strings.TrimSpace(title) == "" {
					return "", errorAt(start, "expected a page title in `[[]]`")
				}

				return title, nil
			}
		default:
			r.pos++
		}
	}

	return "", errorAt(start, "missing `]]`")
}

// readBlockRef reads `((uuid))` and returns the UUID. A list can not start
// with a list in a query, so `((` always starts a block ref.
func (r *reader) readBlockRef() (string, error) {
	start := r.pos
	end := strings.Index(r.input[start:], "))")
	if end < 0 {
		return "", errorAt(start, "missing `))`")
	}

	id := r.input[start+2 : start+end]
	if !isBlockID(id) {
		return "", errorAt(start, "expected the UUID of a block in `(())`")
	}

	r.pos = start + end + 2
	return id, nil
}

// isBlockID returns true for text that can be written in `((uuid))`.
func isBlockID(id string) bool {
	return id != "" && !strings.ContainsFunc(id, func(c rune) bool {
		return isSpace(c) || strings.ContainsRune("()[]\";", c)
	})
}

func (r *reader) readString() (string, error) {
	start := r.pos
	r.pos++

	var b strings.Builder
	for r.pos < len(r.input) {
		c := r.input[r.pos]
		switch c {
		case '"':
			r.pos++
			return b.String(), nil
		case '\\':
			r.pos++
			if r.pos >= len(r.input) {
				return "", errorAt(start, "missing `\"`")
			}

			switch e := r.input[r.pos]; e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			default:
				b.WriteByte(e)
			}
		default:
			b.WriteByte(c)
		}
		r.pos++
	}

	return "", errorAt(start, "missing `\"`")
}

func (r *reader) readSymbol() string {
	start := r.pos
	for r.pos < len(r.input) {
		c, size := utf8.DecodeRuneInString(r.input[r.pos:])
		switch {
		case isSpace(c), strings.ContainsRune("()[]\";", c):
			return r.input[start:r.pos]
		}
		r.pos += size
	}

	return r.input[start:]
}
