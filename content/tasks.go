package content

import (
	"strconv"
	"time"
)

// TaskStatus is the type of a task.
type TaskStatus int

const (
	TaskStatusNone TaskStatus = iota
	// TaskStatusTodo is a TODO task.
	TaskStatusTodo
	// TaskStatusDoing is a DOING task.
	TaskStatusDoing
	// TaskStatusDone is a DONE task.
	TaskStatusDone
	// TaskStatusLater is a LATER task.
	TaskStatusLater
	// TaskStatusNow is a NOW task.
	TaskStatusNow
	// TaskStatusCancelled is a CANCELLED task.
	TaskStatusCancelled
	// TaskStatusCanceled is a CANCELED task.
	TaskStatusCanceled
	// TaskStatusInProgress is a IN-PROGRESS task.
	TaskStatusInProgress
	// TaskStatusWait is a WAIT task.
	TaskStatusWait
	// TaskStatusWaiting is a WAITING task.
	TaskStatusWaiting
)

type TaskMarker struct {
	baseNode

	Status TaskStatus
}

func NewTaskMarker(t TaskStatus) *TaskMarker {
	return &TaskMarker{
		Status: t,
	}
}

// WithStatus sets the status of the task marker.
func (t *TaskMarker) WithStatus(status TaskStatus) *TaskMarker {
	t.Status = status
	return t
}

func (t *TaskMarker) debug(p *debugPrinter) {
	p.StartType("TaskMarker")
	switch t.Status {
	case TaskStatusNone:
		p.Field("type", "none")
	case TaskStatusTodo:
		p.Field("type", "todo")
	case TaskStatusDoing:
		p.Field("type", "doing")
	case TaskStatusDone:
		p.Field("type", "done")
	case TaskStatusLater:
		p.Field("type", "later")
	case TaskStatusNow:
		p.Field("type", "now")
	case TaskStatusCancelled:
		p.Field("type", "cancelled")
	case TaskStatusInProgress:
		p.Field("type", "in-progress")
	case TaskStatusWait:
		p.Field("type", "wait")
	case TaskStatusWaiting:
		p.Field("type", "waiting")
	}
	p.EndType()
}

func (t *TaskMarker) isInline() {}

// Priority is how urgent a task is.
type Priority int

const (
	// PriorityNone is used when a task does not have a priority.
	PriorityNone Priority = iota
	// PriorityA is the highest priority, written as `[#A]`.
	PriorityA
	// PriorityB is the middle priority, written as `[#B]`.
	PriorityB
	// PriorityC is the lowest priority, written as `[#C]`.
	PriorityC
)

// TaskPriority is the priority of a task. Logseq puts it at the start of the
// content of a block, after the task marker if there is one:
//
//	TODO [#A] Water the plants
//
// Brackets used this way elsewhere in a block are not a priority to Logseq and
// are kept as text instead.
type TaskPriority struct {
	baseNode

	Priority Priority
}

func NewTaskPriority(priority Priority) *TaskPriority {
	return &TaskPriority{
		Priority: priority,
	}
}

// WithPriority sets the priority.
func (t *TaskPriority) WithPriority(priority Priority) *TaskPriority {
	t.Priority = priority
	return t
}

func (t *TaskPriority) debug(p *debugPrinter) {
	p.StartType("TaskPriority")
	switch t.Priority {
	case PriorityNone:
		p.Field("priority", "none")
	case PriorityA:
		p.Field("priority", "a")
	case PriorityB:
		p.Field("priority", "b")
	case PriorityC:
		p.Field("priority", "c")
	}
	p.EndType()
}

func (t *TaskPriority) isInline() {}

var _ InlineNode = (*TaskPriority)(nil)

// TaskDateType is the type of a date that is attached to a task.
type TaskDateType int

const (
	// TaskDateTypeScheduled is a `SCHEDULED` date, which is when work on a
	// task is intended to start.
	TaskDateTypeScheduled TaskDateType = iota
	// TaskDateTypeDeadline is a `DEADLINE` date, which is when a task has to
	// be finished.
	TaskDateTypeDeadline
)

// RepeaterType controls how the next occurrence of a repeating date is picked
// when the task is completed.
type RepeaterType int

const (
	// RepeaterTypeCumulate, written as `+`, moves the date forward one
	// interval, even if that leaves it in the past.
	RepeaterTypeCumulate RepeaterType = iota
	// RepeaterTypeCatchUp, written as `++`, moves the date forward whole
	// intervals until it is in the future.
	RepeaterTypeCatchUp
	// RepeaterTypeRestart, written as `.+`, moves the date to one interval
	// after the day the task was completed.
	RepeaterTypeRestart
)

// RepeaterUnit is the unit of the interval of a Repeater.
type RepeaterUnit int

const (
	// RepeaterUnitHour repeats every N hours, written as `h`.
	RepeaterUnitHour RepeaterUnit = iota
	// RepeaterUnitDay repeats every N days, written as `d`.
	RepeaterUnitDay
	// RepeaterUnitWeek repeats every N weeks, written as `w`.
	RepeaterUnitWeek
	// RepeaterUnitMonth repeats every N months, written as `m`.
	RepeaterUnitMonth
	// RepeaterUnitYear repeats every N years, written as `y`.
	RepeaterUnitYear
)

// Repeater describes how a `SCHEDULED` or `DEADLINE` date repeats when the
// task it belongs to is completed.
type Repeater struct {
	// Type is how the next occurrence is picked.
	Type RepeaterType
	// Value is how many units to move the date forward by.
	Value int
	// Unit is the unit of Value.
	Unit RepeaterUnit
}

func NewRepeater(repeaterType RepeaterType, value int, unit RepeaterUnit) *Repeater {
	return &Repeater{
		Type:  repeaterType,
		Value: value,
		Unit:  unit,
	}
}

// String formats the repeater the way it is stored in Markdown, such as `.+1d`.
// Returns an empty string if the type or unit is not one of the known values.
func (r *Repeater) String() string {
	var prefix string
	switch r.Type {
	case RepeaterTypeCumulate:
		prefix = "+"
	case RepeaterTypeCatchUp:
		prefix = "++"
	case RepeaterTypeRestart:
		prefix = ".+"
	default:
		return ""
	}

	var unit string
	switch r.Unit {
	case RepeaterUnitHour:
		unit = "h"
	case RepeaterUnitDay:
		unit = "d"
	case RepeaterUnitWeek:
		unit = "w"
	case RepeaterUnitMonth:
		unit = "m"
	case RepeaterUnitYear:
		unit = "y"
	default:
		return ""
	}

	return prefix + strconv.Itoa(r.Value) + unit
}

// TaskDate is a `SCHEDULED` or `DEADLINE` date belonging to a task. Logseq
// writes these on their own lines directly after the content of the block that
// has the task marker:
//
//	TODO Water the plants
//	SCHEDULED: <2024-01-15 Mon 09:00 .+3d>
//
// The date can carry a time of day and a Repeater, both of which are optional.
type TaskDate struct {
	baseNode
	previousLineAwareImpl

	// Type is whether this is a scheduled date or a deadline.
	Type TaskDateType

	// Date is the day this date refers to. When HasTime is set the time of day
	// is part of the date, otherwise it is midnight and not written out.
	Date time.Time

	// HasTime indicates if the time of day of Date is meaningful.
	HasTime bool

	// Repeater describes how the date repeats, or nil if it does not repeat.
	Repeater *Repeater
}

// NewTaskDate creates a date without a time of day. Any time of day in the
// given time is dropped.
func NewTaskDate(dateType TaskDateType, date time.Time) *TaskDate {
	return &TaskDate{
		Type: dateType,
		Date: truncateToDay(date),
	}
}

// NewTaskDateWithTime creates a date that includes the time of day. Seconds
// and smaller units are dropped, as Logseq stores minute precision.
func NewTaskDateWithTime(dateType TaskDateType, date time.Time) *TaskDate {
	return &TaskDate{
		Type:    dateType,
		Date:    truncateToMinute(date),
		HasTime: true,
	}
}

// NewScheduled creates a `SCHEDULED` date without a time of day.
func NewScheduled(date time.Time) *TaskDate {
	return NewTaskDate(TaskDateTypeScheduled, date)
}

// NewScheduledWithTime creates a `SCHEDULED` date that includes the time of day.
func NewScheduledWithTime(date time.Time) *TaskDate {
	return NewTaskDateWithTime(TaskDateTypeScheduled, date)
}

// NewDeadline creates a `DEADLINE` date without a time of day.
func NewDeadline(date time.Time) *TaskDate {
	return NewTaskDate(TaskDateTypeDeadline, date)
}

// NewDeadlineWithTime creates a `DEADLINE` date that includes the time of day.
func NewDeadlineWithTime(date time.Time) *TaskDate {
	return NewTaskDateWithTime(TaskDateTypeDeadline, date)
}

// WithDate sets the date, dropping any time of day.
func (t *TaskDate) WithDate(date time.Time) *TaskDate {
	t.Date = truncateToDay(date)
	t.HasTime = false
	return t
}

// WithDateAndTime sets the date including its time of day.
func (t *TaskDate) WithDateAndTime(date time.Time) *TaskDate {
	t.Date = truncateToMinute(date)
	t.HasTime = true
	return t
}

// WithRepeater sets the repeater of this date. Passing nil makes the date
// non-repeating.
func (t *TaskDate) WithRepeater(repeater *Repeater) *TaskDate {
	t.Repeater = repeater
	return t
}

func (t *TaskDate) WithPreviousLineType(previousLineType PreviousLineType) *TaskDate {
	t.previousLineType = previousLineType
	return t
}

func (t *TaskDate) debug(p *debugPrinter) {
	p.StartType("TaskDate")
	switch t.Type {
	case TaskDateTypeScheduled:
		p.Field("type", "scheduled")
	case TaskDateTypeDeadline:
		p.Field("type", "deadline")
	}

	if t.HasTime {
		p.Field("date", t.Date.Format("2006-01-02 15:04"))
	} else {
		p.Field("date", t.Date.Format("2006-01-02"))
	}

	if t.Repeater != nil {
		p.Field("repeater", t.Repeater.String())
	}

	debugPreviousLineAware(p, t)
	p.EndType()
}

func (t *TaskDate) isBlock() {}

var _ BlockNode = (*TaskDate)(nil)

func truncateToDay(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
}

func truncateToMinute(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), date.Hour(), date.Minute(), 0, 0, date.Location())
}

// Scheduled gets the `SCHEDULED` date of this block, or nil if it does not
// have one.
func (b *Block) Scheduled() *TaskDate {
	return b.taskDate(TaskDateTypeScheduled)
}

// Deadline gets the `DEADLINE` date of this block, or nil if it does not have
// one.
func (b *Block) Deadline() *TaskDate {
	return b.taskDate(TaskDateTypeDeadline)
}

// SetScheduled sets the `SCHEDULED` date of this block, replacing the one that
// is already there. Passing nil removes the date.
func (b *Block) SetScheduled(date *TaskDate) {
	b.setTaskDate(TaskDateTypeScheduled, date)
}

// SetDeadline sets the `DEADLINE` date of this block, replacing the one that
// is already there. Passing nil removes the date.
func (b *Block) SetDeadline(date *TaskDate) {
	b.setTaskDate(TaskDateTypeDeadline, date)
}

func (b *Block) taskDate(dateType TaskDateType) *TaskDate {
	for node := b.FirstChild(); node != nil; node = node.NextSibling() {
		if date, ok := node.(*TaskDate); ok && date.Type == dateType {
			return date
		}
	}

	return nil
}

func (b *Block) setTaskDate(dateType TaskDateType, date *TaskDate) {
	existing := b.taskDate(dateType)

	if date == nil {
		if existing != nil {
			existing.RemoveSelf()
		}
		return
	}

	date.Type = dateType

	if existing != nil {
		b.ReplaceChild(existing, date)
		return
	}

	// Logseq keeps these dates directly after the content of the task, so
	// place the date after the properties, the first paragraph and any date
	// that is already there.
	var after Node
	seenParagraph := false
	for node := b.FirstChild(); node != nil; node = node.NextSibling() {
		if _, ok := node.(*Paragraph); ok {
			if seenParagraph {
				break
			}
			seenParagraph = true
		} else if _, ok := node.(*Properties); !ok {
			if _, ok := node.(*TaskDate); !ok {
				break
			}
		}

		after = node
	}

	if after == nil {
		b.PrependChild(date)
	} else {
		b.InsertChildAfter(date, after)
	}
}

// Logbook represents a logbook of a task. Logseq will manage these
// automatically when a task changes state. They are used both for tracking if
// a task has been completed, for use with repeating tasks and for time tracking
// if the user has enabled that feature.
//
// These are commonly part of a `Block` with a task marker.
//
// A logbook node can only contain children of type `LogbookEntry`.
type Logbook struct {
	baseNodeWithChildren
	previousLineAwareImpl
}

func NewLogbook(entries ...LogbookEntry) *Logbook {
	l := &Logbook{}
	l.self = l
	l.childValidator = allowOnlyLogbookEntries
	for _, entry := range entries {
		l.AddChild(entry)
	}
	return l
}

func (l *Logbook) WithPreviousLineType(t PreviousLineType) *Logbook {
	l.previousLineType = t
	return l
}

func (l *Logbook) debug(p *debugPrinter) {
	p.StartType("TaskLogbook")
	debugPreviousLineAware(p, l)
	p.Children(l)
	p.EndType()
}

func (l *Logbook) isBlock() {}

var _ BlockNode = (*Logbook)(nil)

// LogbookEntry represents a single entry in a logbook.
type LogbookEntry interface {
	Node
	isLogbookEntry()
}

// LogbookEntryRaw represents a raw logbook entry, this is used for entries that
// are not supported by this library.
type LogbookEntryRaw struct {
	baseNode
	Value string
}

func NewLogbookEntryRaw(value string) *LogbookEntryRaw {
	return &LogbookEntryRaw{
		Value: value,
	}
}

// WithValue sets the value of the logbook entry.
func (t *LogbookEntryRaw) WithValue(value string) *LogbookEntryRaw {
	t.Value = value
	return t
}

func (t *LogbookEntryRaw) debug(p *debugPrinter) {
	p.StartType("LogbookEntryRaw")
	p.Field("value", t.Value)
	p.EndType()
}

func (t *LogbookEntryRaw) isLogbookEntry() {}

func allowOnlyLogbookEntries(n Node) bool {
	_, ok := n.(LogbookEntry)
	return ok
}
