package logseq

import (
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
)

type Query = indexing.Query

// All returns a query that matches everything.
func All() Query {
	return indexing.All()
}

// None returns a query that matches nothing.
func None() Query {
	return indexing.None()
}

// And returns a query that matches all of the given queries.
func And(queries ...Query) Query {
	return indexing.And(queries...)
}

// Or returns a query that matches any of the given queries.
func Or(queries ...Query) Query {
	return indexing.Or(queries...)
}

// Not returns a query that matches anything that does not match the given query.
func Not(query Query) Query {
	return indexing.Not(query)
}

// TitleMatches returns a query that matches the title of a page against a
// text.
func TitleMatches(text string) Query {
	return indexing.TitleMatches(text)
}

// ContentMatches returns a query that matches the content of a page or a
// block against a text.
func ContentMatches(text string) Query {
	return indexing.ContentMatches(text)
}

// BlockIDEquals matches the block with the given id, which is the identifier
// that block references such as `((id))` point at.
func BlockIDEquals(id string) Query {
	return indexing.BlockIDEquals(id)
}

// PropertyMatches returns a query that matches the given text against the
// value of a property.
func PropertyMatches(property, text string) Query {
	return indexing.PropertyMatches(property, text)
}

// PropertyEquals returns a query that checks that a property has a specific
// value.
func PropertyEquals(property, value string) Query {
	return indexing.PropertyEquals(property, value)
}

// HasProperty returns a query that matches pages and blocks that have the
// given property, whatever its value is. A property with an empty value, such
// as `tags:: `, also matches. Property names are matched without regard for
// case, so `template` also matches `Template::`.
func HasProperty(property string) Query {
	return indexing.HasProperty(property)
}

// PropertyReferences checks if the given property references a specific
// page, as both a link and a tag.
func PropertyReferences(property, target string) Query {
	return indexing.PropertyReferences(property, target)
}

// PropertyReferencesTag checks if the given property references a
// specific page as a tag.
func PropertyReferencesTag(property, tag string) Query {
	return indexing.PropertyReferencesTag(property, tag)
}

// InNamespace matches the pages that are directly in a namespace, so `Parent`
// matches the page `Parent/Child` but not `Parent/Child/Grandchild`.
func InNamespace(namespace string) Query {
	return indexing.InNamespace(namespace)
}

// UnderNamespace matches the pages anywhere in a namespace, so `Parent` matches
// both `Parent/Child` and `Parent/Child/Grandchild`.
func UnderNamespace(namespace string) Query {
	return indexing.UnderNamespace(namespace)
}

// HasAlias matches the page that has the given title as one of the aliases in
// its `alias` property.
func HasAlias(alias string) Query {
	return indexing.HasAlias(alias)
}

// References returns a query that matches if the given page is referenced
// both as a link and a tag.
func References(page string) Query {
	return indexing.References(page)
}

// ReferencesTag returns a query that matches if the given page is
// referenced as a tag.
func ReferencesTag(page string) Query {
	return indexing.ReferencesTag(page)
}

// PathReferences matches the blocks that are about the given page: the blocks
// on that page, the blocks that reference it, and all blocks below a block
// that references it. Logseq calls these the path refs of a block, and a
// simple query such as `{{query [[page]]}}` matches on them. Page titles are
// matched without regard for case. Matches only blocks, never pages.
//
// Use it with Not to leave out a part of the graph, such as all tasks below a
// block tagged `#shopping`.
func PathReferences(page string) Query {
	return indexing.PathReferences(page)
}

// LinksToURL returns a query that checks if a block or page links to a
// URL.
func LinksToURL(url string) Query {
	return indexing.LinksToURL(url)
}

// IsTask matches the blocks that have a task marker, such as `TODO` or `DONE`.
func IsTask() Query {
	return indexing.IsTask()
}

// IsOpenTask matches the blocks with a task that is not finished, such as
// `TODO`, `DOING` or `WAITING`. See content.TaskStatus.IsOpen.
func IsOpenTask() Query {
	return indexing.IsOpenTask()
}

// IsClosedTask matches the blocks with a task that is finished, either as
// `DONE` or as cancelled. See content.TaskStatus.IsClosed.
func IsClosedTask() Query {
	return indexing.IsClosedTask()
}

// TaskStatusIs matches the blocks with a task that has one of the given
// statuses. The two spellings of cancelled are different statuses, so pass
// both content.TaskStatusCancelled and content.TaskStatusCanceled to match
// all cancelled tasks. Matches nothing if no known status is given.
func TaskStatusIs(statuses ...content.TaskStatus) Query {
	return indexing.TaskStatusIs(statuses...)
}

// PriorityIs matches the blocks that have the given priority, such as `[#A]`.
// Matches nothing for content.PriorityNone, use Not with the other priorities
// to find blocks without a priority.
func PriorityIs(priority content.Priority) Query {
	return indexing.PriorityIs(priority)
}

// ScheduledBetween matches the blocks with a `SCHEDULED` date from start,
// inclusive, to end, exclusive. A zero time leaves that end of the range open,
// so passing two zero times matches every block with a scheduled date.
//
// A date without a time of day is at midnight in the local time zone, so a
// range that covers a whole day matches it.
func ScheduledBetween(start, end time.Time) Query {
	return indexing.ScheduledBetween(start, end)
}

// DeadlineBetween matches the blocks with a `DEADLINE` date from start,
// inclusive, to end, exclusive. A zero time leaves that end of the range open,
// so passing two zero times matches every block with a deadline.
//
// A date without a time of day is at midnight in the local time zone, so a
// range that covers a whole day matches it.
func DeadlineBetween(start, end time.Time) Query {
	return indexing.DeadlineBetween(start, end)
}
