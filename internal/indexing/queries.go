package indexing

import (
	"fmt"
	"regexp"
	"regexp/syntax"
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/content"
)

type Query interface {
	isQuery()
}

type all struct{}

func (a *all) isQuery() {}

type none struct{}

func (n *none) isQuery() {}

type and struct {
	clauses []Query
}

func (a *and) isQuery() {}

type or struct {
	clauses []Query
}

func (o *or) isQuery() {}

type not struct {
	clause Query
}

func (n *not) isQuery() {}

type fieldMatches struct {
	field   string
	text    string
	partial bool
}

func (f *fieldMatches) isQuery() {}

type fieldEquals struct {
	field string
	value string
	// normalized marks fields that hold a page title, which is matched without
	// regard for case.
	normalized bool
}

func (f *fieldEquals) isQuery() {}

type fieldRefs struct {
	field  string
	target string
	tag    bool
}

func (f *fieldRefs) isQuery() {}

// fieldPattern matches a keyword field with a regular expression, which must
// match the whole value.
type fieldPattern struct {
	field   string
	pattern string
}

func (f *fieldPattern) isQuery() {}

// numberRange matches a numeric field from min to max. Infinite bounds leave
// that end open.
type numberRange struct {
	field        string
	min, max     float64
	minInclusive bool
	maxInclusive bool
}

func (n *numberRange) isQuery() {}

// dateRange matches a date field from start, inclusive, to end, exclusive. A
// zero time leaves that end of the range open.
type dateRange struct {
	field string
	start time.Time
	end   time.Time
}

func (d *dateRange) isQuery() {}

func All() *all {
	return &all{}
}

func None() *none {
	return &none{}
}

func And(clauses ...Query) *and {
	return &and{
		clauses: clauses,
	}
}

func Or(clauses ...Query) *or {
	return &or{
		clauses: clauses,
	}
}

func Not(clause Query) *not {
	return &not{
		clause: clause,
	}
}

func TitleMatches(text string) Query {
	return &fieldMatches{
		field: "title",
		text:  text,
	}
}

func TitlePartiallyMatches(text string) Query {
	return &fieldMatches{
		field:   "title",
		text:    text,
		partial: true,
	}
}

func BlockIDEquals(id string) Query {
	return &fieldEquals{
		field: "id",
		value: id,
	}
}

// ParentIs matches the blocks directly below the block with the id, which is
// the value of its `id` property.
func ParentIs(id string) Query {
	return &fieldEquals{
		field: "parent",
		value: id,
	}
}

// AncestorIs matches the blocks below the block with the id at any depth,
// which is the value of its `id` property.
func AncestorIs(id string) Query {
	return &fieldEquals{
		field: "ancestor",
		value: id,
	}
}

// ParentAt matches the blocks directly below the block at the location on a
// page, see utils.LocationOf. It matches on every page, so combine it with a query
// for the page.
func ParentAt(location []int) Query {
	return &fieldEquals{
		field: "parent:location",
		value: formatLocation(location),
	}
}

// HasBlocks matches the pages with one or more blocks.
func HasBlocks() Query {
	return &fieldEquals{
		field: "has",
		value: "blocks",
	}
}

// TopLevel matches the blocks at the top of their page, which have no block
// above them.
func TopLevel() Query {
	return &numberRange{
		field:        "depth",
		min:          1,
		max:          1,
		minInclusive: true,
		maxInclusive: true,
	}
}

// AncestorAt matches the blocks below the block at the location on a page at
// any depth, see utils.LocationOf. It matches on every page, so combine it with a
// query for the page.
func AncestorAt(location []int) Query {
	return &fieldEquals{
		field: "ancestor:location",
		value: formatLocation(location),
	}
}

// TextIs matches the blocks with the text, with regard for case. The text of
// a block is its Markdown, the same as :block/content in Logseq, see
// blockText.
func TextIs(text string) Query {
	return &fieldEquals{
		field: "text",
		value: text,
	}
}

// TextMatches matches the blocks with a text that matches the text, with
// regard for case, the same as TextIs does for the whole text.
func TextMatches(match TextMatch, text string) (Query, error) {
	pattern, err := textPattern(match, text)
	if err != nil {
		return nil, err
	}
	return &fieldPattern{field: "text", pattern: pattern}, nil
}

func ContentMatches(text string) Query {
	return &fieldMatches{
		field: "content",
		text:  text,
	}
}

func PropertyMatches(property string, text string) Query {
	return &fieldMatches{
		field: "prop:" + property,
		text:  text,
	}
}

func PropertyEquals(property string, value string) Query {
	return &fieldEquals{
		field: "prop:" + property,
		value: value,
	}
}

// PropertyNumberBetween matches the property when its value is a number from
// min to max. Use math.Inf to leave an end open.
func PropertyNumberBetween(property string, min float64, max float64, minInclusive bool, maxInclusive bool) Query {
	return &numberRange{
		field:        "prop:" + property + ":number",
		min:          min,
		max:          max,
		minInclusive: minInclusive,
		maxInclusive: maxInclusive,
	}
}

// PropertyDateBetween matches the property when its value is a date from
// start, inclusive, to end, exclusive. A zero time leaves that end open.
// Dates are values such as 2024-01-15, 20240115 or a reference to a journal.
func PropertyDateBetween(property string, start time.Time, end time.Time) Query {
	return &dateRange{
		field: "prop:" + property + ":date",
		start: start,
		end:   end,
	}
}

func HasProperty(property string) Query {
	return &fieldEquals{
		field:      "props",
		value:      property,
		normalized: true,
	}
}

func PropertyReferences(property string, target string) Query {
	return &fieldRefs{
		field:  "prop:" + property,
		target: target,
	}
}

func PropertyReferencesTag(property string, target string) Query {
	return &fieldRefs{
		field:  "prop:" + property,
		target: target,
		tag:    true,
	}
}

func InNamespace(namespace string) Query {
	return &fieldEquals{
		field:      "namespace",
		value:      namespace,
		normalized: true,
	}
}

func UnderNamespace(namespace string) Query {
	return &fieldEquals{
		field:      "namespaces",
		value:      namespace,
		normalized: true,
	}
}

func HasAlias(alias string) Query {
	return &fieldRefs{
		field:  "alias",
		target: alias,
	}
}

func References(page string) Query {
	return &fieldRefs{
		field:  "pages",
		target: page,
	}
}

func ReferencesTag(tag string) Query {
	return &fieldRefs{
		field:  "pages",
		target: tag,
		tag:    true,
	}
}

func PathReferences(page string) Query {
	return &fieldRefs{
		field:  "path",
		target: page,
	}
}

// OnPages matches the blocks on the pages with the given sub paths. Matches
// nothing if no sub path is given.
func OnPages(subPaths ...string) Query {
	if len(subPaths) == 0 {
		return None()
	}

	clauses := make([]Query, 0, len(subPaths))
	for _, subPath := range subPaths {
		clauses = append(clauses, &fieldEquals{
			field: "page",
			value: subPath,
		})
	}

	return Or(clauses...)
}

// BlockAt matches the block at the location on the page with the sub path,
// see utils.LocationOf.
func BlockAt(subPath string, location []int) Query {
	return &fieldEquals{
		field: "_id",
		value: blockIDAt(subPath, location),
	}
}

// PagesAt matches the pages with the given sub paths. Matches nothing if no
// sub path is given.
func PagesAt(subPaths ...string) Query {
	if len(subPaths) == 0 {
		return None()
	}

	clauses := make([]Query, 0, len(subPaths))
	for _, subPath := range subPaths {
		clauses = append(clauses, &fieldEquals{
			field: "_id",
			value: subPath,
		})
	}

	return Or(clauses...)
}

// TextMatch is how a title or the text of a block is compared with a text,
// for the queries that match a part of it.
type TextMatch int

const (
	// MatchPrefix matches the values that start with the text.
	MatchPrefix TextMatch = iota
	// MatchSuffix matches the values that end with the text.
	MatchSuffix
	// MatchIncludes matches the values that include the text.
	MatchIncludes
	// MatchFind matches the values with a match of the text as a regular
	// expression anywhere in them, the same as re-find in Clojure. The
	// syntax is that of the regexp package. `^` and `$` are supported only
	// at the start and the end of the expression, and `\b` and `\B` are not
	// supported.
	MatchFind
)

// anyText is the regular expression for any text, also over more than one
// line.
const anyText = "(?s:.*)"

// textPattern returns the regular expression for the values that match the
// text. The values can have more than one line. The index matches the
// expression with all of the value, so a part of a value is matched with
// anyText before and after it.
func textPattern(match TextMatch, text string) (string, error) {
	quoted := regexp.QuoteMeta(text)
	switch match {
	case MatchPrefix:
		return quoted + anyText, nil
	case MatchSuffix:
		return anyText + quoted, nil
	case MatchIncludes:
		return anyText + quoted + anyText, nil
	case MatchFind:
		return findPattern(text)
	}
	return "", fmt.Errorf("unknown text match %d", match)
}

// findPattern returns the regular expression for the values with a match of
// the expression anywhere in them. `^` at the start and `$` at the end
// remove anyText on that side.
func findPattern(expr string) (string, error) {
	parsed, err := syntax.Parse(expr, syntax.Perl)
	if err != nil {
		return "", fmt.Errorf("the regular expression %q is not valid: %w", expr, err)
	}

	pattern, err := findAlternatives(parsed)
	if err != nil {
		return "", fmt.Errorf("the regular expression %q is not supported: %w", expr, err)
	}
	return pattern, nil
}

// findAlternatives returns the regular expression for the values with a
// match of one of the alternatives in re, or of re if it has none.
func findAlternatives(re *syntax.Regexp) (string, error) {
	if re.Op != syntax.OpAlternate {
		return findOne(re)
	}

	parts := make([]string, 0, len(re.Sub))
	for _, sub := range re.Sub {
		part, err := findOne(sub)
		if err != nil {
			return "", err
		}
		parts = append(parts, "(?:"+part+")")
	}
	return strings.Join(parts, "|"), nil
}

func findOne(re *syntax.Regexp) (string, error) {
	subs := []*syntax.Regexp{re}
	if re.Op == syntax.OpConcat {
		subs = re.Sub
	}

	start, end := anyText, anyText
	if len(subs) > 0 && subs[0].Op == syntax.OpBeginText {
		start, subs = "", subs[1:]
	}
	if len(subs) > 0 && subs[len(subs)-1].Op == syntax.OpEndText {
		end, subs = "", subs[:len(subs)-1]
	}

	var body strings.Builder
	for _, sub := range subs {
		if err := checkFindable(sub); err != nil {
			return "", err
		}
		body.WriteString(sub.String())
	}

	return start + "(?:" + body.String() + ")" + end, nil
}

// checkFindable returns an error for the parts of a regular expression that
// the index can not match, and makes all repeats greedy. A lazy repeat finds
// a match in the same values as a greedy one, but the index does not accept
// it.
func checkFindable(re *syntax.Regexp) error {
	switch re.Op {
	case syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText, syntax.OpEndText:
		return fmt.Errorf("^ and $ are only supported at the start and the end")
	case syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return fmt.Errorf("\\b and \\B are not supported")
	}

	re.Flags &^= syntax.NonGreedy
	for _, sub := range re.Sub {
		if err := checkFindable(sub); err != nil {
			return err
		}
	}
	return nil
}

// titlePattern returns the regular expression for the titles that match the
// text. Titles are stored in lower case, see normalizeRef, so a regular
// expression is matched without regard for case instead.
func titlePattern(match TextMatch, text string) (string, error) {
	if match == MatchFind {
		return textPattern(match, "(?i)"+text)
	}
	return textPattern(match, normalizeRef(text))
}

// PageTitleMatches matches the pages with a title that matches the text, and
// the blocks on them. Titles are matched without regard for case.
func PageTitleMatches(match TextMatch, text string) (Query, error) {
	return titleQuery(SortFieldTitle, match, text)
}

// ReferencesMatching matches the blocks that reference a page with a title
// that matches the text, the same as References does for one title.
func ReferencesMatching(match TextMatch, text string) (Query, error) {
	return titleQuery("pages:ref", match, text)
}

// PathReferencesMatching matches the blocks about a page with a title that
// matches the text, the same as PathReferences does for one title.
func PathReferencesMatching(match TextMatch, text string) (Query, error) {
	return titleQuery("path:ref", match, text)
}

func titleQuery(field string, match TextMatch, text string) (Query, error) {
	pattern, err := titlePattern(match, text)
	if err != nil {
		return nil, err
	}
	return &fieldPattern{field: field, pattern: pattern}, nil
}

// PageTitleIs matches the page with the title, and the blocks on it. Titles
// are matched without regard for case.
func PageTitleIs(title string) Query {
	return &fieldEquals{
		field:      SortFieldTitle,
		value:      title,
		normalized: true,
	}
}

func LinksToURL(url string) Query {
	return &fieldEquals{
		field: "link",
		value: url,
	}
}

func IsTask() Query {
	return Or(IsOpenTask(), IsClosedTask())
}

func IsOpenTask() Query {
	return &fieldEquals{
		field: "task:state",
		value: "open",
	}
}

func IsClosedTask() Query {
	return &fieldEquals{
		field: "task:state",
		value: "closed",
	}
}

func TaskStatusIs(statuses ...content.TaskStatus) Query {
	clauses := make([]Query, 0, len(statuses))
	for _, status := range statuses {
		marker := status.String()
		if marker == "" {
			continue
		}

		clauses = append(clauses, &fieldEquals{
			field: "task",
			value: marker,
		})
	}

	if len(clauses) == 0 {
		return None()
	}

	return Or(clauses...)
}

func PriorityIs(priority content.Priority) Query {
	value := priorityValue(priority)
	if value == "" {
		return None()
	}

	return &fieldEquals{
		field: "priority",
		value: value,
	}
}

func ScheduledBetween(start time.Time, end time.Time) Query {
	return &dateRange{
		field: "scheduled",
		start: start,
		end:   end,
	}
}

func JournalBetween(start time.Time, end time.Time) Query {
	return &dateRange{
		field: "date",
		start: start,
		end:   end,
	}
}

func DeadlineBetween(start time.Time, end time.Time) Query {
	return &dateRange{
		field: "deadline",
		start: start,
		end:   end,
	}
}
