// Package query is a model of queries on a Logseq graph. A query is a tree of
// filters with an optional sort order. It can be built in code, read from text
// with Parse, changed, and written back as text with Query.String. Dates such
// as `today` stay relative in the model, so a stored query keeps its meaning
// from day to day. Queries are run with Graph.Query in the logseq package.
//
// The text form is the simple query syntax of Logseq, the part inside
// `{{query ...}}`, with some additions. Every simple query that Logseq can
// read can be read by Parse, except for `between` with a timestamp property.
//
// Filters from Logseq:
//
//	(and a b ...)            all of the filters match
//	(or a b ...)             one or more of the filters match
//	(not a ...)              the filters do not all match
//	[[page]] or #page        blocks about the page, see PageRef
//	"text"                   blocks with the text
//	(task TODO LATER)        tasks with one of the statuses, also `todo`
//	(priority A B)           blocks with one of the priorities
//	(page [[name]])          blocks on the page
//	(property key value)     blocks with the property value
//	(property key)           blocks with the property
//	(page-property key val)  pages with the property value
//	(page-property key)      pages with the property
//	(page-tags a b)          pages tagged with one of the tags
//	(all-page-tags)          pages that are used as tags
//	(namespace [[name]])     pages directly in the namespace
//	(between -7d today)      blocks on journals from the first to the last date
//	(sort-by key asc)        sort by a property, `desc` if no order is given
//	(sample 10)              a random sample of the results
//
// Additions:
//
//	(scheduled)              blocks with a `SCHEDULED` date
//	(scheduled today)        blocks scheduled on the date
//	(scheduled < today)      blocks scheduled before the date, also with
//	                         `<=`, `>`, `>=` and `=`
//	(scheduled -7d today)    blocks scheduled from the first to the last date
//	(deadline ...)           the same for `DEADLINE` dates
//	(journal ...)            the same for the date of the journal a block is
//	                         on; a range is written as `between`
//	(page-ref "title")       the same as [[title]], for titles that have
//	                         brackets
//	(refs [[page]])          blocks that reference the page themselves,
//	                         without the blocks below them, see RefsFilter
//	(parent ((uuid)))        blocks directly below the block with the
//	                         `id::` property, see ParentFilter
//	(ancestor ((uuid)))      blocks below the block at any depth, see
//	                         AncestorFilter
//	(journal-page ...)       journal pages, with the same dates as
//	                         `journal`; it matches pages, not blocks
//	(title starts-with "x")  pages with a title that starts with the text,
//	                         also with `=`, `ends-with` and `includes`
//	(alias [[name]])         pages with the alias
//	(has-block (task TODO))  pages with one or more blocks that match the
//	                         filter
//	(refs (page-tags x))     blocks that reference one or more of the pages
//	                         that the filter matches, see ReferencesFilter;
//	                         `page-ref` does the same for blocks about them
//	(sort-by priority asc)   sort by a built-in value: `priority`,
//	                         `scheduled`, `deadline`, `journal-day` or `title`
//
// A date is `today`, `yesterday`, `tomorrow`, an offset from today such as
// `-7d`, `+2w`, `+1m` or `-1y`, a date such as `2024-01-15`, or the title of a
// journal page such as `[[Jan 15th, 2024]]`. An offset without a unit is in
// days.
//
// Text that does not start with `(`, `[`, `"` or `#` is a search for the text.
package query
