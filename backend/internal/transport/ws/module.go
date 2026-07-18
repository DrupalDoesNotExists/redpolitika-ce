package ws

import "go.uber.org/fx"

// Module is the FX module for the WebSocket layer.
var Module = fx.Module("ws",
	fx.Provide(
		NewHub,
		NewLiveHandler,
	),
)
