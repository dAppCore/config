package config

import (
	"sync"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
)

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
