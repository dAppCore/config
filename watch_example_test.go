package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	"github.com/fsnotify/fsnotify"
)

type Backend = watchBackend

func ExampleBackend_Add() {
	backend := newFakeWatchBackend()
	err := resultError(backend.Add("/example/config.yaml"))
	core.Println(err == nil, backend.addCount())
	// Output: true 1
}

func ExampleBackend_Close() {
	backend := newFakeWatchBackend()
	err := resultError(backend.Close())
	core.Println(err == nil, backend.closed)
	// Output: true true
}

func ExampleBackend_Events() {
	backend := newFakeWatchBackend()
	backend.emit(fsnotify.Event{Name: "/example/config.yaml", Op: fsnotify.Write})
	event := <-backend.Events()
	core.Println(event.Name)
	// Output: /example/config.yaml
}

func ExampleBackend_Errors() {
	backend := newFakeWatchBackend()
	backend.errors <- core.Fail(core.NewError("watch failed"))
	result := <-backend.Errors()
	core.Println(!result.OK)
	// Output: true
}

func ExampleConfig_Watch() {
	m := coreio.NewMockMedium()
	path := "/example/config.yaml"
	_ = m.Write(path, "app:\n  name: before\n")
	backend := newFakeWatchBackend()
	previous := newWatchBackend
	newWatchBackend = func() core.Result {
		return core.Ok(backend)
	}
	defer func() {
		newWatchBackend = previous
	}()

	cfg, _ := configResult(New(WithMedium(m), WithPath(path)))
	err := resultError(cfg.Watch())
	cfg.StopWatch()
	core.Println(err == nil, backend.closed)
	// Output: true true
}

func ExampleConfig_StopWatch() {
	m := coreio.NewMockMedium()
	path := "/example/config.yaml"
	_ = m.Write(path, "app:\n  name: before\n")
	backend := newFakeWatchBackend()
	previous := newWatchBackend
	newWatchBackend = func() core.Result {
		return core.Ok(backend)
	}
	defer func() {
		newWatchBackend = previous
	}()

	cfg, _ := configResult(New(WithMedium(m), WithPath(path)))
	_ = cfg.Watch()
	cfg.StopWatch()
	core.Println(backend.closed)
	// Output: true
}
