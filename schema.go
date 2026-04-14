package config

type ViewManifest struct {
	Title       string          `yaml:"title"`
	Height      int             `yaml:"height"`
	Width       int             `yaml:"width"`
	Resizable   bool            `yaml:"resizable"`
	Permissions ViewPermissions `yaml:"permissions"`
}

type ViewPermissions struct {
	Clipboard     bool `yaml:"clipboard"`
	Filesystem    bool `yaml:"filesystem"`
	Network       bool `yaml:"network"`
	Notifications bool `yaml:"notifications"`
	Camera        bool `yaml:"camera"`
	Microphone    bool `yaml:"microphone"`
}

type BuildManifest struct {
	Targets []BuildTarget `yaml:"targets"`
	Output  string        `yaml:"output"`
	Flags   []string      `yaml:"flags"`
	LDFlags string        `yaml:"ldflags"`
	CGO     bool          `yaml:"cgo"`
}

type BuildTarget struct {
	OS   string `yaml:"os"`
	Arch string `yaml:"arch"`
}

type PackageManifest struct {
	Name         string   `yaml:"name"`
	Module       string   `yaml:"module"`
	Version      string   `yaml:"version"`
	Description  string   `yaml:"description"`
	Licence      string   `yaml:"licence"`
	Dependencies []string `yaml:"dependencies"`
	Tags         []string `yaml:"tags"`
}
