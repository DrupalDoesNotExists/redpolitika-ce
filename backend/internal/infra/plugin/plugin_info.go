package plugin

import "google.golang.org/grpc"

// PluginInfo holds metadata about a running plugin.
// No archetypes (A15) — a plugin is defined by its capabilities, not its type.
type PluginInfo struct {
	Name         string
	Conn         *grpc.ClientConn
	Capabilities []string
	Methods      []string
}
