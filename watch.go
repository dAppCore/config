package config

import (
	"reflect"
	"slices"
	"sync"
	"time"

	coreerr "dappco.re/go/core/log"
	"github.com/fsnotify/fsnotify"
)

// debounceWindow coalesces rapid filesystem events from multi-step editor saves.
const debounceWindow = 100 * time.Millisecond

type fileWatcher struct {
	mu      sync.Mutex
	w       *fsnotify.Watcher
	stop    chan struct{}
	stopped bool
}

// Watch starts monitoring the config file for changes. When the file is modified,
// registered OnChange callbacks are invoked for every key whose value changed
// between the previous state and the reloaded state. Rapid filesystem events
// within a 100ms window are coalesced into a single reload+diff pass.
//
//	cfg.Watch()
//	defer cfg.StopWatch()
func (c *Config) Watch() error {
	c.mu.Lock()
	if c.watcher != nil {
		c.mu.Unlock()
		return nil
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		c.mu.Unlock()
		return coreerr.E("config.Watch", "failed to create watcher", err)
	}
	path := c.path
	fw := &fileWatcher{w: w, stop: make(chan struct{})}
	c.watcher = fw
	c.mu.Unlock()

	if err := w.Add(path); err != nil {
		_ = w.Close()
		c.mu.Lock()
		c.watcher = nil
		c.mu.Unlock()
		return coreerr.E("config.Watch", "failed to watch path: "+path, err)
	}

	go c.watchLoop(fw)
	return nil
}

// StopWatch stops the filesystem watcher if one is running.
//
//	defer cfg.StopWatch()
func (c *Config) StopWatch() {
	c.mu.Lock()
	fw := c.watcher
	c.watcher = nil
	c.mu.Unlock()
	if fw == nil {
		return
	}
	fw.mu.Lock()
	if !fw.stopped {
		fw.stopped = true
		close(fw.stop)
		_ = fw.w.Close()
	}
	fw.mu.Unlock()
}

func (c *Config) watchLoop(fw *fileWatcher) {
	var timer *time.Timer
	for {
		select {
		case <-fw.stop:
			return
		case ev, ok := <-fw.w.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
				continue
			}
			// Atomic-save editors (vim, VSCode) rename/replace the file on save.
			// fsnotify tracks the old inode, so the watch silently dies — re-Add
			// the watch on the same path so subsequent saves still fire events.
			if ev.Op&(fsnotify.Rename|fsnotify.Remove) != 0 {
				c.mu.RLock()
				path := c.path
				c.mu.RUnlock()
				// Best-effort: the new file may not exist yet during an atomic
				// swap window. Add silently retries on the next event cycle.
				_ = fw.w.Add(path)
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounceWindow, func() {
				c.reloadAndNotify()
			})
		case _, ok := <-fw.w.Errors:
			if !ok {
				return
			}
		}
	}
}

// reloadAndNotify snapshots the current values, reloads the underlying file,
// and fires OnChange callbacks for each key whose value differs between the
// snapshot and the reloaded state. Source on the broadcast ConfigChanged is
// "file" — it distinguishes filesystem reloads from in-process Set() calls.
func (c *Config) reloadAndNotify() {
	before := c.snapshotAll()

	if err := c.LoadFile(c.medium, c.path); err != nil {
		return
	}

	after := c.snapshotAll()
	changes := diffSnapshots(before, after)

	c.mu.RLock()
	callbacks := append([]func(string, any){}, c.callbacks...)
	attached := c.core
	c.mu.RUnlock()

	for _, change := range changes {
		for _, fn := range callbacks {
			fn(change.Key, change.Value)
		}
		if attached != nil {
			attached.ACTION(ConfigChanged{
				Key:      change.Key,
				Value:    change.Value,
				Previous: change.Previous,
				Source:   "file",
			})
		}
	}
}

// snapshotAll copies every key/value currently known to the full viper into a
// flat map so the watcher can diff before/after reload. The read lock guards
// the underlying viper during the AllSettings walk.
func (c *Config) snapshotAll() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]any, len(c.full.AllKeys()))
	for _, key := range c.full.AllKeys() {
		out[key] = c.full.Get(key)
	}
	return out
}

// configChange describes a single key-level transition between snapshots.
type configChange struct {
	Key      string
	Value    any
	Previous any
}

// diffSnapshots returns every key whose value changed (or appeared/disappeared)
// between before and after. Order is lexical so repeated reloads produce a
// deterministic callback sequence.
func diffSnapshots(before, after map[string]any) []configChange {
	keys := make(map[string]struct{}, len(before)+len(after))
	for k := range before {
		keys[k] = struct{}{}
	}
	for k := range after {
		keys[k] = struct{}{}
	}
	ordered := make([]string, 0, len(keys))
	for k := range keys {
		ordered = append(ordered, k)
	}
	sortStrings(ordered)

	changes := make([]configChange, 0, len(ordered))
	for _, k := range ordered {
		prev, hadPrev := before[k]
		next, hasNext := after[k]
		if hadPrev == hasNext && equalAny(prev, next) {
			continue
		}
		changes = append(changes, configChange{Key: k, Value: next, Previous: prev})
	}
	return changes
}

// sortStrings sorts keys lexically via slices.Sort so the diff helpers stay
// dependency-thin without pulling in the banned sort package.
func sortStrings(keys []string) {
	slices.Sort(keys)
}

// equalAny compares two any values, including map[string]any and []any shapes
// that yaml/viper commonly produce. Falls back to reflect.DeepEqual so nested
// structures compare correctly regardless of concrete element type.
func equalAny(a, b any) bool {
	return reflect.DeepEqual(a, b)
}
