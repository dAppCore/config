package config

import (
	"errors"
	"sync"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
)

type mockConfigStore struct {
	bucket   string
	key      string
	value    string
	calls    int
	failWith error
}

func (s *mockConfigStore) Set(bucket string, key string, value string) error {
	s.calls++
	s.bucket = bucket
	s.key = key
	s.value = value
	if s.failWith != nil {
		return s.failWith
	}
	return nil
}

func TestConfig_MergeFrom_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	base, err := New(WithMedium(m), WithPath("/base.yaml"))
	assert.NoError(t, err)
	assert.NoError(t, base.Set("app.name", "base"))

	src, err := New(WithMedium(m), WithPath("/src.yaml"))
	assert.NoError(t, err)
	assert.NoError(t, src.Set("app.name", "src"))
	assert.NoError(t, src.Set("dev.editor", "vim"))

	base.MergeFrom(src)

	var name, editor string
	assert.NoError(t, base.Get("app.name", &name))
	assert.Equal(t, "base", name) // closest wins — base not overridden
	assert.NoError(t, base.Get("dev.editor", &editor))
	assert.Equal(t, "vim", editor) // gap filled from src
}

func TestConfig_MergeFrom_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	base, err := New(WithMedium(m), WithPath("/base.yaml"))
	assert.NoError(t, err)

	// Nil source is a no-op, not a panic.
	base.MergeFrom(nil)
}

func TestConfig_OnChange_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/cb.yaml"))
	assert.NoError(t, err)

	var mu sync.Mutex
	seen := map[string]any{}
	cfg.OnChange(func(key string, value any) {
		mu.Lock()
		defer mu.Unlock()
		seen[key] = value
	})

	assert.NoError(t, cfg.Set("dev.editor", "vim"))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "vim", seen["dev.editor"])
}

func TestConfig_OnChange_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/cb.yaml"))
	assert.NoError(t, err)

	// Nil callback is silently ignored, not stored.
	cfg.OnChange(nil)
	assert.NoError(t, cfg.Set("dev.editor", "vim"))
}

func TestConfig_Set_BroadcastsConfigChanged_Good(t *testing.T) {
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
		return core.Result{}
	})

	cfg, err := New(WithMedium(m), WithPath("/b.yaml"), WithCore(c))
	assert.NoError(t, err)

	assert.NoError(t, cfg.Set("dev.editor", "vim"))

	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, len(events), 1)
	assert.Equal(t, "dev.editor", events[0].Key)
	assert.Equal(t, "vim", events[0].Value)
	assert.Equal(t, "set", events[0].Source)
}

func TestConfig_Medium_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/medium.yaml"))
	assert.NoError(t, err)
	assert.Same(t, m, cfg.Medium())
}

func TestConfig_WithDefaults_Good(t *testing.T) {
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
	assert.NoError(t, err)

	var editor, version string
	assert.NoError(t, cfg.Get("dev.editor", &editor))
	assert.Equal(t, "vim", editor)
	assert.NoError(t, cfg.Get("app.version", &version))
	assert.Equal(t, "0.1.0", version)
}

func TestConfig_WithDefaults_Bad(t *testing.T) {
	// An explicit Set() shadows the default — defaults are the floor, not a
	// ceiling.
	m := coreio.NewMockMedium()
	cfg, err := New(
		WithMedium(m),
		WithPath("/defaults.yaml"),
		WithDefaults(map[string]any{"dev.editor": "vim"}),
	)
	assert.NoError(t, err)
	assert.NoError(t, cfg.Set("dev.editor", "nano"))

	var editor string
	assert.NoError(t, cfg.Get("dev.editor", &editor))
	assert.Equal(t, "nano", editor)
}

func TestConfig_SetDefault_Good(t *testing.T) {
	// SetDefault installs a runtime default — visible only while no other
	// source has set the key.
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/d.yaml"))
	assert.NoError(t, err)

	cfg.SetDefault("feature.beta", true)

	var beta bool
	assert.NoError(t, cfg.Get("feature.beta", &beta))
	assert.True(t, beta)
}

func TestConfig_SetDefault_Ugly(t *testing.T) {
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
		return core.Result{}
	})

	cfg, err := New(WithMedium(m), WithPath("/d.yaml"), WithCore(c))
	assert.NoError(t, err)

	cfg.SetDefault("feature.beta", true)

	mu.Lock()
	defer mu.Unlock()
	assert.Empty(t, events)
}

func TestConfig_WithDefaults_FileWins_Good(t *testing.T) {
	// File values shadow defaults even when both are present.
	m := coreio.NewMockMedium()
	m.Files["/defaults.yaml"] = "dev:\n  editor: nano\n"

	cfg, err := New(
		WithMedium(m),
		WithPath("/defaults.yaml"),
		WithDefaults(map[string]any{"dev.editor": "vim"}),
	)
	assert.NoError(t, err)

	var editor string
	assert.NoError(t, cfg.Get("dev.editor", &editor))
	assert.Equal(t, "nano", editor)
}

func TestConfig_AttachCore_Good(t *testing.T) {
	// AttachCore wires a Core instance in after construction. Subsequent Set()
	// calls must broadcast ConfigChanged, even though the Config was created
	// without WithCore.
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/attach.yaml"))
	assert.NoError(t, err)

	c := core.New()
	var mu sync.Mutex
	var events []ConfigChanged
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if cc, ok := msg.(ConfigChanged); ok {
			mu.Lock()
			events = append(events, cc)
			mu.Unlock()
		}
		return core.Result{}
	})

	// Before AttachCore, Set() does not broadcast.
	assert.NoError(t, cfg.Set("before.attach", "silent"))
	mu.Lock()
	assert.Empty(t, events)
	mu.Unlock()

	cfg.AttachCore(c)

	// After AttachCore, Set() broadcasts.
	assert.NoError(t, cfg.Set("after.attach", "noisy"))
	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, len(events), 1)
	assert.Equal(t, "after.attach", events[0].Key)
	assert.Equal(t, "noisy", events[0].Value)
}

func TestConfig_AttachCore_Ugly(t *testing.T) {
	// AttachCore is safe to call with nil — it simply leaves the Config in
	// pre-attach state with no panics on subsequent Set() calls.
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/attach.yaml"))
	assert.NoError(t, err)

	cfg.AttachCore(nil)
	assert.NoError(t, cfg.Set("quiet", "ok"))
}

func TestConfig_PersistToStore_Good(t *testing.T) {
	store := &mockConfigStore{}
	cfg, err := New(WithStore(store))
	assert.NoError(t, err)

	assert.NoError(t, cfg.Set("app.name", "core"))

	assert.Equal(t, 1, store.calls)
	assert.Equal(t, "config", store.bucket)
	assert.Equal(t, "app.name", store.key)
	assert.Equal(t, "\"core\"", store.value)
}

func TestConfig_PersistToStore_Bad(t *testing.T) {
	store := &mockConfigStore{failWith: errors.New("store write failed")}
	cfg, err := New(WithStore(store))
	assert.NoError(t, err)

	assert.NoError(t, cfg.Set("app.name", "core"))
	assert.Equal(t, 1, store.calls)
}

func TestConfig_PersistToStore_Ugly(t *testing.T) {
	store := &mockConfigStore{}
	_, err := New(WithStore(store))
	assert.NoError(t, err)

	assert.NotPanics(t, func() {
		persistToStore(nil, "app.name", "core")
		persistToStore(store, "", "core")
	})
	assert.Equal(t, 0, store.calls)
}
