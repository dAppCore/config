package config

import (
	"errors"
	"sync"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

type mockConfigStore struct {
	bucket   string
	key      string
	value    string
	calls    int
	failWith error
}

func (s *mockConfigStore) Set(bucket, key, value string) error {
	s.calls++
	s.bucket = bucket
	s.key = key
	s.value = value
	if s.failWith != nil {
		return s.failWith
	}
	return nil
}

func TestConfig_MergeFrom_Good(t *core.T) {
	m := coreio.NewMockMedium()
	base, err := New(WithMedium(m), WithPath("/base.yaml"))
	core.AssertNoError(t, err)
	core.AssertNoError(t, base.Set("app.name", "base"))

	src, err := New(WithMedium(m), WithPath("/src.yaml"))
	core.AssertNoError(t, err)
	core.AssertNoError(t, src.Set("app.name", "src"))
	core.AssertNoError(t, src.Set("dev.editor", "vim"))

	base.MergeFrom(src)

	var name, editor string
	core.AssertNoError(t, base.Get("app.name", &name))
	core.AssertEqual(t, "base", name) // closest wins — base not overridden
	core.AssertNoError(t, base.Get("dev.editor", &editor))
	core.AssertEqual(t, "vim", editor) // gap filled from src
}

func TestConfig_MergeFrom_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	base, err := New(WithMedium(m), WithPath("/base.yaml"))
	core.AssertNoError(t, err)

	// Nil source is a no-op, not a panic.
	base.MergeFrom(nil)
}

func TestConfig_OnChange_Good(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/cb.yaml"))
	core.AssertNoError(t, err)

	var mu sync.Mutex
	seen := map[string]any{}
	cfg.OnChange(func(key string, value any) {
		mu.Lock()
		defer mu.Unlock()
		seen[key] = value
	})

	core.AssertNoError(t, cfg.Set("dev.editor", "vim"))

	mu.Lock()
	defer mu.Unlock()
	core.AssertEqual(t, "vim", seen["dev.editor"])
}

func TestConfig_OnChange_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/cb.yaml"))
	core.AssertNoError(t, err)

	// Nil callback is silently ignored, not stored.
	cfg.OnChange(nil)
	core.AssertNoError(t, cfg.Set("dev.editor", "vim"))
}

func TestConfig_Set_BroadcastsConfigChanged_Good(t *core.T) {
	m := coreio.NewMockMedium()
	c := core.New()

	var mu sync.Mutex
	var events []ConfigChanged
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if cc, ok := msg.(ConfigChanged); ok {
			mu.Lock()
			events = append(events, cc)
			mu.Unlock()
		}
		return core.Ok(nil)
	})

	cfg, err := New(WithMedium(m), WithPath("/b.yaml"), WithCore(c))
	core.AssertNoError(t, err)

	core.AssertNoError(t, cfg.Set("dev.editor", "vim"))

	mu.Lock()
	defer mu.Unlock()
	core.AssertGreaterOrEqual(t, len(events), 1)
	core.AssertEqual(t, "dev.editor", events[0].Key)
	core.AssertEqual(t, "vim", events[0].Value)
	core.AssertEqual(t, "set", events[0].Source)
}

func TestConfig_Medium_Good(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/medium.yaml"))
	core.AssertNoError(t, err)
	core.AssertSame(t, m, cfg.Medium())
}

func TestConfig_WithDefaults_Good(t *core.T) {
	// WithDefaults seeds the lowest-precedence layer: unset keys resolve to
	// the default; keys supplied by file/env/Set still win.
	m := coreio.NewMockMedium()
	cfg, err := New(
		WithMedium(m),
		WithPath("/defaults.yaml"),
		WithDefaults(map[string]any{
			"dev.editor":  "vim",
			"app.version": "0.1.0",
		}),
	)
	core.AssertNoError(t, err)

	var editor, version string
	core.AssertNoError(t, cfg.Get("dev.editor", &editor))
	core.AssertEqual(t, "vim", editor)
	core.AssertNoError(t, cfg.Get("app.version", &version))
	core.AssertEqual(t, "0.1.0", version)
}

func TestConfig_WithDefaults_Bad(t *core.T) {
	// An explicit Set() shadows the default — defaults are the floor, not a
	// ceiling.
	m := coreio.NewMockMedium()
	cfg, err := New(
		WithMedium(m),
		WithPath("/defaults.yaml"),
		WithDefaults(map[string]any{"dev.editor": "vim"}),
	)
	core.AssertNoError(t, err)
	core.AssertNoError(t, cfg.Set("dev.editor", "nano"))

	var editor string
	core.AssertNoError(t, cfg.Get("dev.editor", &editor))
	core.AssertEqual(t, "nano", editor)
}

func TestConfig_SetDefault_Good(t *core.T) {
	// SetDefault installs a runtime default — visible only while no other
	// source has set the key.
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/d.yaml"))
	core.AssertNoError(t, err)

	cfg.SetDefault("feature.beta", true)

	var beta bool
	core.AssertNoError(t, cfg.Get("feature.beta", &beta))
	core.AssertTrue(t, beta)
}

func TestConfig_SetDefault_Ugly(t *core.T) {
	// Defaults never broadcast ConfigChanged — they are a silent baseline.
	m := coreio.NewMockMedium()
	c := core.New()

	var mu sync.Mutex
	var events []ConfigChanged
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if cc, ok := msg.(ConfigChanged); ok {
			mu.Lock()
			events = append(events, cc)
			mu.Unlock()
		}
		return core.Ok(nil)
	})

	cfg, err := New(WithMedium(m), WithPath("/d.yaml"), WithCore(c))
	core.AssertNoError(t, err)

	cfg.SetDefault("feature.beta", true)

	mu.Lock()
	defer mu.Unlock()
	core.AssertEmpty(t, events)
}

func TestConfig_WithDefaults_FileWins_Good(t *core.T) {
	// File values shadow defaults even when both are present.
	m := coreio.NewMockMedium()
	m.Files["/defaults.yaml"] = "dev:\n  editor: nano\n"

	cfg, err := New(
		WithMedium(m),
		WithPath("/defaults.yaml"),
		WithDefaults(map[string]any{"dev.editor": "vim"}),
	)
	core.AssertNoError(t, err)

	var editor string
	core.AssertNoError(t, cfg.Get("dev.editor", &editor))
	core.AssertEqual(t, "nano", editor)
}

func TestConfig_AttachCore_Good(t *core.T) {
	// AttachCore wires a Core instance in after construction. Subsequent Set()
	// calls must broadcast ConfigChanged, even though the Config was created
	// without WithCore.
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/attach.yaml"))
	core.AssertNoError(t, err)

	c := core.New()
	var mu sync.Mutex
	var events []ConfigChanged
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if cc, ok := msg.(ConfigChanged); ok {
			mu.Lock()
			events = append(events, cc)
			mu.Unlock()
		}
		return core.Ok(nil)
	})

	// Before AttachCore, Set() does not broadcast.
	core.AssertNoError(t, cfg.Set("before.attach", "silent"))
	mu.Lock()
	core.AssertEmpty(t, events)
	mu.Unlock()

	cfg.AttachCore(c)

	// After AttachCore, Set() broadcasts.
	core.AssertNoError(t, cfg.Set("after.attach", "noisy"))
	mu.Lock()
	defer mu.Unlock()
	core.AssertGreaterOrEqual(t, len(events), 1)
	core.AssertEqual(t, "after.attach", events[0].Key)
	core.AssertEqual(t, "noisy", events[0].Value)
}

func TestConfig_AttachCore_Ugly(t *core.T) {
	// AttachCore is safe to call with nil — it simply leaves the Config in
	// pre-attach state with no panics on subsequent Set() calls.
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/attach.yaml"))
	core.AssertNoError(t, err)

	cfg.AttachCore(nil)
	core.AssertNoError(t, cfg.Set("quiet", "ok"))
}

func TestConfig_PersistToStore_Good(t *core.T) {
	store := &mockConfigStore{}
	m := coreio.NewMockMedium()
	cfg, err := New(WithStore(store), WithMedium(m), WithPath("/store.yaml"))
	core.AssertNoError(t, err)

	core.AssertNoError(t, cfg.Set("app.name", "core"))

	core.AssertEqual(t, 1, store.calls)
	core.AssertEqual(t, "config", store.bucket)
	core.AssertEqual(t, "app.name", store.key)
	core.AssertEqual(t, "\"core\"", store.value)
}

func TestConfig_PersistToStore_Bad(t *core.T) {
	store := &mockConfigStore{failWith: errors.New("store write failed")}
	m := coreio.NewMockMedium()
	cfg, err := New(WithStore(store), WithMedium(m), WithPath("/store.yaml"))
	core.AssertNoError(t, err)

	core.AssertNoError(t, cfg.Set("app.name", "core"))
	core.AssertEqual(t, 1, store.calls)
}

func TestConfig_PersistToStore_Ugly(t *core.T) {
	store := &mockConfigStore{}
	m := coreio.NewMockMedium()
	_, err := New(WithStore(store), WithMedium(m), WithPath("/store.yaml"))
	core.AssertNoError(t, err)

	core.AssertNotPanics(t, func() {
		persistToStore(nil, "app.name", "core")
		persistToStore(store, "", "core")
	})
	core.AssertEqual(t, 0, store.calls)
}
