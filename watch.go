package config

import (
	"reflect"
	"slices"
	"sync"
	"time"

	core "dappco.re/go"
	coreerr "dappco.re/go/log"
	"github.com/fsnotify/fsnotify"
)

// debounceWindow coalesces rapid filesystem events from multi-step editor saves.
const debounceWindow = 100 * time.Millisecond

type fileWatcher struct {
	mu      sync.Mutex
	w       watchBackend
	stop    chan struct{}
	stopped bool
}

type watchBackend interface {
	Add(string) core.Result
	Close() core.Result
	Events() <-chan fsnotify.Event
	Errors() <-chan core.Result
}

type fsnotifyBackend struct {
	w      *fsnotify.Watcher
	errors <-chan core.Result
}

func (b fsnotifyBackend) Add(path string) core.Result {
	return core.ResultOf(nil, b.w.Add(path))
}

func (b fsnotifyBackend) Close() core.Result {
	return core.ResultOf(nil, b.w.Close())
}

func (b fsnotifyBackend) Events() <-chan fsnotify.Event {
	return b.w.Events
}

func (b fsnotifyBackend) Errors() <-chan core.Result {
	return b.errors
}

var newWatchBackend = func() core.Result {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return core.Fail(err)
	}
	errors := make(chan core.Result)
	go func() {
		defer close(errors)
		for err := range w.Errors {
			errors <- core.Fail(err)
		}
	}()
	return core.Ok(fsnotifyBackend{w: w, errors: errors})
}

// Watch starts monitoring the config file for changes. When the file is modified,
// registered OnChange callbacks are invoked for every key whose value changed
// between the previous state and the reloaded state. Rapid filesystem events
// within a 100ms window are coalesced into a single reload+diff pass.
//
//	cfg.Watch()
//	defer cfg.StopWatch()
func (c *Config) Watch() core.Result {
	c.mu.Lock()
	if c.watcher != nil {
		c.mu.Unlock()
		return core.Ok(nil)
	}
	wResult := newWatchBackend()
	if !wResult.OK {
		c.mu.Unlock()
		return core.Fail(coreerr.E("config.Watch", "failed to create watcher", resultCause(wResult).(error)))
	}
	w := wResult.Value.(watchBackend)
	path := c.path
	fw := &fileWatcher{w: w, stop: make(chan struct{})}
	c.watcher = fw
	c.mu.Unlock()

	if r := w.Add(path); !r.OK {
		watchErr := resultCause(r).(error)
		if closeResult := w.Close(); !closeResult.OK {
			watchErr = core.ErrorJoin(watchErr, resultCause(closeResult).(error))
		}
		c.mu.Lock()
		c.watcher = nil
		c.mu.Unlock()
		return core.Fail(coreerr.E("config.Watch", "failed to watch path: "+path, watchErr))
	}

	go c.watchLoop(fw)
	return core.Ok(nil)
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
		if r := fw.w.Close(); !r.OK {
			// StopWatch is best-effort; callers cannot act on watcher close errors.
		}
	}
	fw.mu.Unlock()
}

func (c *Config) watchLoop(fw *fileWatcher) {
	reloadRequests := make(chan struct{}, 1)
	done := make(chan struct{})
	go c.watchReloadLoop(fw.stop, done, reloadRequests)
	defer close(done)

	for {
		select {
		case <-fw.stop:
			return
		case ev, ok := <-fw.w.Events():
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
				// Best-effort for atomic-save editors: the replacement file may
				// not exist during the swap. There is no automatic retry loop;
				// another fsnotify event is required to attempt Add again.
				if r := fw.w.Add(path); !r.OK {
					// The current filesystem event still requests a reload below.
				}
			}
			requestReload(reloadRequests)
		case _, ok := <-fw.w.Errors():
			if !ok {
				return
			}
		}
	}
}

func (c *Config) watchReloadLoop(stop, done <-chan struct{}, requests <-chan struct{}) {
	timer := time.NewTimer(debounceWindow)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	for {
		select {
		case <-stop:
			return
		case <-done:
			return
		case <-requests:
			resetDebounceTimer(timer)
		case <-timer.C:
			select {
			case <-stop:
				return
			case <-done:
				return
			default:
				c.reloadAndNotify()
			}
		}
	}
}

func requestReload(requests chan<- struct{}) {
	select {
	case requests <- struct{}{}:
	default:
	}
}

func resetDebounceTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(debounceWindow)
}

// reloadAndNotify snapshots the current values, reloads the underlying file,
// and fires OnChange callbacks for each key whose value differs between the
// snapshot and the reloaded state. Source on the broadcast ConfigChanged is
// "file" — it distinguishes filesystem reloads from in-process Set() calls.
func (c *Config) reloadAndNotify() {
	before := c.snapshotAll()

	if r := c.loadFile(c.medium, c.path, false); !r.OK {
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
				Source:   configChangeSourceFile,
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
