package detect

// ExternalNode marks a detect method that delegates to an external plugin.
// The node is non-nil so Rule.DetectNode() returns it, but Detect() always
// returns nil — actual detection is done by the plugin process.
// ConfigJSON holds the serialized YAML args for the plugin.
type ExternalNode struct {
	PluginName string // plugin name (matches PluginInfo.Name)
	ConfigJSON string // JSON-encoded YAML args for the plugin
}

// Detect returns nil — detection is handled by the plugin.
func (n *ExternalNode) Detect(text string) []MatchRange { return nil }

// IsExternal checks if a Node is an ExternalNode (plugin-delegated).
func IsExternal(n Node) bool {
	_, ok := n.(*ExternalNode)
	return ok
}
