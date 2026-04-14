package config

import (
	"cmp"
	"iter"
	"os"
	"slices"

	core "dappco.re/go/core"
)

func normaliseEnvPrefix(prefix string) string {
	if prefix == "" || core.HasSuffix(prefix, "_") {
		return prefix
	}
	return prefix + "_"
}

// Env returns an iterator over environment variables with the given prefix,
// providing them as dot-notation keys and values.
//
// The prefix may be supplied with or without a trailing underscore.
//
// For example, with prefix "CORE_CONFIG_":
//
//	CORE_CONFIG_FOO_BAR=baz  -> yields ("foo.bar", "baz")
func Env(prefix string) iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		prefix = normaliseEnvPrefix(prefix)

		type entry struct {
			key   string
			value any
		}

		var entries []entry

		// os.Environ is the canonical way to walk every variable — core has no
		// equivalent enumerator, so this is a framework-boundary stdlib call.
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

			key := core.TrimPrefix(name, prefix)
			key = core.Lower(key)
			key = core.Replace(key, "_", ".")

			entries = append(entries, entry{key: key, value: value})
		}

		slices.SortFunc(entries, func(a, b entry) int {
			return cmp.Compare(a.key, b.key)
		})

		for _, entry := range entries {
			if !yield(entry.key, entry.value) {
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
