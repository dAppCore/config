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

// mockConfigReaderStore implements both ConfigStoreWriter and ConfigStoreReader
// so loadStoreState() rehydrates previously-persisted values on New().
//
//	store := &mockConfigReaderStore{entries: map[string]string{"app.name": "\"core\""}}
//	cfg, _ := configResult(New(WithStore(store), WithMedium(coreio.NewMockMedium()), WithPath("/store.yaml")))
type mockConfigReaderStore struct {
	mockConfigStore
	entries  map[string]string
	getBkt   string
	getCalls int
	getFail  error
}

func (s *mockConfigReaderStore) GetAll(bucket string) (map[string]string, error) {
	s.getCalls++
	s.getBkt = bucket
	if s.getFail != nil {
		return nil, s.getFail
	}
	return s.entries, nil
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

// TestConfig_loadStoreState_Good asserts New() rehydrates JSON-encoded values
// from a ConfigStoreReader so pre-Commit Set() calls survive a restart.
func TestConfig_loadStoreState_Good(t *core.T) {
	store := &mockConfigReaderStore{entries: map[string]string{
		configExtraAppNameKey:     "\"core\"",
		configExtraDevEditorKey:   "\"vim\"",
		configExtraFeatureBetaKey: "true",
	}}
	m := coreio.NewMockMedium()

	cfg, err := configResult(New(WithStore(store), WithMedium(m), WithPath(configExtraStorePath)))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, store.getCalls)
	core.AssertEqual(t, "config", store.getBkt)

	var name, editor string
	var beta bool
	core.AssertNoError(t, resultError(cfg.Get(configExtraAppNameKey, &name)))
	core.AssertEqual(t, "core", name)
	core.AssertNoError(t, resultError(cfg.Get(configExtraDevEditorKey, &editor)))
	core.AssertEqual(t, "vim", editor)
	core.AssertNoError(t, resultError(cfg.Get(configExtraFeatureBetaKey, &beta)))
	core.AssertTrue(t, beta)
}

// TestConfig_loadStoreState_Bad asserts a GetAll() failure surfaces as a New()
// error rather than silently dropping the stored state.
func TestConfig_loadStoreState_Bad(t *core.T) {
	store := &mockConfigReaderStore{getFail: core.NewError("store read refused")}
	m := coreio.NewMockMedium()

	_, err := configResult(New(WithStore(store), WithMedium(m), WithPath(configExtraStorePath)))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to load config store state")
}

// TestConfig_loadStoreState_Ugly asserts empty keys are skipped, non-JSON raw
// values fall back to plain strings, and empty raw decodes to "".
func TestConfig_loadStoreState_Ugly(t *core.T) {
	store := &mockConfigReaderStore{entries: map[string]string{
		"":                      "\"ignored\"",
		configExtraAppNameKey:   "plain-not-json",
		configExtraDevEditorKey: "",
	}}
	m := coreio.NewMockMedium()

	cfg, err := configResult(New(WithStore(store), WithMedium(m), WithPath(configExtraStorePath)))
	core.AssertNoError(t, err)

	var name string
	core.AssertNoError(t, resultError(cfg.Get(configExtraAppNameKey, &name)))
	core.AssertEqual(t, "plain-not-json", name) // non-JSON raw falls through verbatim
}

// TestConfig_loadStoreState_NoReader asserts a writer-only store leaves
// loadStoreState() a no-op (no GetAll call, New() still succeeds).
func TestConfig_loadStoreState_NoReader(t *core.T) {
	store := &mockConfigStore{}
	m := coreio.NewMockMedium()

	_, err := configResult(New(WithStore(store), WithMedium(m), WithPath(configExtraStorePath)))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, store.calls)
}

// TestConfig_decodeStoredConfigValue_Good asserts JSON scalars and structures
// decode to their native Go shapes.
func TestConfig_decodeStoredConfigValue_Good(t *core.T) {
	core.AssertEqual(t, "core", decodeStoredConfigValue("\"core\""))
	core.AssertEqual(t, true, decodeStoredConfigValue("true"))
	core.AssertEqual(t, float64(42), decodeStoredConfigValue("42"))
}

// TestConfig_decodeStoredConfigValue_Bad asserts non-JSON raw input falls back
// to the verbatim string rather than erroring.
func TestConfig_decodeStoredConfigValue_Bad(t *core.T) {
	core.AssertEqual(t, "not json at all", decodeStoredConfigValue("not json at all"))
}

// TestConfig_decodeStoredConfigValue_Ugly asserts the empty-string short-circuit.
func TestConfig_decodeStoredConfigValue_Ugly(t *core.T) {
	core.AssertEqual(t, "", decodeStoredConfigValue(""))
}
