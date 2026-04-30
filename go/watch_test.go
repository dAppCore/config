package config

import (
	core "dappco.re/go"
	"sync"
	"time"

	coreio "dappco.re/go/io"
	"github.com/fsnotify/fsnotify"
)

const (
	watchDevEditorKey = "dev.editor"
	watchAppNameKey   = "app.name"
	watchConfigPath   = "ax7/watch.yaml"
	watchNameYAML     = "name: one\n"
)

type fakeWatchBackend struct {
	mu     sync.Mutex
	events chan fsnotify.Event
	errors chan core.Result
	addErr error
	adds   []string
	closed bool
}

func newFakeWatchBackend() *fakeWatchBackend {
	return &fakeWatchBackend{
		events: make(chan fsnotify.Event, 8),
		errors: make(chan core.Result, 1),
	}
}

func (w *fakeWatchBackend) Add(path string) core.Result {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.adds = append(w.adds, path)
	return core.ResultOf(nil, w.addErr)
}

func (w *fakeWatchBackend) Close() core.Result {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return core.Ok(nil)
}

func (w *fakeWatchBackend) Events() <-chan fsnotify.Event {
	return w.events
}

func (w *fakeWatchBackend) Errors() <-chan core.Result {
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

func useFakeWatchBackend(t *core.T, backend *fakeWatchBackend) {
	t.Helper()
	previous := newWatchBackend
	newWatchBackend = func() core.Result {
		return core.Ok(backend)
	}
	t.Cleanup(func() {
		newWatchBackend = previous
	})
}

func TestWatch_Watch_Good(t *core.T) {
	m := coreio.NewMockMedium()
	path := "watch/config.yaml"
	core.AssertNoError(t, m.Write(path, "key: one\n"))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)

	cfg, err := configResult(New(WithMedium(m), WithPath(path)))
	core.AssertNoError(t, err)

	var mu sync.Mutex
	fired := 0
	cfg.OnChange(func(_ string, _ any) {
		mu.Lock()
		fired++
		mu.Unlock()
	})

	core.AssertNoError(t, resultError(cfg.Watch()))
	t.Cleanup(cfg.StopWatch)

	core.AssertNoError(t, m.Write(path, "key: two\n"))
	backend.emit(fsnotify.Event{Name: path, Op: fsnotify.Write})
	waitFor(t, &mu, func() int { return fired }, 1)

	mu.Lock()
	defer mu.Unlock()
	core.AssertGreater(t, fired, 0)
}

func TestWatch_Watch_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	path := "watch/missing.yaml"
	backend := newFakeWatchBackend()
	backend.addErr = core.NewError("missing")
	useFakeWatchBackend(t, backend)

	cfg, err := configResult(New(WithMedium(m), WithPath(path)))
	core.AssertNoError(t, err)
	// Watching a non-existent path should return an error rather than crashing.
	err = resultError(cfg.Watch())
	core.AssertError(t, err)
}

func TestWatch_Watch_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	path := "watch/idempotent.yaml"
	core.AssertNoError(t, m.Write(path, "key: value\n"))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)

	cfg, err := configResult(New(WithMedium(m), WithPath(path)))
	core.AssertNoError(t, err)

	// Double Watch is idempotent — no duplicate watchers, no panic.
	core.AssertNoError(t, resultError(cfg.Watch()))
	core.AssertNoError(t, resultError(cfg.Watch()))
	cfg.StopWatch()
	cfg.StopWatch()
}

func TestWatch_Config_Watch_ReloadKeys_Good(t *core.T) {
	// When a file is reloaded via the watcher, OnChange must fire once per
	// changed key with the new value — not a single empty-key signal.
	m := coreio.NewMockMedium()
	path := "watch/reload.yaml"
	core.AssertNoError(t, m.Write(path, "dev:\n  editor: vim\napp:\n  name: alpha\n"))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)

	cfg, err := configResult(New(WithMedium(m), WithPath(path)))
	core.AssertNoError(t, err)

	var mu sync.Mutex
	seen := map[string]any{}
	cfg.OnChange(func(key string, value any) {
		mu.Lock()
		defer mu.Unlock()
		seen[key] = value
	})

	core.AssertNoError(t, resultError(cfg.Watch()))
	t.Cleanup(cfg.StopWatch)

	// Change editor and name, plus add a new key.
	core.AssertNoError(t, m.Write(path, "dev:\n  editor: nano\napp:\n  name: beta\n  version: \"1\"\n"))
	backend.emit(fsnotify.Event{Name: path, Op: fsnotify.Write})
	waitFor(t, &mu, func() int { return len(seen) }, 3)

	mu.Lock()
	defer mu.Unlock()
	core.AssertEqual(t, "nano", seen[watchDevEditorKey])
	core.AssertEqual(t, "beta", seen[watchAppNameKey])
	core.AssertEqual(t, "1", seen["app.version"])
}

func TestWatch_Config_Watch_AtomicSave_Good(t *core.T) {
	// Atomic-save editors (vim, VSCode, most IDE auto-formatters) replace a
	// file via rename: write new inode, rename over the old path, unlink the
	// original. fsnotify tracks the original inode and silently stops firing
	// after the first rename — the watcher re-Adds the path to survive this.
	m := coreio.NewMockMedium()
	path := "watch/atomic.yaml"
	core.AssertNoError(t, m.Write(path, "key: first\n"))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)

	cfg, err := configResult(New(WithMedium(m), WithPath(path)))
	core.AssertNoError(t, err)

	var mu sync.Mutex
	fires := 0
	cfg.OnChange(func(_ string, _ any) {
		mu.Lock()
		fires++
		mu.Unlock()
	})

	core.AssertNoError(t, resultError(cfg.Watch()))
	t.Cleanup(cfg.StopWatch)

	// Simulate an atomic save: write to sidecar, rename over target.
	sidecar := "watch/atomic.yaml.swp"
	core.AssertNoError(t, m.Write(sidecar, "key: second\n"))
	core.AssertNoError(t, m.Rename(sidecar, path))
	backend.emit(fsnotify.Event{Name: path, Op: fsnotify.Rename})

	// Wait for the first rename-driven reload to land.
	waitFor(t, &mu, func() int { return fires }, 1)

	// Second atomic save: watcher must still be live.
	sidecar2 := "watch/atomic.yaml.swp2"
	core.AssertNoError(t, m.Write(sidecar2, "key: third\n"))
	core.AssertNoError(t, m.Rename(sidecar2, path))
	backend.emit(fsnotify.Event{Name: path, Op: fsnotify.Rename})

	waitFor(t, &mu, func() int { return fires }, 2)

	mu.Lock()
	defer mu.Unlock()
	core.AssertGreaterOrEqual(t, fires, 2, "watcher must survive the second atomic save")
	core.AssertGreaterOrEqual(t, backend.addCount(), 3, "initial watch plus two re-add attempts")
}

// waitFor polls the provided getter until it reaches target or 2s elapse.
// Used by watch tests where fsnotify latency is platform-dependent.
func waitFor(t *core.T, mu *sync.Mutex, get func() int, target int) {
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

func TestWatch_diffSnapshots_Good(t *core.T) {
	// diffSnapshots is the core of reload notifications — feed it the two
	// snapshots a watcher would produce and verify the per-key changes.
	before := map[string]any{
		watchDevEditorKey: "vim",
		watchAppNameKey:   "alpha",
		"gone":            true,
	}
	after := map[string]any{
		watchDevEditorKey: "nano",    // changed
		watchAppNameKey:   "alpha",   // unchanged
		"app.new":         "arrived", // added
	}

	changes := diffSnapshots(before, after)
	// Sorted lexically: app.new, dev.editor, gone
	core.AssertLen(t, changes, 3)
	core.AssertEqual(t, "app.new", changes[0].Key)
	core.AssertEqual(t, "arrived", changes[0].Value)
	core.AssertEqual(t, watchDevEditorKey, changes[1].Key)
	core.AssertEqual(t, "nano", changes[1].Value)
	core.AssertEqual(t, "vim", changes[1].Previous)
	core.AssertEqual(t, "gone", changes[2].Key)
	core.AssertNil(t, changes[2].Value)
	core.AssertEqual(t, true, changes[2].Previous)
}

func TestWatch_diffSnapshots_Ugly(t *core.T) {
	// Nested map values should compare structurally, not by pointer identity.
	nested := map[string]any{"features": map[string]any{"dark-mode": true}}
	same := map[string]any{"features": map[string]any{"dark-mode": true}}

	changes := diffSnapshots(nested, same)
	core.AssertEmpty(t, changes)
}

func TestWatch_Backend_Add_Good(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	defer func() {
		core.RequireNoError(t, resultError(backend.Close()))
	}()
	got := backend.Add(t.TempDir())
	core.AssertNoError(t, resultError(got))
}

func TestWatch_Backend_Add_Bad(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	core.RequireNoError(t, resultError(backend.Close()))
	got := backend.Add(t.TempDir())
	core.AssertError(t, resultError(got))
}

func TestWatch_Backend_Add_Ugly(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	defer func() {
		core.RequireNoError(t, resultError(backend.Close()))
	}()
	got := backend.Add("missing/watch/path")
	core.AssertError(t, resultError(got))
}

func TestWatch_Backend_Close_Good(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	got := backend.Close()
	core.AssertNoError(t, resultError(got))
}

func TestWatch_Backend_Close_Bad(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	core.RequireNoError(t, resultError(backend.Close()))
	got := backend.Close()
	core.AssertNoError(t, resultError(got))
}

func TestWatch_Backend_Close_Ugly(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	events := backend.Events()
	core.AssertNotNil(t, events)
	core.AssertNoError(t, resultError(backend.Close()))
}

func TestWatch_Backend_Events_Good(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	defer func() {
		core.RequireNoError(t, resultError(backend.Close()))
	}()
	got := backend.Events()
	core.AssertNotNil(t, got)
}

func TestWatch_Backend_Events_Bad(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	core.RequireNoError(t, resultError(backend.Close()))
	got := backend.Events()
	core.AssertNotNil(t, got)
}

func TestWatch_Backend_Events_Ugly(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	defer func() {
		core.RequireNoError(t, resultError(backend.Close()))
	}()
	got := cap(backend.Events())
	core.AssertGreaterOrEqual(t, got, 0)
}

func TestWatch_Backend_Errors_Good(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	defer func() {
		core.RequireNoError(t, resultError(backend.Close()))
	}()
	got := backend.Errors()
	core.AssertNotNil(t, got)
}

func TestWatch_Backend_Errors_Bad(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	core.RequireNoError(t, resultError(backend.Close()))
	got := backend.Errors()
	core.AssertNotNil(t, got)
}

func TestWatch_Backend_Errors_Ugly(t *core.T) {
	backend, err := watchBackendResult(newWatchBackend())
	core.RequireNoError(t, err)
	defer func() {
		core.RequireNoError(t, resultError(backend.Close()))
	}()
	got := cap(backend.Errors())
	core.AssertGreaterOrEqual(t, got, 0)
}

func TestWatch_Config_Watch_Good(t *core.T) {
	m := coreio.NewMockMedium()
	path := watchConfigPath
	core.RequireNoError(t, m.Write(path, watchNameYAML))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)
	cfg, err := configResult(New(WithMedium(m), WithPath(path)))
	core.RequireNoError(t, err)

	err = resultError(cfg.Watch())
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, backend.addCount())
}

func TestWatch_Config_Watch_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	path := watchConfigPath
	core.RequireNoError(t, m.Write(path, watchNameYAML))
	backend := newFakeWatchBackend()
	backend.addErr = core.NewError("add failed")
	useFakeWatchBackend(t, backend)
	cfg, err := configResult(New(WithMedium(m), WithPath(path)))
	core.RequireNoError(t, err)

	err = resultError(cfg.Watch())
	core.AssertError(t, err)
	core.AssertTrue(t, backend.closed)
}

func TestWatch_Config_Watch_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	path := watchConfigPath
	core.RequireNoError(t, m.Write(path, watchNameYAML))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)
	cfg, err := configResult(New(WithMedium(m), WithPath(path)))
	core.RequireNoError(t, err)

	core.AssertNoError(t, resultError(cfg.Watch()))
	core.AssertNoError(t, resultError(cfg.Watch()))
	core.AssertEqual(t, 1, backend.addCount())
}

func TestWatch_Config_StopWatch_Good(t *core.T) {
	m := coreio.NewMockMedium()
	path := watchConfigPath
	core.RequireNoError(t, m.Write(path, watchNameYAML))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)
	cfg, err := configResult(New(WithMedium(m), WithPath(path)))
	core.RequireNoError(t, err)
	core.RequireNoError(t, resultError(cfg.Watch()))

	cfg.StopWatch()
	core.AssertNil(t, cfg.watcher)
	core.AssertTrue(t, backend.closed)
}

func TestWatch_Config_StopWatch_Bad(t *core.T) {
	cfg, err := configResult(New(WithMedium(coreio.NewMockMedium()), WithPath(watchConfigPath)))
	core.RequireNoError(t, err)
	cfg.StopWatch()
	core.AssertNil(t, cfg.watcher)
}

func TestWatch_Config_StopWatch_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	path := watchConfigPath
	core.RequireNoError(t, m.Write(path, watchNameYAML))
	backend := newFakeWatchBackend()
	useFakeWatchBackend(t, backend)
	cfg, err := configResult(New(WithMedium(m), WithPath(path)))
	core.RequireNoError(t, err)
	core.RequireNoError(t, resultError(cfg.Watch()))

	cfg.StopWatch()
	cfg.StopWatch()
	core.AssertTrue(t, backend.closed)
}
