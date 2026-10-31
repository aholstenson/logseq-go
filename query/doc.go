// Package query is a model of queries on a Logseq graph. A query is a tree of
// filters with an optional sort order. It can be built in code, read from text
// with Parse, changed, and written back as text with Query.String. Dates such
// as `today` stay relative in the model, so a stored query keeps its meaning
// from day to day. Queries are run with Graph.Query in the logseq package.
//
// The text form is the simple query syntax of Logseq, the part inside
// `{{query ...}}`, with some additions. Every simple query that Logseq can
// read can be read by Parse. `(between created-at -7d today)` is read as two
// comparisons of the time, see TimestampFilter.
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
//	(by-page false)          show the results without grouping them by
//	                         page, see AdvancedQuery.GroupByPage
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
//	(parent [[page]])        blocks at the top of the page, which is their
//	                         parent in Logseq
//	(top-level)              blocks at the top of their page, see
//	                         TopLevelFilter
//	(journal-page ...)       journal pages, with the same dates as
//	                         `journal`; it matches pages, not blocks
//	(title starts-with "x")  pages with a title that starts with the text,
//	                         also with `=`, `ends-with`, `includes` and
//	                         `matches` for a regular expression, see
//	                         TitleMatches
//	(content includes "x")   blocks with a text that includes the text,
//	                         with regard for case, also with `=`,
//	                         `starts-with`, `ends-with` and `matches`, see
//	                         ContentFilter
//	(alias [[name]])         pages with the alias
//	(under-namespace [[x]])  pages in the namespace at any depth, see
//	                         NamespaceFilter
//	(has-block (task TODO))  pages with one or more blocks that match the
//	                         filter; `(has-block)` for pages with blocks
//	(property rating > 7)    blocks with a property that is a number or a
//	                         date that compares with the value, also with
//	                         `=`, `<`, `<=`, `>=` and dates such as `today`;
//	                         `page-property` for pages, see
//	                         PropertyCompareFilter
//	(updated-at > -7d-ms)    blocks changed after the time, also with
//	                         `created-at`, the other comparisons, and
//	                         `page-updated-at` for pages; the time is an
//	                         input with a time of day, such as
//	                         `right-now-ms`, or a time such as
//	                         `2024-01-15T10:30`, see TimestampFilter
//	(page (title = "x"))     blocks on the pages that the filter matches,
//	                         see OnPageFilter
//	(parent (task TODO))     blocks directly below a block that the filter
//	                         matches; `ancestor` for any depth, see
//	                         BelowFilter
//	(has-child (task TODO))  blocks with a block directly below that the
//	                         filter matches; `has-descendant` for any depth,
//	                         see HasChildFilter
//	(refs (page-tags x))     blocks that reference one or more of the pages
//	                         that the filter matches, see ReferencesFilter;
//	                         `page-ref` does the same for blocks about them
//	(sort-by priority asc)   sort by a built-in value: `priority`,
//	                         `scheduled`, `deadline`, `journal-day`, `title`
//	                         or `updated-at`
//	(limit 10)               the first results, after they are sorted and
//	                         sampled; like `sort-by` it is read from the
//	                         top-level `and`
//
// A date is `today`, `yesterday`, `tomorrow`, an offset from today such as
// `-7d`, `+2w`, `+1m` or `-1y`, a date such as `2024-01-15`, or the title of a
// journal page such as `[[Jan 15th, 2024]]`. An offset without a unit is in
// days.
//
// Text that does not start with `(`, `[`, `"` or `#` is a search for the text.
package query
