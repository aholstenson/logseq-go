package indexing

import (
	"regexp"
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

// AncestorAt matches the blocks below the block at the location on a page at
// any depth, see utils.LocationOf. It matches on every page, so combine it with a
// query for the page.
func AncestorAt(location []int) Query {
	return &fieldEquals{
		field: "ancestor:location",
		value: formatLocation(location),
	}
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

// TextMatch is how a title is compared with a text, for the queries that
// match a part of a title.
type TextMatch int

const (
	// MatchPrefix matches the titles that start with the text.
	MatchPrefix TextMatch = iota
	// MatchSuffix matches the titles that end with the text.
	MatchSuffix
	// MatchIncludes matches the titles that include the text.
	MatchIncludes
)

// titlePattern returns the regular expression for the titles that match the
// text. Titles are stored in lower case, see normalizeRef.
func titlePattern(match TextMatch, text string) string {
	quoted := regexp.QuoteMeta(normalizeRef(text))
	switch match {
	case MatchPrefix:
		return quoted + ".*"
	case MatchSuffix:
		return ".*" + quoted
	}
	return ".*" + quoted + ".*"
}

// PageTitleMatches matches the pages with a title that matches the text, and
// the blocks on them. Titles are matched without regard for case.
func PageTitleMatches(match TextMatch, text string) Query {
	return &fieldPattern{
		field:   SortFieldTitle,
		pattern: titlePattern(match, text),
	}
}

// ReferencesMatching matches the blocks that reference a page with a title
// that matches the text, the same as References does for one title.
func ReferencesMatching(match TextMatch, text string) Query {
	return &fieldPattern{
		field:   "pages:ref",
		pattern: titlePattern(match, text),
	}
}

// PathReferencesMatching matches the blocks about a page with a title that
// matches the text, the same as PathReferences does for one title.
func PathReferencesMatching(match TextMatch, text string) Query {
	return &fieldPattern{
		field:   "path:ref",
		pattern: titlePattern(match, text),
	}
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
