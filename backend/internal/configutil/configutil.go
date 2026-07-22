// Package configutil provides MarshalConfig — serializes YAML method args to JSON.
// Extracted to avoid import cycles between domain packages (detect/fix/model).
package configutil

import "encoding/json"

// MarshalConfig serializes YAML method args to JSON string for plugin transport.
// Empty args → empty string.
func MarshalConfig(args map[string]interface{}) string {
	if len(args) == 0 {
		return ""
	}
	b, _ := json.Marshal(args)
	return string(b)
}
