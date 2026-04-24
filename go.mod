module dappco.re/go/config

go 1.26.0

require (
	dappco.re/go/core v0.8.0-alpha.1
	dappco.re/go/io v0.8.0-alpha.1
	dappco.re/go/log v0.8.0-alpha.1
	github.com/spf13/viper v1.21.0 // Note: multi-source config loader (YAML/JSON/env/flags); no core equivalent
	gopkg.in/yaml.v3 v3.0.1 // Note: YAML decoder; no core equivalent
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)
