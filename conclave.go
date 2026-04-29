package config

import (
	"sync"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// conclaveRootFn is swapped in by go-session or a similar session-scoped
// storage provider. Until set, ForConclave falls back to ~/.core/conclaves/{name}.
var (
	conclaveMu   sync.RWMutex
	conclaveRoot = defaultConclaveRoot
)

const callerForConclave = "config.ForConclave"

// ConclaveRootFunc resolves the on-disk root directory for a Conclave by name.
//
//	config.SetConclaveRootFunc(func(name string) core.Result {
//	    return core.Ok(session.ConclaveRoot(name))
//	})
type ConclaveRootFunc func(name string) core.Result

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
//
//  1. Conclave `{root}/.core/config.yaml`
//
//  2. Project `.core/config.yaml` (and ancestors up to repo boundary)
//
//  3. User-global `~/.core/config.yaml`
//
//     alpha, _ := config.ForConclave("workspace-alpha")
//     alpha.Get("theme", &theme)
func ForConclave(name string, opts ...Option) core.Result {
	conclaveMu.RLock()
	resolver := conclaveRoot
	conclaveMu.RUnlock()

	rootResult := resolver(name)
	if !rootResult.OK {
		return core.Fail(coreerr.E(callerForConclave, "failed to resolve conclave root: "+name, resultCause(rootResult).(error)))
	}
	root := rootResult.Value.(string)
	if root == "" {
		return core.Fail(coreerr.E(callerForConclave, "failed to resolve conclave root: "+name, nil))
	}
	if isSymlinkedCoreDir(coreio.Local, core.Path(root, ".core")) {
		return core.Fail(coreerr.E(callerForConclave, "symlinked conclave .core directory rejected: "+root, nil))
	}

	conclaveOpts := append([]Option{}, opts...)
	conclaveOpts = append(conclaveOpts, WithPath(core.Path(root, ".core", "config.yaml")))

	// Project + global inheritance is discovered from the current working dir,
	// not the conclave root — the conclave usually sits outside the project
	// tree (e.g. under XDG config/conclaves/). Discover() handles ~/.core/ as
	// the final fallback layer.
	baseResult := Discover(opts...)
	if !baseResult.OK {
		return core.Fail(coreerr.E(callerForConclave, "failed to discover base config: "+name, resultCause(baseResult).(error)))
	}
	base := baseResult.Value.(*Config)

	conclaveResult := New(conclaveOpts...)
	if !conclaveResult.OK {
		return core.Fail(coreerr.E(callerForConclave, "failed to load conclave config: "+name, resultCause(conclaveResult).(error)))
	}
	conclaveCfg := conclaveResult.Value.(*Config)

	// Conclave wins over base — MergeFrom only fills gaps.
	conclaveCfg.MergeFrom(base)
	return core.Ok(conclaveCfg)
}

func defaultConclaveRoot(name string) core.Result {
	if !isSafePathElement(name) {
		return core.Fail(coreerr.E("config.defaultConclaveRoot", "invalid conclave name: "+name, nil))
	}
	return core.Ok(core.Path(XDG().Config(), "conclaves", name))
}
