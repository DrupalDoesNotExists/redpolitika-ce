package fix

// ExternalFixNode marks a fix method that delegates to an external plugin.
// Fix() returns the matchStr unchanged — actual fix is done by the plugin
// and called from the usecase via FixFunctionProvider.
// ConfigJSON holds the serialized YAML args for the plugin.
type ExternalFixNode struct {
	PluginName string // plugin name (matches PluginInfo.Name)
	MethodName string // method name registered by the plugin
	ConfigJSON string // JSON-encoded YAML args for the plugin
}

// Fix returns matchStr unchanged — plugin handles the actual fix.
// The usecase calls FixFunctionProvider after detection.
func (n *ExternalFixNode) Fix(matchStr string, ctx Context) string { return matchStr }

// IsExternalFix checks if a Node is an ExternalFixNode (plugin-delegated).
func IsExternalFix(n Node) bool {
	_, ok := n.(*ExternalFixNode)
	return ok
}
