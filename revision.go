package logseq

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// ErrConflict is returned when a transaction is saved and a page it would
// write or check is no longer at the revision the transaction expects, because
// the file was changed or removed after the page was opened.
var ErrConflict = errors.New("page has changed since it was opened")

// revisionOf calculates the revision of a page file from its content. The
// revision is a hash of the content, so two writes that come too quickly for
// the modification time of the file to change still give different revisions.
func revisionOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// fileRevision reads the revision of the file at the given path. A file that
// does not exist has the empty revision, the same as a new page.
func fileRevision(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}

	return revisionOf(data), nil
}

// ownWriteTimeout is how long a write made by the graph is remembered while
// it waits for its change event. The events are debounced for a second, so
// this leaves a large margin for slow indexing.
const ownWriteTimeout = time.Minute

type ownWrite struct {
	revision string
	at       time.Time
}

// ownWrites remembers the revisions that the graph wrote to its files, so
// that the change events for them can be told apart from changes made by
// other programs.
type ownWrites struct {
	mu     sync.Mutex
	writes map[string][]ownWrite
}

// record remembers that the graph is about to write the given revision to a
// path. Removing a file is recorded as the empty revision. Writes are recorded
// before they happen, so that the change event can not arrive first.
func (o *ownWrites) record(path string, revision string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	now := time.Now()
	o.prune(now)

	if o.writes == nil {
		o.writes = make(map[string][]ownWrite)
	}

	o.writes[path] = append(o.writes[path], ownWrite{
		revision: revision,
		at:       now,
	})
}

// consume checks if the given revision of a path was written by the graph.
// Writes that are found are forgotten, together with the writes made before
// them, as a debounced change event can cover more than one write.
func (o *ownWrites) consume(path string, revision string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.prune(time.Now())

	writes := o.writes[path]
	for i, write := range writes {
		if write.revision != revision {
			continue
		}

		if i == len(writes)-1 {
			delete(o.writes, path)
		} else {
			o.writes[path] = writes[i+1:]
		}

		return true
	}

	return false
}

// prune forgets the writes that are older than ownWriteTimeout. A write that
// never gets a change event, such as when nothing watches the graph, is
// removed this way.
func (o *ownWrites) prune(now time.Time) {
	for path, writes := range o.writes {
		i := 0
		for i < len(writes) && now.Sub(writes[i].at) > ownWriteTimeout {
			i++
		}

		if i == len(writes) {
			delete(o.writes, path)
		} else if i > 0 {
			o.writes[path] = writes[i:]
		}
	}
}

// checkRevision makes sure the file at the given path is at the expected
// revision, returning ErrConflict if it is not.
func checkRevision(path string, expected string) error {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return fmt.Errorf("page at %s is a directory", path)
	}

	revision, err := fileRevision(path)
	if err != nil {
		return fmt.Errorf("failed to check if page can be saved at %s: %w", path, err)
	}

	if revision != expected {
		return fmt.Errorf("%w: %s", ErrConflict, path)
	}

	return nil
}
