package logseq

import "sync"

// Watcher watches for changes in the graph. Simplifies the process of monitoring
// the graph for changes and reacting to them.
type Watcher struct {
	changes chan ChangeEvent
	closer  func()
	done    chan struct{}

	closeOnce sync.Once
}

// Close stops the watcher. Closing a watcher that is already closed does
// nothing.
func (w *Watcher) Close() error {
	w.closeOnce.Do(func() {
		close(w.done)
		w.closer()
	})

	return nil
}

// Events returns the channel that changes to the graph are sent to.
//
// The channel is not closed when the watcher is closed, as the graph may be in
// the middle of sending an event to it. Receive from Done to tell that the
// watcher has been closed:
//
//	for {
//		select {
//		case event := <-watcher.Events():
//			// Handle the event
//		case <-watcher.Done():
//			return
//		}
//	}
func (w *Watcher) Events() <-chan ChangeEvent {
	return w.changes
}

// Done returns a channel that is closed when the watcher is closed.
func (w *Watcher) Done() <-chan struct{} {
	return w.done
}
