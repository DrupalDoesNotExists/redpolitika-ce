// Package rules implements YAML loading, parsing, merging, and persistence for RuleSet (A5/A24).
package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

// Loader reads rule YAML files from disk with layer support.
// Layers (A24): base/ → project/ → override/ — each later layer overrides the previous.
type Loader struct {
	baseDir     string
	projectDir  string
	overrideDir string
}

// NewLoader creates a rules Loader.
// baseDir: required, contains base rule YAML files.
// projectDir: optional, project-specific overrides.
// overrideDir: optional, per-environment overrides.
func NewLoader(baseDir, projectDir, overrideDir string) *Loader {
	return &Loader{
		baseDir:     baseDir,
		projectDir:  projectDir,
		overrideDir: overrideDir,
	}
}

// LoadAll reads all rules from the layered directories and returns a merged RuleSet.
// Layer order: base → project → override (A24).
func (l *Loader) LoadAll(ctx context.Context) (*model.RuleSet, error) {
	var layers []string

	if l.baseDir != "" {
		layers = append(layers, l.baseDir)
	}
	if l.projectDir != "" {
		layers = append(layers, l.projectDir)
	}
	if l.overrideDir != "" {
		layers = append(layers, l.overrideDir)
	}

	if len(layers) == 0 {
		return nil, &Error{Op: "LoadAll", Message: "no rule directories configured", Err: nil}
	}

	var merged *model.RuleSet
	var allRefs []unresolvedRef

	for _, dir := range layers {
		rules, hash, refs, err := l.loadDir(dir)
		if err != nil {
			return nil, &Error{Op: "LoadAll", Message: fmt.Sprintf("load %s", dir), Err: err}
		}
		allRefs = append(allRefs, refs...)

		rs := model.NewRuleSet(rules, hash)

		if merged == nil {
			merged = rs
		} else {
			merged = merged.Merge(rs)
		}
	}

	// Resolve refs after all layers are merged
	if len(allRefs) > 0 {
		if err := resolveRefs(allRefs, merged); err != nil {
			return nil, &Error{Op: "LoadAll", Message: "resolve refs", Err: err}
		}
	}

	return merged, nil
}

// loadDir reads all .yaml and .yml files in a directory and returns parsed rules.
// Missing directory is treated as an empty layer (no error) so the image can
// start without mounted rules.
func (l *Loader) loadDir(dir string) ([]*model.Rule, uint64, []unresolvedRef, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil, nil
		}
		return nil, 0, nil, fmt.Errorf("load rules dir %s: %w", dir, err)
	}

	var allRules []*model.Rule
	var allRefs []unresolvedRef
	layerHash := uint64(0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, 0, nil, &Error{Op: "loadDir", Message: fmt.Sprintf("read %s", entry.Name()), Err: err}
		}

		result, err := ParseYAML(data)
		if err != nil {
			return nil, 0, nil, &Error{Op: "loadDir", Message: fmt.Sprintf("parse %s", entry.Name()), Err: err}
		}

		// Accumulate hash for this layer
		layerHash = hashFile(layerHash, data)
		allRules = append(allRules, result.Rules...)
		allRefs = append(allRefs, result.Refs...)
	}

	// Sort rules by ID for deterministic ordering
	sort.Slice(allRules, func(i, j int) bool {
		return allRules[i].ID().Value() < allRules[j].ID().Value()
	})

	return allRules, layerHash, allRefs, nil
}

// hashFile combines an existing hash with file content hash.
func hashFile(current uint64, data []byte) uint64 {
	// FNV-1a 64 mixing
	h := current
	for _, b := range data {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}

// resolveRefs resolves all RefNode refs after all rules are loaded and merged.
// Performs cycle detection via DFS (3-color) before resolution.
func resolveRefs(refs []unresolvedRef, rs *model.RuleSet) error {
	// 1. Build rule lookup by ID
	rulesByID := make(map[string]*model.Rule)
	for _, r := range rs.Rules() {
		rulesByID[r.ID().Value()] = r
	}

	// 2. Build dependency graph for cycle detection
	graph := make(map[string][]string) // ruleID → refs to other rules
	for _, r := range refs {
		graph[r.RuleID] = append(graph[r.RuleID], r.Node.RefID())
	}

	// 3. Cycle detection via DFS (3-color)
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	var dfs func(node string) error
	dfs = func(node string) error {
		color[node] = gray
		for _, neighbor := range graph[node] {
			if color[neighbor] == gray {
				return fmt.Errorf("cycle detected: %s → %s", node, neighbor)
			}
			if color[neighbor] == white {
				if err := dfs(neighbor); err != nil {
					return err
				}
			}
		}
		color[node] = black
		return nil
	}
	for _, r := range refs {
		if color[r.RuleID] == white {
			if err := dfs(r.RuleID); err != nil {
				return err
			}
		}
	}

	// 4. Resolve each ref
	for _, r := range refs {
		targetRule, ok := rulesByID[r.Node.RefID()]
		if !ok {
			return fmt.Errorf("ref %q: rule %q not found", r.RuleID, r.Node.RefID())
		}
		targetNode := targetRule.DetectNode()
		if targetNode == nil {
			return fmt.Errorf("ref %q: rule %q has no detect tree (server-side method?)", r.RuleID, r.Node.RefID())
		}
		r.Node.Resolve(targetNode)
	}

	return nil
}

// Watch returns a reload notification channel (stub — A25).
// Returns a closed channel so early readers don't block.
// Use fsnotify when hot-reload is needed.
func (l *Loader) Watch(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
