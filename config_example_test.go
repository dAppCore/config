package config_test

import (
	. "dappco.re/go"
	config "dappco.re/go/config"
)

func exampleConfigMedium(prefix string) (*Fs, string, func()) {
	root := (&Fs{}).New("/")
	dir := root.TempDir(prefix)
	resolved := PathEvalSymlinks(dir)
	if resolved.OK {
		dir = resolved.Value.(string)
	}
	return (&Fs{}).New(dir), "/config.yaml", func() {
		root.DeleteAll(dir)
	}
}

func ExampleWithMedium() {
	fs, path, cleanup := exampleConfigMedium("go-config-medium")
	defer cleanup()
	fs.Write(path, "agent: codex\n")

	r := config.New(config.WithMedium(fs), config.WithPath(path))
	cfg := r.Value.(*config.Config)
	var agent string
	cfg.Get("agent", &agent)

	Println(agent)
	// Output: codex
}

func ExampleWithPath() {
	fs, _, cleanup := exampleConfigMedium("go-config-path")
	defer cleanup()

	r := config.New(config.WithMedium(fs), config.WithPath("/custom.yaml"))
	cfg := r.Value.(*config.Config)

	Println(cfg.Path())
	// Output: /custom.yaml
}

func ExampleWithEnvPrefix() {
	Setenv("APP_MODE", "test")
	defer Unsetenv("APP_MODE")
	fs, path, cleanup := exampleConfigMedium("go-config-env-prefix")
	defer cleanup()

	r := config.New(config.WithMedium(fs), config.WithPath(path), config.WithEnvPrefix("APP"))
	cfg := r.Value.(*config.Config)
	var mode string
	cfg.Get("mode", &mode)

	Println(mode)
	// Output: test
}

func ExampleNew() {
	fs, path, cleanup := exampleConfigMedium("go-config-new")
	defer cleanup()

	r := config.New(config.WithMedium(fs), config.WithPath(path))

	Println(r.OK)
	// Output: true
}

func ExampleConfig_LoadFile() {
	fs, path, cleanup := exampleConfigMedium("go-config-loadfile")
	defer cleanup()
	fs.Write("/extra.yaml", "agent: codex\n")
	cfg := config.New(config.WithMedium(fs), config.WithPath(path)).Value.(*config.Config)

	r := cfg.LoadFile(fs, "/extra.yaml")
	var agent string
	cfg.Get("agent", &agent)

	Println(r.OK)
	Println(agent)
	// Output:
	// true
	// codex
}

func ExampleConfig_Get() {
	fs, path, cleanup := exampleConfigMedium("go-config-get")
	defer cleanup()
	cfg := config.New(config.WithMedium(fs), config.WithPath(path)).Value.(*config.Config)
	cfg.Set("agent", "codex")

	var agent string
	r := cfg.Get("agent", &agent)

	Println(r.OK)
	Println(agent)
	// Output:
	// true
	// codex
}

func ExampleConfig_Set() {
	fs, path, cleanup := exampleConfigMedium("go-config-set")
	defer cleanup()
	cfg := config.New(config.WithMedium(fs), config.WithPath(path)).Value.(*config.Config)

	r := cfg.Set("agent", "codex")
	var agent string
	cfg.Get("agent", &agent)

	Println(r.OK)
	Println(agent)
	// Output:
	// true
	// codex
}

func ExampleConfig_Commit() {
	fs, path, cleanup := exampleConfigMedium("go-config-commit")
	defer cleanup()
	cfg := config.New(config.WithMedium(fs), config.WithPath(path)).Value.(*config.Config)
	cfg.Set("agent", "codex")

	r := cfg.Commit()
	content := fs.Read(path)

	Println(r.OK)
	Println(Contains(content.Value.(string), "agent: codex"))
	// Output:
	// true
	// true
}

func ExampleConfig_All() {
	fs, path, cleanup := exampleConfigMedium("go-config-all")
	defer cleanup()
	cfg := config.New(config.WithMedium(fs), config.WithPath(path)).Value.(*config.Config)
	cfg.Set("zulu", "last")
	cfg.Set("alpha", "first")

	for key, value := range cfg.All() {
		Println(key, value)
	}
	// Output:
	// alpha first
	// zulu last
}

func ExampleConfig_Path() {
	fs, _, cleanup := exampleConfigMedium("go-config-path-method")
	defer cleanup()
	cfg := config.New(config.WithMedium(fs), config.WithPath("/settings.yaml")).Value.(*config.Config)

	Println(cfg.Path())
	// Output: /settings.yaml
}

func ExampleLoad() {
	fs, _, cleanup := exampleConfigMedium("go-config-load")
	defer cleanup()
	fs.Write("/config.yaml", "agent: codex\n")

	r := config.Load(fs, "/config.yaml")
	data := r.Value.(map[string]any)

	Println(data["agent"])
	// Output: codex
}

func ExampleSave() {
	fs, _, cleanup := exampleConfigMedium("go-config-save")
	defer cleanup()

	r := config.Save(fs, "/config.yaml", map[string]any{"agent": "codex"})
	content := fs.Read("/config.yaml")

	Println(r.OK)
	Println(Contains(content.Value.(string), "agent: codex"))
	// Output:
	// true
	// true
}
