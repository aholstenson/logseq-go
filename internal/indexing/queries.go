package indexing

import (
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
