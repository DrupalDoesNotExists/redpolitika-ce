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
	for _, dir := range layers {
		rules, hash, err := l.loadDir(dir)
		if err != nil {
			return nil, &Error{Op: "LoadAll", Message: fmt.Sprintf("load %s", dir), Err: err}
		}

		rs := model.NewRuleSet(rules, hash)

		if merged == nil {
			merged = rs
		} else {
			merged = merged.Merge(rs)
		}
	}

	return merged, nil
}

// loadDir reads all .yaml and .yml files in a directory and returns parsed rules.
func (l *Loader) loadDir(dir string) ([]*model.Rule, uint64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, fmt.Errorf("load rules dir %s: %w", dir, err)
	}

	var allRules []*model.Rule
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
			return nil, 0, &Error{Op: "loadDir", Message: fmt.Sprintf("read %s", entry.Name()), Err: err}
		}

		rules, err := ParseYAML(data)
		if err != nil {
			return nil, 0, &Error{Op: "loadDir", Message: fmt.Sprintf("parse %s", entry.Name()), Err: err}
		}

		// Accumulate hash for this layer
		layerHash = hashFile(layerHash, data)
		allRules = append(allRules, rules...)
	}

	// Sort rules by ID for deterministic ordering
	sort.Slice(allRules, func(i, j int) bool {
		return allRules[i].ID().Value() < allRules[j].ID().Value()
	})

	return allRules, layerHash, nil
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

// Watch returns a reload notification channel (stub — A25).
// Returns a closed channel so early readers don't block.
// Use fsnotify when hot-reload is needed.
func (l *Loader) Watch(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
