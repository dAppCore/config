package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func ExampleFeature() {
	resetFeatureRegistry()
	defer resetFeatureRegistry()
	SetFeature("dark-mode", true)
	core.Println(Feature("dark-mode"))
	// Output: true
}

func ExampleFeatureFromConfig() {
	cfg, _ := configResult(New(
		WithMedium(coreio.NewMockMedium()),
		WithPath("/example/config.yaml"),
		WithDefaults(map[string]any{"features.dark-mode": true}),
	))
	core.Println(FeatureFromConfig(cfg, "dark-mode"))
	// Output: true
}

func ExampleSetFeatureSource() {
	resetFeatureRegistry()
	defer resetFeatureRegistry()
	cfg, _ := configResult(New(
		WithMedium(coreio.NewMockMedium()),
		WithPath("/example/config.yaml"),
		WithDefaults(map[string]any{"features.beta": true}),
	))
	SetFeatureSource(cfg)
	core.Println(Feature("beta"))
	// Output: true
}

func ExampleSetFeature() {
	resetFeatureRegistry()
	defer resetFeatureRegistry()
	SetFeature("beta", true)
	core.Println(Feature("beta"))
	// Output: true
}

func ExampleFeatures() {
	resetFeatureRegistry()
	defer resetFeatureRegistry()
	SetFeature("alpha", true)
	SetFeature("beta", false)
	core.Println(Features()[0])
	// Output: alpha
}
