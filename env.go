package config

import (
	"iter"
	"os"

	core "dappco.re/go/core"
)

// Env returns an iterator over environment variables with the given prefix,
// providing them as dot-notation keys and values.
//
// For example, with prefix "CORE_CONFIG_":
//
//	CORE_CONFIG_FOO_BAR=baz  -> yields ("foo.bar", "baz")
func Env(prefix string) iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for _, env := range os.Environ() {
			if !core.HasPrefix(env, prefix) {
				continue
			}

			parts := core.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}

			name := parts[0]
			value := parts[1]

			// Strip prefix and convert to dot notation
			key := core.TrimPrefix(name, prefix)
			key = core.Lower(key)
			key = core.Replace(key, "_", ".")

			if !yield(key, value) {
				return
			}
		}
	}
}

// LoadEnv parses environment variables with the given prefix and returns
// them as a flat map with dot-notation keys.
//
// Deprecated: Use Env for iterative access or collect into a map manually.
func LoadEnv(prefix string) map[string]any {
	result := make(map[string]any)
	for k, v := range Env(prefix) {
		result[k] = v
	}
	return result
}
