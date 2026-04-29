package config

import (
	"context"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func exampleService() (*Service, *coreio.MockMedium) {
	m := coreio.NewMockMedium()
	path := "/example/.core/config.yaml"
	_ = m.Write(path, "app:\n  name: service\n")
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{
			Path:   path,
			Medium: m,
		}),
	}
	_ = svc.OnStartup(context.Background())
	return svc, m
}

func ExampleNewConfigService() {
	result := NewConfigService(core.New())
	core.Println(result.OK)
	// Output: true
}

func ExampleNewConfigServiceWith() {
	factory := NewConfigServiceWith(ServiceOptions{Path: "/example/.core/config.yaml"})
	result := factory(core.New())
	core.Println(result.OK)
	// Output: true
}

func ExampleService_OnStartup() {
	svc, _ := exampleService()
	core.Println(svc.Config() != nil)
	// Output: true
}

func ExampleService_OnShutdown() {
	svc, _ := exampleService()
	result := svc.OnShutdown(context.Background())
	core.Println(result.OK)
	// Output: true
}

func ExampleService_Get() {
	svc, _ := exampleService()
	var name string
	err := resultError(svc.Get("app.name", &name))
	core.Println(err == nil, name)
	// Output: true service
}

func ExampleService_Set() {
	svc, _ := exampleService()
	err := resultError(svc.Set("dev.editor", "vim"))
	var editor string
	_ = svc.Get("dev.editor", &editor)
	core.Println(err == nil, editor)
	// Output: true vim
}

func ExampleService_Commit() {
	svc, m := exampleService()
	_ = svc.Set("dev.editor", "vim")
	err := resultError(svc.Commit())
	core.Println(err == nil && m.Exists("/example/.core/config.yaml"))
	// Output: true
}

func ExampleService_LoadFile() {
	svc, m := exampleService()
	_ = m.Write("/example/.core/extra.yaml", "dev:\n  shell: zsh\n")
	err := resultError(svc.LoadFile(m, ".core/extra.yaml"))
	var shell string
	_ = svc.Get("dev.shell", &shell)
	core.Println(err == nil, shell)
	// Output: true zsh
}

func ExampleService_Config() {
	svc, _ := exampleService()
	core.Println(svc.Config().Path())
	// Output: /example/.core/config.yaml
}
