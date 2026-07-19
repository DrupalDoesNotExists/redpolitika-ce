// Package buildinfo holds version metadata injected at link time via -ldflags.
// Defaults are for local `go build` / `go test` without Makefile.
package buildinfo

// These variables are set by the linker, e.g.:
//
//	-X github.com/drupaldoesnotexists/redpolitika/ce/internal/buildinfo.Version=v0.1.0b
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
	License   = "BSL-1.1"
	Module    = "ce"
	Component = "redpolitika"
)
