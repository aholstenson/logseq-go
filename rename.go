package logseq

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aholstenson/logseq-go/internal/indexing"
)

// changeDebounce is how long a file must be left alone before its change is
// handled. It is also the window in which a removed file and a created file
// are paired into a rename.
const changeDebounce = time.Second

// pendingChange is a change to a file that waits to be handled, either for
// its debounce timer to fire or for the change handler to pick it up.
type pendingChange struct {
	path  string
	timer *time.Timer

	// claimed is set when the change was handled together with another
	// change, such as the other half of a rename, so that it is skipped.
	claimed bool
}

// unpairedChange is a removed or created file that was handled without being
// paired into a rename.
type unpairedChange struct {
	revision string
	at       time.Time
}

// pendingChanges keeps track of the changes that wait to be handled, so that
// a change can look at the changes that happened at about the same time.
type pendingChanges struct {
	mu      sync.Mutex
	changes map[string]*pendingChange

	// unpaired is the removed and created files that were handled within the
	// last debounce window. Changes are handled one at a time, so these are
	// needed to tell that a rename has more than one possible match.
	unpaired []unpairedChange
}

func newPendingChanges() *pendingChanges {
	return &pendingChanges{
		changes: make(map[string]*pendingChange),
	}
}

// schedule debounces a change to a path. When no other change to the path
// arrives for the delay, fire is called with the change.
func (p *pendingChanges) schedule(path string, delay time.Duration, fire func(change *pendingChange)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if change, found := p.changes[path]; found && change.timer.Stop() {
		change.timer.Reset(delay)
		return
	}

	// Either there is no change for the path, or its timer already fired and
	// the change is on its way to the handler. A new change makes sure that
	// what happens after that point is also handled.
	change := &pendingChange{path: path}
	change.timer = time.AfterFunc(delay, func() {
		fire(change)
	})
	p.changes[path] = change
}

// take marks a change as being handled. It returns false if the change was
// claimed while it waited, as it then has been handled already.
func (p *pendingChanges) take(change *pendingChange) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if change.claimed {
		return false
	}

	if p.changes[change.path] == change {
		delete(p.changes, change.path)
	}

	return true
}

// paths returns the paths that have changes waiting to be handled.
func (p *pendingChanges) paths() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	paths := make([]string, 0, len(p.changes))
	for path := range p.changes {
		paths = append(paths, path)
	}

	return paths
}

// claim takes over the waiting change of a path, so that it is handled
// together with the change that is handled now.
func (p *pendingChanges) claim(path string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	change, found := p.changes[path]
	if !found {
		return
	}

	change.claimed = true
	change.timer.Stop()
	delete(p.changes, path)
}

// handledUnpaired records that a removed or created file with the given
// revision was handled without being paired into a rename.
func (p *pendingChanges) handledUnpaired(revision string) {
	if revision == "" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	p.pruneUnpaired(now)
	p.unpaired = append(p.unpaired, unpairedChange{
		revision: revision,
		at:       now,
	})
}

// hasUnpaired checks if a removed or created file with the given revision was
// handled without being paired within the last debounce window.
func (p *pendingChanges) hasUnpaired(revision string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pruneUnpaired(time.Now())
	for _, change := range p.unpaired {
		if change.revision == revision {
			return true
		}
	}

	return false
}

func (p *pendingChanges) pruneUnpaired(now time.Time) {
	i := 0
	for i < len(p.unpaired) && now.Sub(p.unpaired[i].at) > changeDebounce {
		i++
	}

	p.unpaired = p.unpaired[i:]
}

// stop stops the timers of all waiting changes.
func (p *pendingChanges) stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, change := range p.changes {
		change.timer.Stop()
	}
}

// claimRenameTarget finds the file that a removed file with the given revision
// was moved to. If exactly one waiting change is for a new file with that
// revision, and no other removed or created file with it was seen in the
// debounce window, the change is claimed and its path is returned. Otherwise an empty string is
// returned, as the removal can not be told apart from a delete.
func (g *Graph) claimRenameTarget(ctx context.Context, pending *pendingChanges, revision string) string {
	removed, created := g.renameCandidates(ctx, pending, revision)
	if len(removed) != 0 || len(created) != 1 || pending.hasUnpaired(revision) {
		return ""
	}

	pending.claim(created[0])
	return created[0]
}

// claimRenameSource finds the removed file that a new file with the given
// revision was moved from. If exactly one waiting change is for a removed file
// with that revision, and no other removed or created file with it was seen in
// the debounce window, the change is claimed and its path is returned. Otherwise an empty string is
// returned, as the new file can not be told apart from a created page.
func (g *Graph) claimRenameSource(ctx context.Context, pending *pendingChanges, revision string) string {
	removed, created := g.renameCandidates(ctx, pending, revision)
	if len(removed) != 1 || len(created) != 0 || pending.hasUnpaired(revision) {
		return ""
	}

	pending.claim(removed[0])
	return removed[0]
}

// renameCandidates finds the files with waiting changes that can be one half
// of a rename of content with the given revision. Removed files are the files
// that no longer exist and had the revision when they were indexed. Created
// files are the files that exist, are not indexed yet and have the revision.
// Only files that are pages of the graph are included.
func (g *Graph) renameCandidates(ctx context.Context, pending *pendingChanges, revision string) (removed []string, created []string) {
	if revision == "" {
		return nil, nil
	}

	for _, path := range pending.paths() {
		if g.createPageDeletedEvent(path) == nil {
			// The file is not a page of the graph.
			continue
		}

		subPath, _ := filepath.Rel(g.directory, path)
		state, err := g.index.GetFileState(ctx, subPath)
		if err != nil {
			continue
		}

		_, err = os.Stat(path)
		if os.IsNotExist(err) {
			if state.Revision == revision {
				removed = append(removed, path)
			}
		} else if err == nil && state == (indexing.FileState{}) {
			if fileRevision, err := fileRevision(path); err == nil && fileRevision == revision {
				created = append(created, path)
			}
		}
	}

	return removed, created
}
