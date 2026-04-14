package config

import (
	"sync"

	core "dappco.re/go/core"
	coreerr "dappco.re/go/core/log"
)

// conclaveRootFn is swapped in by go-session or a similar session-scoped
// storage provider. Until set, ForConclave falls back to ~/.core/conclaves/{name}.
var (
	conclaveMu   sync.RWMutex
	conclaveRoot = defaultConclaveRoot
)

// ConclaveRootFunc resolves the on-disk root directory for a Conclave by name.
//
//	config.SetConclaveRootFunc(func(name string) (string, error) {
//	    return session.ConclaveRoot(name)
//	})
type ConclaveRootFunc func(name string) (string, error)

// SetConclaveRootFunc installs a resolver that maps a Conclave name to its
// on-disk root directory. Passing nil restores the default resolver.
//
//	config.SetConclaveRootFunc(resolver)
func SetConclaveRootFunc(fn ConclaveRootFunc) {
	conclaveMu.Lock()
	defer conclaveMu.Unlock()
	if fn == nil {
		conclaveRoot = defaultConclaveRoot
		return
	}
	conclaveRoot = fn
}

// ForConclave returns a Config scoped to the named Conclave. The returned
// config inherits from the parent (project walking up from cwd, then the
// user-global ~/.core/) and overrides with values found in the Conclave's own
// `.core/` directory. Resolution precedence from highest to lowest:
//  1. Conclave `{root}/.core/config.yaml`
//  2. Project `.core/config.yaml` (and ancestors up to repo boundary)
//  3. User-global `~/.core/config.yaml`
//
//	alpha, _ := config.ForConclave("workspace-alpha")
//	alpha.Get("theme", &theme)
func ForConclave(name string, opts ...Option) (*Config, error) {
	conclaveMu.RLock()
	resolver := conclaveRoot
	conclaveMu.RUnlock()

	root, err := resolver(name)
	if err != nil {
		return nil, coreerr.E("config.ForConclave", "failed to resolve conclave root: "+name, err)
	}

	conclaveOpts := append([]Option{}, opts...)
	conclaveOpts = append(conclaveOpts, WithPath(core.Path(root, ".core", "config.yaml")))

	// Project + global inheritance is discovered from the current working dir,
	// not the conclave root — the conclave usually sits outside the project
	// tree (e.g. under XDG config/conclaves/). Discover() handles ~/.core/ as
	// the final fallback layer.
	base, err := Discover(opts...)
	if err != nil {
		return nil, coreerr.E("config.ForConclave", "failed to discover base config: "+name, err)
	}

	conclaveCfg, err := New(conclaveOpts...)
	if err != nil {
		return nil, coreerr.E("config.ForConclave", "failed to load conclave config: "+name, err)
	}

	// Conclave wins over base — MergeFrom only fills gaps.
	conclaveCfg.MergeFrom(base)
	return conclaveCfg, nil
}

func defaultConclaveRoot(name string) (string, error) {
	return core.Path(XDG().Config(), "conclaves", name), nil
}
