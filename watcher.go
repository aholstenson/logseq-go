package logseq

// Watcher watches for changes in the graph. Simplifies the process of monitoring
// the graph for changes and reacting to them.
type Watcher struct {
	changes chan ChangeEvent
	closer  func()
}

func (w *Watcher) Close() error {
	w.closer()
	close(w.changes)
	return nil
}

func (w *Watcher) Events() <-chan ChangeEvent {
	return w.changes
}
