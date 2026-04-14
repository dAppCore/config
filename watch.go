package config

import (
	"path/filepath"
	"time"

	coreerr "dappco.re/go/core/log"
	"github.com/fsnotify/fsnotify"
)

const watchDebounce = 100 * time.Millisecond

func (c *Config) Watch() error {
	c.watchMu.Lock()
	if c.watcher != nil || c.path == "" {
		c.watchMu.Unlock()
		if c.path == "" {
			return coreerr.E("config.Watch", "config path is empty", nil)
		}
		return nil
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	c.watcherStop = stop
	c.watcherDone = done
	c.watchMu.Unlock()

	watchPath := c.path
	if absPath, err := filepath.Abs(watchPath); err == nil {
		watchPath = absPath
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.watchMu.Lock()
		c.watcherStop = nil
		c.watcherDone = nil
		c.watchMu.Unlock()
		return coreerr.E("config.Watch", "failed to create watcher", err)
	}
	if err := watcher.Add(watchPath); err != nil {
		_ = watcher.Close()
		c.watchMu.Lock()
		c.watcherStop = nil
		c.watcherDone = nil
		c.watchMu.Unlock()
		close(stop)
		return coreerr.E("config.Watch", "failed to watch config path", err)
	}

	c.watchMu.Lock()
	c.watcher = watcher
	c.watchMu.Unlock()

	go c.watchLoop(watcher, stop, done, watchPath)
	return nil
}

func (c *Config) watchLoop(watcher *fsnotify.Watcher, stop chan struct{}, done chan struct{}, watchPath string) {
	defer close(done)
	defer func() {
		_ = watcher.Close()
	}()

	for {
		select {
		case <-stop:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if filepath.Base(event.Name) != filepath.Base(watchPath) {
				continue
			}
			if !shouldReloadEvent(event) {
				continue
			}
			c.scheduleReload()
		case <-watcher.Errors:
			continue
		}
	}
}

func shouldReloadEvent(event fsnotify.Event) bool {
	if event.Op&fsnotify.Write == 0 &&
		event.Op&fsnotify.Create == 0 &&
		event.Op&fsnotify.Remove == 0 &&
		event.Op&fsnotify.Rename == 0 {
		return false
	}
	return true
}

func (c *Config) scheduleReload() {
	c.watchMu.Lock()
	if c.watchTimer != nil {
		if !c.watchTimer.Stop() {
			select {
			case <-c.watchTimer.C:
			default:
			}
		}
	}
	c.watchTimer = time.AfterFunc(watchDebounce, func() {
		c.handleReload()
	})
	c.watchMu.Unlock()
}

func (c *Config) handleReload() {
	c.mu.RLock()
	old := cloneSettings(c.full.AllSettings())
	c.mu.RUnlock()

	if err := c.LoadFile(c.medium, c.path); err != nil {
		return
	}

	c.mu.RLock()
	latest := cloneSettings(c.full.AllSettings())
	c.mu.RUnlock()

	oldFlat := map[string]any{}
	latestFlat := map[string]any{}
	flattenForDiff("", old, oldFlat)
	flattenForDiff("", latest, latestFlat)
	changed := changedKeys(oldFlat, latestFlat)

	c.emitChanges(changed)
}

func (c *Config) StopWatch() {
	c.watchMu.Lock()
	stop := c.watcherStop
	done := c.watcherDone
	timer := c.watchTimer
	watcher := c.watcher
	c.watcherStop = nil
	c.watcherDone = nil
	c.watchTimer = nil
	c.watcher = nil
	c.watchMu.Unlock()

	if timer != nil {
		timer.Stop()
	}
	if stop != nil {
		select {
		case <-stop:
		default:
			close(stop)
		}
	}
	if watcher != nil {
		_ = watcher.Close()
	}
	if done != nil {
		select {
		case <-done:
		default:
		}
	}
}

func (c *Config) OnChange(fn func(string, any)) {
	if fn == nil {
		return
	}

	c.watchMu.Lock()
	c.watchers = append(c.watchers, fn)
	c.watchMu.Unlock()
}
