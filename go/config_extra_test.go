package config

import (
	"sync"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

const (
	configExtraAppNameKey     = "app.name"
	configExtraDevEditorKey   = "dev.editor"
	configExtraDefaultsPath   = "/defaults.yaml"
	configExtraFeatureBetaKey = "feature.beta"
	configExtraStorePath      = "/store.yaml"
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
	base, err := configResult(New(WithMedium(m), WithPath("/base.yaml")))
	core.AssertNoError(t, err)
	core.AssertNoError(t, resultError(base.Set(configExtraAppNameKey, "base")))

	src, err := configResult(New(WithMedium(m), WithPath("/src.yaml")))
	core.AssertNoError(t, err)
	core.AssertNoError(t, resultError(src.Set(configExtraAppNameKey, "src")))
	core.AssertNoError(t, resultError(src.Set(configExtraDevEditorKey, "vim")))

	base.MergeFrom(src)

	var name, editor string
	core.AssertNoError(t, resultError(base.Get(configExtraAppNameKey, &name)))
	core.AssertEqual(t, "base", name) // closest wins — base not overridden
	core.AssertNoError(t, resultError(base.Get(configExtraDevEditorKey, &editor)))
	core.AssertEqual(t, "vim", editor) // gap filled from src
}

func TestConfig_MergeFrom_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	base, err := configResult(New(WithMedium(m), WithPath("/base.yaml")))
	core.AssertNoError(t, err)

	// Nil source is a no-op, not a panic.
	base.MergeFrom(nil)
}

func TestConfig_OnChange_Good(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := configResult(New(WithMedium(m), WithPath("/cb.yaml")))
	core.AssertNoError(t, err)

	var mu sync.Mutex
	seen := map[string]any{}
	cfg.OnChange(func(key string, value any) {
		mu.Lock()
		defer mu.Unlock()
		seen[key] = value
	})

	core.AssertNoError(t, resultError(cfg.Set(configExtraDevEditorKey, "vim")))

	mu.Lock()
	defer mu.Unlock()
	core.AssertEqual(t, "vim", seen[configExtraDevEditorKey])
}

func TestConfig_OnChange_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := configResult(New(WithMedium(m), WithPath("/cb.yaml")))
	core.AssertNoError(t, err)

	// Nil callback is silently ignored, not stored.
	cfg.OnChange(nil)
	core.AssertNoError(t, resultError(cfg.Set(configExtraDevEditorKey, "vim")))
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

	cfg, err := configResult(New(WithMedium(m), WithPath("/b.yaml"), WithCore(c)))
	core.AssertNoError(t, err)

	core.AssertNoError(t, resultError(cfg.Set(configExtraDevEditorKey, "vim")))

	mu.Lock()
	defer mu.Unlock()
	core.AssertGreaterOrEqual(t, len(events), 1)
	core.AssertEqual(t, configExtraDevEditorKey, events[0].Key)
	core.AssertEqual(t, "vim", events[0].Value)
	core.AssertEqual(t, "set", events[0].Source)
}

func TestConfig_Medium_Good(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := configResult(New(WithMedium(m), WithPath("/medium.yaml")))
	core.AssertNoError(t, err)
	core.AssertSame(t, m, cfg.Medium())
}

func TestConfig_SetDefault_Good(t *core.T) {
	// SetDefault installs a runtime default — visible only while no other
	// source has set the key.
	m := coreio.NewMockMedium()
	cfg, err := configResult(New(WithMedium(m), WithPath("/d.yaml")))
	core.AssertNoError(t, err)

	cfg.SetDefault(configExtraFeatureBetaKey, true)

	var beta bool
	core.AssertNoError(t, resultError(cfg.Get(configExtraFeatureBetaKey, &beta)))
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

	cfg, err := configResult(New(WithMedium(m), WithPath("/d.yaml"), WithCore(c)))
	core.AssertNoError(t, err)

	cfg.SetDefault(configExtraFeatureBetaKey, true)

	mu.Lock()
	defer mu.Unlock()
	core.AssertEmpty(t, events)
}

func TestConfig_WithDefaults_FileWins_Good(t *core.T) {
	// File values shadow defaults even when both are present.
	m := coreio.NewMockMedium()
	m.Files[configExtraDefaultsPath] = "dev:\n  editor: nano\n"

	cfg, err := configResult(New(
		WithMedium(m),
		WithPath(configExtraDefaultsPath),
		WithDefaults(map[string]any{configExtraDevEditorKey: "vim"}),
	))
	core.AssertNoError(t, err)

	var editor string
	core.AssertNoError(t, resultError(cfg.Get(configExtraDevEditorKey, &editor)))
	core.AssertEqual(t, "nano", editor)
}

func TestConfig_AttachCore_Good(t *core.T) {
	// AttachCore wires a Core instance in after construction. Subsequent Set()
	// calls must broadcast ConfigChanged, even though the Config was created
	// without WithCore.
	m := coreio.NewMockMedium()
	cfg, err := configResult(New(WithMedium(m), WithPath("/attach.yaml")))
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
	core.AssertNoError(t, resultError(cfg.Set("before.attach", "silent")))
	mu.Lock()
	core.AssertEmpty(t, events)
	mu.Unlock()

	cfg.AttachCore(c)

	// After AttachCore, Set() broadcasts.
	core.AssertNoError(t, resultError(cfg.Set("after.attach", "noisy")))
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
	cfg, err := configResult(New(WithMedium(m), WithPath("/attach.yaml")))
	core.AssertNoError(t, err)

	cfg.AttachCore(nil)
	core.AssertNoError(t, resultError(cfg.Set("quiet", "ok")))
}

func TestConfig_Config_Set_PersistToStore_Good(t *core.T) {
	store := &mockConfigStore{}
	m := coreio.NewMockMedium()
	cfg, err := configResult(New(WithStore(store), WithMedium(m), WithPath(configExtraStorePath)))
	core.AssertNoError(t, err)

	core.AssertNoError(t, resultError(cfg.Set(configExtraAppNameKey, "core")))

	core.AssertEqual(t, 1, store.calls)
	core.AssertEqual(t, "config", store.bucket)
	core.AssertEqual(t, configExtraAppNameKey, store.key)
	core.AssertEqual(t, "\"core\"", store.value)
}

func TestConfig_Config_Set_PersistToStore_Bad(t *core.T) {
	store := &mockConfigStore{failWith: core.NewError("store write failed")}
	m := coreio.NewMockMedium()
	cfg, err := configResult(New(WithStore(store), WithMedium(m), WithPath(configExtraStorePath)))
	core.AssertNoError(t, err)

	core.AssertNoError(t, resultError(cfg.Set(configExtraAppNameKey, "core")))
	core.AssertEqual(t, 1, store.calls)
}

func TestConfig_persistToStore_Ugly(t *core.T) {
	store := &mockConfigStore{}
	m := coreio.NewMockMedium()
	_, err := configResult(New(WithStore(store), WithMedium(m), WithPath(configExtraStorePath)))
	core.AssertNoError(t, err)

	core.AssertNotPanics(t, func() {
		persistToStore(nil, configExtraAppNameKey, "core")
		persistToStore(store, "", "core")
	})
	core.AssertEqual(t, 0, store.calls)
}
