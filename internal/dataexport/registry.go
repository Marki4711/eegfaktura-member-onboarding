package dataexport

import (
	"fmt"
	"sort"
	"sync"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// Registry is a global, thread-safe Plugin registry. Plugins register
// themselves at init-time via Register; the registry is queried at runtime
// by the job service, worker, and HTTP handlers.
//
// Pattern follows database/sql.Driver — plugins live in their own packages
// and are pulled in via side-effect import in cmd/server/main.go.

var (
	registryMu sync.RWMutex
	registry   = map[string]Plugin{}
)

// Register adds a plugin to the global registry. Called from each
// plugin package's init() function. Panics on duplicate Type() to
// catch developer mistakes at startup.
func Register(p Plugin) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if p == nil {
		panic("dataexport: Register called with nil plugin")
	}
	if _, dup := registry[p.Type()]; dup {
		panic(fmt.Sprintf("dataexport: plugin type %q already registered", p.Type()))
	}
	registry[p.Type()] = p
}

// Get returns the plugin registered under the given type, or nil if none.
func Get(pluginType string) Plugin {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[pluginType]
}

// List returns all registered plugins, sorted by Type() for stable
// output to the admin UI.
func List() []Plugin {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Plugin, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type() < out[j].Type() })
	return out
}

// PluginInfos returns the publicly visible metadata for all registered
// plugins (used by the GET /plugins endpoint).
func PluginInfos() []shared.DataExportPluginInfo {
	plugins := List()
	out := make([]shared.DataExportPluginInfo, len(plugins))
	for i, p := range plugins {
		stds := p.StandardConfigs()
		configs := make([]shared.DataExportStandardConfigInfo, len(stds))
		for j, s := range stds {
			configs[j] = shared.DataExportStandardConfigInfo{
				Name:   s.Name,
				Config: s.Config,
			}
		}
		out[i] = shared.DataExportPluginInfo{
			Type:            p.Type(),
			DisplayName:     p.DisplayName(),
			StandardConfigs: configs,
		}
	}
	return out
}
