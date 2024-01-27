package content

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
