package config

import (
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
// registered OnChange callbacks are invoked with the empty key to signal a full
// reload. Rapid filesystem events within a 100ms window are debounced.
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
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
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

// reloadAndNotify reloads the underlying file and fires OnChange callbacks.
func (c *Config) reloadAndNotify() {
	if err := c.LoadFile(c.medium, c.path); err != nil {
		return
	}
	c.mu.RLock()
	callbacks := append([]func(string, any){}, c.callbacks...)
	attached := c.core
	c.mu.RUnlock()

	for _, fn := range callbacks {
		fn("", nil)
	}
	if attached != nil {
		attached.ACTION(ConfigChanged{Source: "file"})
	}
}
