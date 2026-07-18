package plugin

import (
	"go.uber.org/fx"
)

// Module is the FX module for plugin infrastructure.
// Registers plugin Manager and Registry, and wires graceful shutdown.
var Module = fx.Module("plugin",
	fx.Provide(
		NewManager,
		NewRegistry,
	),
	fx.Invoke(func(lc fx.Lifecycle, mgr *Manager) {
		lc.Append(fx.StopHook(func() {
			mgr.StopAll()
		}))
	}),
)
