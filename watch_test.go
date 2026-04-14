package config

import (
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
