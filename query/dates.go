package query

import (
	"strconv"
	"time"
)

// Date is a date in a query. It is relative to today, such as `today` or
// `-7d`, a fixed date, or the title of a journal page. Only one of the fields
// is set. The zero Date is not a valid date.
type Date struct {
	// Relative is a date relative to today, such as `today`, `-7d` or `+1w`,
	// in the forms that logseq.ParseDateInput reads.
	Relative string

	// Journal is the title of a journal page, such as `Jan 15th, 2024`, which
	// stands for the date of the journal.
	Journal string

	// Fixed is a fixed date, at midnight in its location.
	Fixed time.Time
}

// Today is the date of today.
func Today() Date {
	return Date{Relative: "today"}
}

// DaysFromToday is the date a number of days from today, or before today
// for a negative number.
func DaysFromToday(days int) Date {
	switch {
	case days == 0:
		return Today()
	case days > 0:
		return Date{Relative: "+" + strconv.Itoa(days) + "d"}
	default:
		return Date{Relative: strconv.Itoa(days) + "d"}
	}
}

// Relative is a date relative to today, such as `today`, `-7d` or `+1w`.
func Relative(input string) Date {
	return Date{Relative: input}
}

// On is a fixed date. The time of day is left out.
func On(date time.Time) Date {
	return Date{Fixed: time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())}
}

// JournalDate is the date of the journal page with the title, such as
// `Jan 15th, 2024`.
func JournalDate(title string) Date {
	return Date{Journal: title}
}

// IsZero returns true if no field of the date is set.
func (d Date) IsZero() bool {
	return d.Relative == "" && d.Journal == "" && d.Fixed.IsZero()
}

// String returns the date in the text form that Parse reads.
func (d Date) String() string {
	switch {
	case d.Relative != "":
		return d.Relative
	case d.Journal != "":
		return formatPageRef(d.Journal)
	case !d.Fixed.IsZero():
		return d.Fixed.Format("2006-01-02")
	}

	return ""
}

// DateField is the date that a DateFilter matches on.
type DateField int

const (
	// FieldJournal is the date of the journal page a block is on, which
	// Logseq calls the journal day.
	FieldJournal DateField = iota
	// FieldScheduled is the `SCHEDULED` date of a block.
	FieldScheduled
	// FieldDeadline is the `DEADLINE` date of a block.
	FieldDeadline
)

var dateFieldNames = map[DateField]string{
	FieldJournal:   "journal",
	FieldScheduled: "scheduled",
	FieldDeadline:  "deadline",
}

// Op is how a DateFilter compares a date.
type Op int

const (
	// Exists matches when the block has the date, whatever it is.
	Exists Op = iota
	// Equal matches the date.
	Equal
	// Less matches dates before the date.
	Less
	// LessOrEqual matches the date and dates before it.
	LessOrEqual
	// Greater matches dates after the date.
	Greater
	// GreaterOrEqual matches the date and dates after it.
	GreaterOrEqual
	// Range matches the dates from Date to End, both included.
	Range
)

var opNames = map[Op]string{
	Equal:          "=",
	Less:           "<",
	LessOrEqual:    "<=",
	Greater:        ">",
	GreaterOrEqual: ">=",
}

// DateFilter matches the blocks with a date that compares with a date in the
// filter. It is written as `(scheduled < today)`, `(deadline -7d today)` or
// `(journal today)`, and a range of journal dates as `(between -7d today)`.
type DateFilter struct {
	// Field is the date of the block to compare.
	Field DateField

	// Op is how to compare the date.
	Op Op

	// Date is the date to compare with. Not used for Exists. The first date
	// of the range for Range.
	Date Date

	// End is the last date of the range for Range. Not used for other
	// operations.
	End Date
}

// Journal creates a filter that compares the date of the journal a block is
// on with the date.
func Journal(op Op, date Date) *DateFilter {
	return &DateFilter{Field: FieldJournal, Op: op, Date: date}
}

// Between creates a filter that matches the blocks on journals from start to
// end, both included.
func Between(start Date, end Date) *DateFilter {
	return &DateFilter{Field: FieldJournal, Op: Range, Date: start, End: end}
}

// OnJournal creates a filter that matches the blocks on journals.
func OnJournal() *DateFilter {
	return &DateFilter{Field: FieldJournal, Op: Exists}
}

// Scheduled creates a filter that compares the `SCHEDULED` date of a block
// with the date.
func Scheduled(op Op, date Date) *DateFilter {
	return &DateFilter{Field: FieldScheduled, Op: op, Date: date}
}

// ScheduledBetween creates a filter that matches the blocks scheduled from
// start to end, both included.
func ScheduledBetween(start Date, end Date) *DateFilter {
	return &DateFilter{Field: FieldScheduled, Op: Range, Date: start, End: end}
}

// HasScheduled creates a filter that matches the blocks with a `SCHEDULED`
// date.
func HasScheduled() *DateFilter {
	return &DateFilter{Field: FieldScheduled, Op: Exists}
}

// Deadline creates a filter that compares the `DEADLINE` date of a block with
// the date.
func Deadline(op Op, date Date) *DateFilter {
	return &DateFilter{Field: FieldDeadline, Op: op, Date: date}
}

// DeadlineBetween creates a filter that matches the blocks with a deadline
// from start to end, both included.
func DeadlineBetween(start Date, end Date) *DateFilter {
	return &DateFilter{Field: FieldDeadline, Op: Range, Date: start, End: end}
}

// HasDeadline creates a filter that matches the blocks with a `DEADLINE`
// date.
func HasDeadline() *DateFilter {
	return &DateFilter{Field: FieldDeadline, Op: Exists}
}
