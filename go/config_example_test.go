package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

type exampleConfigStore struct {
	values map[string]string
}

func (s *exampleConfigStore) Set(bucket, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[bucket+"."+key] = value
	return nil
}

func ExampleWithMedium() {
	m := coreio.NewMockMedium()
	cfg := core.MustCast[*Config](New(WithMedium(m), WithPath("/example/config.yaml")))
	core.Println(cfg.Medium() == m)
	// Output: true
}

func ExampleWithPath() {
	cfg := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/app.yaml")))
	core.Println(cfg.Path())
	// Output: /example/app.yaml
}

func ExampleWithEnvPrefix() {
	result := New(WithMedium(coreio.NewMockMedium()), WithPath("/example/config.yaml"), WithEnvPrefix("APP"))
	cfg, _ := core.Cast[*Config](result)
	core.Println(result.OK && cfg != nil)
	// Output: true
}

func ExampleWithCore() {
	c := core.New()
	cfg := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/config.yaml"), WithCore(c)))
	core.Println(cfg.core == c)
	// Output: true
}

func ExampleWithStore() {
	store := &exampleConfigStore{}
	cfg := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/config.yaml"), WithStore(store)))
	_ = cfg.Set("agent.name", "codex")
	core.Println(store.values["config.agent.name"])
	// Output: "codex"
}

func ExampleWithDefaults() {
	cfg := core.MustCast[*Config](New(
		WithMedium(coreio.NewMockMedium()),
		WithPath("/example/config.yaml"),
		WithDefaults(map[string]any{"app.name": "core"}),
	))
	var name string
	_ = cfg.Get("app.name", &name)
	core.Println(name)
	// Output: core
}

func ExampleConfig_AttachCore() {
	cfg := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/config.yaml")))
	c := core.New()
	cfg.AttachCore(c)
	core.Println(cfg.core == c)
	// Output: true
}

func ExampleNew() {
	result := New(WithMedium(coreio.NewMockMedium()), WithPath("/example/config.yaml"))
	cfg, _ := core.Cast[*Config](result)
	core.Println(result.OK && cfg != nil)
	// Output: true
}

func ExampleConfig_LoadFile() {
	m := coreio.NewMockMedium()
	_ = m.Write("/example/extra.yaml", "app:\n  name: loaded\n")
	cfg := core.MustCast[*Config](New(WithMedium(m), WithPath("/example/config.yaml")))
	_ = cfg.LoadFile(m, "/example/extra.yaml")
	var name string
	_ = cfg.Get("app.name", &name)
	core.Println(name)
	// Output: loaded
}

func ExampleConfig_Get() {
	cfg := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/config.yaml")))
	_ = cfg.Set("dev.editor", "vim")
	var editor string
	_ = cfg.Get("dev.editor", &editor)
	core.Println(editor)
	// Output: vim
}

func ExampleConfig_SetDefault() {
	cfg := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/config.yaml")))
	cfg.SetDefault("feature.beta", true)
	var beta bool
	_ = cfg.Get("feature.beta", &beta)
	core.Println(beta)
	// Output: true
}

func ExampleConfig_Set() {
	cfg := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/config.yaml")))
	_ = cfg.Set("dev.shell", "zsh")
	var shell string
	_ = cfg.Get("dev.shell", &shell)
	core.Println(shell)
	// Output: zsh
}

func ExampleConfig_Commit() {
	m := coreio.NewMockMedium()
	cfg := core.MustCast[*Config](New(WithMedium(m), WithPath("/example/config.yaml")))
	_ = cfg.Set("app.name", "core")
	_ = cfg.Commit()
	core.Println(m.Exists("/example/config.yaml"))
	// Output: true
}

func ExampleConfig_All() {
	cfg := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/config.yaml")))
	_ = cfg.Set("app.name", "core")
	found := false
	for key := range cfg.All() {
		if key == "app.name" {
			found = true
		}
	}
	core.Println(found)
	// Output: true
}

func ExampleConfig_Path() {
	cfg := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/config.yaml")))
	core.Println(cfg.Path())
	// Output: /example/config.yaml
}

func ExampleConfig_Medium() {
	m := coreio.NewMockMedium()
	cfg := core.MustCast[*Config](New(WithMedium(m), WithPath("/example/config.yaml")))
	core.Println(cfg.Medium() == m)
	// Output: true
}

func ExampleConfig_MergeFrom() {
	base := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/base.yaml")))
	_ = base.Set("app.name", "base")
	layer := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/layer.yaml")))
	_ = layer.Set("dev.editor", "vim")
	base.MergeFrom(layer)
	var editor string
	_ = base.Get("dev.editor", &editor)
	core.Println(editor)
	// Output: vim
}

func ExampleConfig_OnChange() {
	cfg := core.MustCast[*Config](New(WithMedium(coreio.NewMockMedium()), WithPath("/example/config.yaml")))
	seen := ""
	cfg.OnChange(func(key string, value any) {
		seen = key + "=" + value.(string)
	})
	_ = cfg.Set("dev.editor", "vim")
	core.Println(seen)
	// Output: dev.editor=vim
}

func ExampleLoad() {
	m := coreio.NewMockMedium()
	_ = m.Write("/example/config.yaml", "app:\n  name: core\n")
	result := Load(m, "/example/config.yaml")
	data, _ := core.Cast[map[string]any](result)
	core.Println(result.OK, data["app"].(map[string]any)["name"])
	// Output: true core
}

func ExampleSave() {
	m := coreio.NewMockMedium()
	result := Save(m, "/example/config.yaml", map[string]any{"app": map[string]any{"name": "core"}})
	core.Println(result.OK && m.Exists("/example/config.yaml"))
	// Output: true
}
