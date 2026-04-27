package config

import (
	"errors"
	"sync"
	"testing"
	"time"

	coreio "dappco.re/go/io"
	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
)

type fakeWatchBackend struct {
	mu     sync.Mutex
	events chan fsnotify.Event
	errors chan error
	addErr error
	adds   []string
	closed bool
}

func newFakeWatchBackend() *fakeWatchBackend {
	return &fakeWatchBackend{
		events: make(chan fsnotify.Event, 8),
		errors: make(chan error, 1),
	}
}

func (w *fakeWatchBackend) Add(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.adds = append(w.adds, path)
	return w.addErr
}

func (w *fakeWatchBackend) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return nil
}

func (w *fakeWatchBackend) Events() <-chan fsnotify.Event {
	return w.events
}

func (w *fakeWatchBackend) Errors() <-chan error {
	return w.errors
}

func (w *fakeWatchBackend) emit(event fsnotify.Event) {
	w.events <- event
}

func (w *fakeWatchBackend) addCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.adds)
}

func useFakeWatchBackend(t *testing.T, backend *fakeWatchBackend) {
	t.Helper()
	previous := newWatchBackend
	newWatchBackend = func() (watchBackend, error) {
		return backend, nil
	}
	t.Cleanup(func() {
		newWatchBackend = previous
	})
}

func TestWatch_Watch_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	path := "watch/config.yaml"
	assert.NoError(t, m.Write(path, "key: one\n"))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)

	cfg, err := New(WithMedium(m), WithPath(path))
	assert.NoError(t, err)

	var mu sync.Mutex
	fired := 0
	cfg.OnChange(func(_ string, _ any) {
		mu.Lock()
		fired++
		mu.Unlock()
	})

	assert.NoError(t, cfg.Watch())
	t.Cleanup(cfg.StopWatch)

	assert.NoError(t, m.Write(path, "key: two\n"))
	backend.emit(fsnotify.Event{Name: path, Op: fsnotify.Write})
	waitFor(t, &mu, func() int { return fired }, 1)

	mu.Lock()
	defer mu.Unlock()
	assert.Greater(t, fired, 0)
}

func TestWatch_Watch_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	path := "watch/missing.yaml"
	backend := newFakeWatchBackend()
	backend.addErr = errors.New("missing")
	useFakeWatchBackend(t, backend)

	cfg, err := New(WithMedium(m), WithPath(path))
	assert.NoError(t, err)
	// Watching a non-existent path should return an error rather than crashing.
	err = cfg.Watch()
	assert.Error(t, err)
}

func TestWatch_Watch_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	path := "watch/idempotent.yaml"
	assert.NoError(t, m.Write(path, "key: value\n"))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)

	cfg, err := New(WithMedium(m), WithPath(path))
	assert.NoError(t, err)

	// Double Watch is idempotent — no duplicate watchers, no panic.
	assert.NoError(t, cfg.Watch())
	assert.NoError(t, cfg.Watch())
	cfg.StopWatch()
	cfg.StopWatch()
}

func TestWatch_ReloadKeys_Good(t *testing.T) {
	// When a file is reloaded via the watcher, OnChange must fire once per
	// changed key with the new value — not a single empty-key signal.
	m := coreio.NewMockMedium()
	path := "watch/reload.yaml"
	assert.NoError(t, m.Write(path, "dev:\n  editor: vim\napp:\n  name: alpha\n"))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)

	cfg, err := New(WithMedium(m), WithPath(path))
	assert.NoError(t, err)

	var mu sync.Mutex
	seen := map[string]any{}
	cfg.OnChange(func(key string, value any) {
		mu.Lock()
		defer mu.Unlock()
		seen[key] = value
	})

	assert.NoError(t, cfg.Watch())
	t.Cleanup(cfg.StopWatch)

	// Change editor and name, plus add a new key.
	assert.NoError(t, m.Write(path, "dev:\n  editor: nano\napp:\n  name: beta\n  version: \"1\"\n"))
	backend.emit(fsnotify.Event{Name: path, Op: fsnotify.Write})
	waitFor(t, &mu, func() int { return len(seen) }, 3)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "nano", seen["dev.editor"])
	assert.Equal(t, "beta", seen["app.name"])
	assert.Equal(t, "1", seen["app.version"])
}

func TestWatch_AtomicSave_Good(t *testing.T) {
	// Atomic-save editors (vim, VSCode, most IDE auto-formatters) replace a
	// file via rename: write new inode, rename over the old path, unlink the
	// original. fsnotify tracks the original inode and silently stops firing
	// after the first rename — the watcher re-Adds the path to survive this.
	m := coreio.NewMockMedium()
	path := "watch/atomic.yaml"
	assert.NoError(t, m.Write(path, "key: first\n"))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)

	cfg, err := New(WithMedium(m), WithPath(path))
	assert.NoError(t, err)

	var mu sync.Mutex
	fires := 0
	cfg.OnChange(func(_ string, _ any) {
		mu.Lock()
		fires++
		mu.Unlock()
	})

	assert.NoError(t, cfg.Watch())
	t.Cleanup(cfg.StopWatch)

	// Simulate an atomic save: write to sidecar, rename over target.
	sidecar := "watch/atomic.yaml.swp"
	assert.NoError(t, m.Write(sidecar, "key: second\n"))
	assert.NoError(t, m.Rename(sidecar, path))
	backend.emit(fsnotify.Event{Name: path, Op: fsnotify.Rename})

	// Wait for the first rename-driven reload to land.
	waitFor(t, &mu, func() int { return fires }, 1)

	// Second atomic save: watcher must still be live.
	sidecar2 := "watch/atomic.yaml.swp2"
	assert.NoError(t, m.Write(sidecar2, "key: third\n"))
	assert.NoError(t, m.Rename(sidecar2, path))
	backend.emit(fsnotify.Event{Name: path, Op: fsnotify.Rename})

	waitFor(t, &mu, func() int { return fires }, 2)

	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, fires, 2, "watcher must survive the second atomic save")
	assert.GreaterOrEqual(t, backend.addCount(), 3, "initial watch plus two re-add attempts")
}

// waitFor polls the provided getter until it reaches target or 2s elapse.
// Used by watch tests where fsnotify latency is platform-dependent.
func waitFor(t *testing.T, mu *sync.Mutex, get func() int, target int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := get()
		mu.Unlock()
		if got >= target {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	mu.Lock()
	got := get()
	mu.Unlock()
	t.Fatalf("timed out waiting for %d events; got %d", target, got)
}

func TestWatch_DiffSnapshots_Good(t *testing.T) {
	// diffSnapshots is the core of reload notifications — feed it the two
	// snapshots a watcher would produce and verify the per-key changes.
	before := map[string]any{
		"dev.editor": "vim",
		"app.name":   "alpha",
		"gone":       true,
	}
	after := map[string]any{
		"dev.editor": "nano",    // changed
		"app.name":   "alpha",   // unchanged
		"app.new":    "arrived", // added
	}

	changes := diffSnapshots(before, after)
	// Sorted lexically: app.new, dev.editor, gone
	assert.Len(t, changes, 3)
	assert.Equal(t, "app.new", changes[0].Key)
	assert.Equal(t, "arrived", changes[0].Value)
	assert.Equal(t, "dev.editor", changes[1].Key)
	assert.Equal(t, "nano", changes[1].Value)
	assert.Equal(t, "vim", changes[1].Previous)
	assert.Equal(t, "gone", changes[2].Key)
	assert.Nil(t, changes[2].Value)
	assert.Equal(t, true, changes[2].Previous)
}

func TestWatch_DiffSnapshots_Ugly(t *testing.T) {
	// Nested map values should compare structurally, not by pointer identity.
	nested := map[string]any{"features": map[string]any{"dark-mode": true}}
	same := map[string]any{"features": map[string]any{"dark-mode": true}}

	changes := diffSnapshots(nested, same)
	assert.Empty(t, changes)
}
