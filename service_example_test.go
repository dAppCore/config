package config_test

import (
	. "dappco.re/go"
	config "dappco.re/go/config"
)

func ExampleNewConfigService() {
	r := config.NewConfigService(New())
	svc := r.Value.(*config.Service)

	Println(r.OK)
	Println(svc != nil)
	// Output:
	// true
	// true
}

func ExampleService_OnStartup() {
	fs, path, cleanup := exampleConfigMedium("go-config-service-startup")
	defer cleanup()
	_ = fs.Write(path, "agent: service\n")
	svc := &config.Service{ServiceRuntime: NewServiceRuntime(nil, config.ServiceOptions{Path: path, Medium: fs})}

	r := svc.OnStartup(Background())
	var agent string
	_ = svc.Get("agent", &agent)

	Println(r.OK)
	Println(agent)
	// Output:
	// true
	// service
}

func ExampleService_Get() {
	fs, path, cleanup := exampleConfigMedium("go-config-service-get")
	defer cleanup()
	svc := &config.Service{ServiceRuntime: NewServiceRuntime(nil, config.ServiceOptions{Path: path, Medium: fs})}
	_ = svc.OnStartup(Background())
	_ = svc.Set("agent", "codex")

	var agent string
	r := svc.Get("agent", &agent)

	Println(r.OK)
	Println(agent)
	// Output:
	// true
	// codex
}

func ExampleService_Set() {
	fs, path, cleanup := exampleConfigMedium("go-config-service-set")
	defer cleanup()
	svc := &config.Service{ServiceRuntime: NewServiceRuntime(nil, config.ServiceOptions{Path: path, Medium: fs})}
	_ = svc.OnStartup(Background())

	r := svc.Set("agent", "codex")
	var agent string
	_ = svc.Get("agent", &agent)

	Println(r.OK)
	Println(agent)
	// Output:
	// true
	// codex
}

func ExampleService_Commit() {
	fs, path, cleanup := exampleConfigMedium("go-config-service-commit")
	defer cleanup()
	svc := &config.Service{ServiceRuntime: NewServiceRuntime(nil, config.ServiceOptions{Path: path, Medium: fs})}
	_ = svc.OnStartup(Background())
	_ = svc.Set("agent", "codex")

	r := svc.Commit()
	content := fs.Read(path)

	Println(r.OK)
	Println(Contains(content.Value.(string), testAgentCodexText))
	// Output:
	// true
	// true
}

func ExampleService_LoadFile() {
	fs, path, cleanup := exampleConfigMedium("go-config-service-loadfile")
	defer cleanup()
	_ = fs.Write(testExtraYAMLPath, testAgentCodexYAML)
	svc := &config.Service{ServiceRuntime: NewServiceRuntime(nil, config.ServiceOptions{Path: path, Medium: fs})}
	_ = svc.OnStartup(Background())

	r := svc.LoadFile(fs, testExtraYAMLPath)
	var agent string
	_ = svc.Get("agent", &agent)

	Println(r.OK)
	Println(agent)
	// Output:
	// true
	// codex
}
