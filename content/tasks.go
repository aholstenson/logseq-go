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

// String returns the marker Logseq writes for this status, or an empty string
// for TaskStatusNone and statuses this library does not know.
func (t TaskStatus) String() string {
	switch t {
	case TaskStatusTodo:
		return "TODO"
	case TaskStatusDoing:
		return "DOING"
	case TaskStatusDone:
		return "DONE"
	case TaskStatusLater:
		return "LATER"
	case TaskStatusNow:
		return "NOW"
	case TaskStatusCancelled:
		return "CANCELLED"
	case TaskStatusCanceled:
		return "CANCELED"
	case TaskStatusInProgress:
		return "IN-PROGRESS"
	case TaskStatusWait:
		return "WAIT"
	case TaskStatusWaiting:
		return "WAITING"
	}

	return ""
}

// IsOpen returns true if the status is a task that is not finished, such as
// TODO, DOING or WAITING. Returns false for TaskStatusNone.
func (t TaskStatus) IsOpen() bool {
	return t != TaskStatusNone && !t.IsClosed()
}

// IsClosed returns true if the status is a task that is finished, either as
// DONE or as cancelled.
func (t TaskStatus) IsClosed() bool {
	switch t {
	case TaskStatusDone, TaskStatusCancelled, TaskStatusCanceled:
		return true
	}

	return false
}

// TaskMarker is the status of a task, such as `TODO` or `DONE`. Logseq only
// reads it as a task when it is the first thing in the first paragraph of a
// block. Use Block.TaskStatus and Block.SetTaskStatus to work with it there.
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
	case TaskStatusCanceled:
		p.Field("type", "canceled")
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

// TaskStatus gets the status of the task in this block, or TaskStatusNone if
// the block is not a task.
func (b *Block) TaskStatus() TaskStatus {
	if marker := b.taskMarker(); marker != nil {
		return marker.Status
	}

	return TaskStatusNone
}

// IsTask returns true if this block has a task marker.
func (b *Block) IsTask() bool {
	return b.TaskStatus() != TaskStatusNone
}

// SetTaskStatus sets the status of the task in this block, replacing the
// marker that is already there. Passing TaskStatusNone removes the marker, but
// keeps the priority and the dates of the block.
//
// The marker goes at the start of the first paragraph of the block. If the
// block does not have a paragraph, one is added.
func (b *Block) SetTaskStatus(status TaskStatus) {
	existing := b.taskMarker()

	if status == TaskStatusNone {
		if existing != nil {
			existing.RemoveSelf()
		}
		return
	}

	if existing != nil {
		existing.Status = status
		return
	}

	b.ensureTaskParagraph().PrependChild(NewTaskMarker(status))
}

// Priority gets the priority of the task in this block, or PriorityNone if it
// does not have one. A block can have a priority without a task marker.
func (b *Block) Priority() Priority {
	if priority := b.taskPriority(); priority != nil {
		return priority.Priority
	}

	return PriorityNone
}

// SetPriority sets the priority of this block, replacing the priority that is
// already there. Passing PriorityNone removes the priority.
//
// The priority goes after the task marker, or at the start of the first
// paragraph of the block if it does not have a marker.
func (b *Block) SetPriority(priority Priority) {
	existing := b.taskPriority()

	if priority == PriorityNone {
		if existing != nil {
			existing.RemoveSelf()
		}
		return
	}

	if existing != nil {
		existing.Priority = priority
		return
	}

	node := NewTaskPriority(priority)
	paragraph := b.ensureTaskParagraph()
	if marker, ok := paragraph.FirstChild().(*TaskMarker); ok {
		paragraph.InsertChildAfter(node, marker)
		return
	}

	paragraph.PrependChild(node)
}

// taskParagraph gets the paragraph that holds the marker and priority of a
// task, which is the first paragraph of the block. Properties before it are
// skipped, but any other node means that the block does not start with a
// paragraph.
func (b *Block) taskParagraph() *Paragraph {
	for node := b.FirstChild(); node != nil; node = node.NextSibling() {
		switch n := node.(type) {
		case *Paragraph:
			return n
		case *Properties:
			continue
		default:
			return nil
		}
	}

	return nil
}

// ensureTaskParagraph gets the paragraph that holds the marker and priority of
// a task, adding an empty one at the start of the block if there is none.
func (b *Block) ensureTaskParagraph() *Paragraph {
	if paragraph := b.taskParagraph(); paragraph != nil {
		return paragraph
	}

	paragraph := NewParagraph()
	b.PrependChild(paragraph)
	return paragraph
}

func (b *Block) taskMarker() *TaskMarker {
	paragraph := b.taskParagraph()
	if paragraph == nil {
		return nil
	}

	marker, _ := paragraph.FirstChild().(*TaskMarker)
	return marker
}

func (b *Block) taskPriority() *TaskPriority {
	paragraph := b.taskParagraph()
	if paragraph == nil {
		return nil
	}

	node := paragraph.FirstChild()
	if _, ok := node.(*TaskMarker); ok {
		node = node.NextSibling()
	}

	priority, _ := node.(*TaskPriority)
	return priority
}

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
// when the task is completed. The rules are the ones Logseq uses, which are
// not the same as the rules of Org mode. See TaskDate.Next.
type RepeaterType int

const (
	// RepeaterTypeCumulate, written as `+`, moves the date forward one
	// interval, even if that leaves it in the past.
	RepeaterTypeCumulate RepeaterType = iota
	// RepeaterTypeCatchUp, written as `++`, keeps a date that is in the
	// future and moves a date that is not forward one interval, even if that
	// leaves it in the past.
	RepeaterTypeCatchUp
	// RepeaterTypeRestart, written as `.+`, moves the date forward whole
	// intervals until it is in the future.
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

// Next returns the date that this date moves to when its task is completed at
// the given time, following the rules of its repeater. Returns Date if the date
// does not repeat.
//
// The intervals are added to the time of day as it is on a clock, so a day is
// always 24 hours on the clock even when daylight saving time changes. Adding
// months or years to a day that the target month does not have gives the last
// day of that month, so 31 January plus one month is the end of February.
func (t *TaskDate) Next(at time.Time) time.Time {
	r := t.Repeater
	if r == nil || r.Value <= 0 {
		return t.Date
	}

	var next time.Time
	switch r.Type {
	case RepeaterTypeCatchUp:
		if t.Date.After(at) {
			return t.Date
		}

		next = r.addTo(t.Date)
	case RepeaterTypeRestart:
		next = r.addTo(t.Date)
		for !next.After(at) {
			next = r.addTo(next)
		}
	default:
		next = r.addTo(t.Date)
	}

	if !t.HasTime {
		next = truncateToDay(next)
	}

	return next
}

// addTo adds one interval of the repeater to a time.
func (r *Repeater) addTo(date time.Time) time.Time {
	switch r.Unit {
	case RepeaterUnitHour:
		return time.Date(date.Year(), date.Month(), date.Day(), date.Hour()+r.Value, date.Minute(), date.Second(), date.Nanosecond(), date.Location())
	case RepeaterUnitDay:
		return date.AddDate(0, 0, r.Value)
	case RepeaterUnitWeek:
		return date.AddDate(0, 0, 7*r.Value)
	case RepeaterUnitMonth:
		return addMonths(date, r.Value)
	case RepeaterUnitYear:
		return addMonths(date, 12*r.Value)
	}

	return date
}

// addMonths adds months to a time, using the last day of the target month if
// it does not have the day of the time.
func addMonths(date time.Time, months int) time.Time {
	first := time.Date(date.Year(), date.Month()+time.Month(months), 1, date.Hour(), date.Minute(), date.Second(), date.Nanosecond(), date.Location())
	lastDay := first.AddDate(0, 1, -1).Day()

	day := date.Day()
	if day > lastDay {
		day = lastDay
	}

	return first.AddDate(0, 0, day-1)
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

// CompleteTask marks the task in this block as done at the given time, in the
// same way as Logseq does when a task is checked:
//
//   - A clock in the logbook that is still running is stopped.
//   - A task that does not repeat gets the status DONE.
//   - A task with a repeating `SCHEDULED` or `DEADLINE` date stays open, so
//     that it can be done again. Each repeating date moves to its next
//     occurrence, see TaskDate.Next. DOING changes to TODO and NOW changes to
//     LATER, other statuses are kept. A state change from the old status to
//     DONE is added to the logbook, which is created if the block does not
//     have one.
//
// Nothing changes if the block is not a task or if the task is closed.
func (b *Block) CompleteTask(at time.Time) {
	status := b.TaskStatus()
	if !status.IsOpen() {
		return
	}

	logbook := b.Logbook()
	if logbook != nil {
		if clock, ok := logbook.LastChild().(*LogbookEntryClock); ok && clock.IsRunning() {
			clock.WithEnd(at)
		}
	}

	repeated := false
	for _, date := range []*TaskDate{b.Scheduled(), b.Deadline()} {
		if date == nil || date.Repeater == nil {
			continue
		}

		date.Date = date.Next(at)
		repeated = true
	}

	if !repeated {
		b.SetTaskStatus(TaskStatusDone)
		return
	}

	switch status {
	case TaskStatusDoing:
		b.SetTaskStatus(TaskStatusTodo)
	case TaskStatusNow:
		b.SetTaskStatus(TaskStatusLater)
	}

	if logbook == nil {
		// Logseq writes the logbook directly after the line before it.
		logbook = NewLogbook().WithPreviousLineType(PreviousLineTypeNonBlank)
		b.insertAfterTaskContent(logbook)
	}

	logbook.AddChild(NewLogbookEntryStateChange(status, TaskStatusDone, at))
}

// Logbook gets the logbook of this block, or nil if it does not have one. Only
// a logbook before the sub blocks belongs to the block.
func (b *Block) Logbook() *Logbook {
	for node := b.FirstChild(); node != nil; node = node.NextSibling() {
		switch n := node.(type) {
		case *Logbook:
			return n
		case *Block:
			return nil
		}
	}

	return nil
}

// insertAfterTaskContent inserts a node after the first paragraph of the
// block and the dates and properties that follow it, which is where Logseq
// puts the logbook of a task.
func (b *Block) insertAfterTaskContent(node Node) {
	var after Node
	seenParagraph := false
loop:
	for child := b.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.(type) {
		case *Paragraph:
			if seenParagraph {
				break loop
			}
			seenParagraph = true
		case *TaskDate, *Properties:
		default:
			break loop
		}

		after = child
	}

	if after == nil {
		b.PrependChild(node)
	} else {
		b.InsertChildAfter(node, after)
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

// LogbookEntryClock is a time tracking entry, written by Logseq when the clock
// on a task is started and stopped:
//
//	CLOCK: [2023-06-26 Mon 17:25:56]--[2023-06-26 Mon 17:25:58] =>  00:00:02
//
// The duration at the end is derived from Start and End when the entry is
// written, so it can never drift from the times it sums up.
//
// The times are kept with second precision no matter how they were written. A
// graph where `:logbook/settings :with-second-support?` is off writes them,
// and the duration, without the seconds.
type LogbookEntryClock struct {
	baseNode

	// Start is when the clock was started.
	Start time.Time

	// End is when the clock was stopped. The zero time means the clock is
	// still running, which Logseq writes without the end and the duration.
	End time.Time
}

// NewLogbookEntryClock creates a clock entry that ran between two times. Pass
// the zero time as the end for a clock that is still running.
func NewLogbookEntryClock(start time.Time, end time.Time) *LogbookEntryClock {
	return &LogbookEntryClock{
		Start: truncateToSecond(start),
		End:   truncateToSecond(end),
	}
}

// WithStart sets when the clock was started.
func (t *LogbookEntryClock) WithStart(start time.Time) *LogbookEntryClock {
	t.Start = truncateToSecond(start)
	return t
}

// WithEnd sets when the clock was stopped. Passing the zero time makes the
// clock running again.
func (t *LogbookEntryClock) WithEnd(end time.Time) *LogbookEntryClock {
	t.End = truncateToSecond(end)
	return t
}

// IsRunning returns true if the clock has not been stopped.
func (t *LogbookEntryClock) IsRunning() bool {
	return t.End.IsZero()
}

// Duration is how long the clock ran, which is zero while it is running.
func (t *LogbookEntryClock) Duration() time.Duration {
	if t.IsRunning() {
		return 0
	}

	return t.End.Sub(t.Start)
}

func (t *LogbookEntryClock) debug(p *debugPrinter) {
	p.StartType("LogbookEntryClock")
	p.Field("start", t.Start.Format("2006-01-02 15:04:05"))
	if !t.IsRunning() {
		p.Field("end", t.End.Format("2006-01-02 15:04:05"))
	}
	p.EndType()
}

func (t *LogbookEntryClock) isLogbookEntry() {}

// LogbookEntryStateChange records that a task changed status, which Logseq
// writes when a repeating task is completed. The entry is written as
// `* State "DONE" from "TODO" [2023-06-26 Mon 17:25:56]`.
type LogbookEntryStateChange struct {
	baseNode

	// From is the status the task had. TaskStatusNone leaves the `from` part
	// out of the entry.
	From TaskStatus

	// To is the status the task changed to.
	To TaskStatus

	// Time is when the change happened.
	Time time.Time
}

func NewLogbookEntryStateChange(from TaskStatus, to TaskStatus, at time.Time) *LogbookEntryStateChange {
	return &LogbookEntryStateChange{
		From: from,
		To:   to,
		Time: truncateToSecond(at),
	}
}

// WithFrom sets the status the task had.
func (t *LogbookEntryStateChange) WithFrom(from TaskStatus) *LogbookEntryStateChange {
	t.From = from
	return t
}

// WithTo sets the status the task changed to.
func (t *LogbookEntryStateChange) WithTo(to TaskStatus) *LogbookEntryStateChange {
	t.To = to
	return t
}

// WithTime sets when the change happened.
func (t *LogbookEntryStateChange) WithTime(at time.Time) *LogbookEntryStateChange {
	t.Time = truncateToSecond(at)
	return t
}

func (t *LogbookEntryStateChange) debug(p *debugPrinter) {
	p.StartType("LogbookEntryStateChange")
	if t.From != TaskStatusNone {
		p.Field("from", t.From.String())
	}
	p.Field("to", t.To.String())
	p.Field("time", t.Time.Format("2006-01-02 15:04:05"))
	p.EndType()
}

func (t *LogbookEntryStateChange) isLogbookEntry() {}

func truncateToSecond(at time.Time) time.Time {
	if at.IsZero() {
		return at
	}

	return time.Date(at.Year(), at.Month(), at.Day(), at.Hour(), at.Minute(), at.Second(), 0, at.Location())
}

func allowOnlyLogbookEntries(n Node) bool {
	_, ok := n.(LogbookEntry)
	return ok
}
