package logseq

// Watcher watches for changes in the graph. Simplifies the process of monitoring
// the graph for changes and reacting to them.
type Watcher struct {
	changes chan ChangeEvent
	closer  func()
	done    chan struct{}
}

func (w *Watcher) Close() error {
	close(w.done)
	w.closer()
	return nil
}

func (w *Watcher) Events() <-chan ChangeEvent {
	return w.changes
}

// Done returns a channel that is closed when the watcher is closed.
func (w *Watcher) Done() <-chan struct{} {
	return w.done
}
