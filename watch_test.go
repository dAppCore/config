package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
)

func TestWatch_Watch_Good(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	assert.NoError(t, coreio.Local.Write(path, "key: one\n"))

	cfg, err := New(WithMedium(coreio.Local), WithPath(path))
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

	assert.NoError(t, coreio.Local.Write(path, "key: two\n"))

	// Allow the debounce window + fsnotify latency to settle.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := fired
		mu.Unlock()
		if got > 0 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Greater(t, fired, 0)
}

func TestWatch_Watch_Bad(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "missing.yaml")

	cfg, err := New(WithMedium(coreio.Local), WithPath(path))
	assert.NoError(t, err)
	// Watching a non-existent path should return an error rather than crashing.
	err = cfg.Watch()
	assert.Error(t, err)
}

func TestWatch_Watch_Ugly(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	assert.NoError(t, coreio.Local.Write(path, "key: value\n"))

	cfg, err := New(WithMedium(coreio.Local), WithPath(path))
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
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	assert.NoError(t, coreio.Local.Write(path, "dev:\n  editor: vim\napp:\n  name: alpha\n"))

	cfg, err := New(WithMedium(coreio.Local), WithPath(path))
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
	assert.NoError(t, coreio.Local.Write(path, "dev:\n  editor: nano\napp:\n  name: beta\n  version: \"1\"\n"))

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := len(seen)
		mu.Unlock()
		if got >= 3 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

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
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	assert.NoError(t, coreio.Local.Write(path, "key: first\n"))

	cfg, err := New(WithMedium(coreio.Local), WithPath(path))
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
	sidecar := filepath.Join(tmp, "config.yaml.swp")
	assert.NoError(t, coreio.Local.Write(sidecar, "key: second\n"))
	assert.NoError(t, os.Rename(sidecar, path))

	// Wait for the first rename-driven reload to land.
	waitFor(t, &mu, func() int { return fires }, 1)

	// Second atomic save: watcher must still be live.
	sidecar2 := filepath.Join(tmp, "config.yaml.swp2")
	assert.NoError(t, coreio.Local.Write(sidecar2, "key: third\n"))
	assert.NoError(t, os.Rename(sidecar2, path))

	waitFor(t, &mu, func() int { return fires }, 2)

	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, fires, 2, "watcher must survive the second atomic save")
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
		"dev.editor": "nano",   // changed
		"app.name":   "alpha",  // unchanged
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
