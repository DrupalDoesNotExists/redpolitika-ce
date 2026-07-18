package rules

import (
	"gopkg.in/yaml.v3"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/detect"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/fix"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

// ---------------------------------------------------------------------------
//  YAML types
// ---------------------------------------------------------------------------

// RuleYAML is the YAML representation of a single rule (§9).
type RuleYAML struct {
	ID       string       `yaml:"id"`
	Severity int          `yaml:"severity"`
	Category string       `yaml:"category"`
	Enabled  *bool        `yaml:"enabled,omitempty"`
	Detect   DetectNodeYAML `yaml:"detect"`
	Fix      FixNodeYAML  `yaml:"fix,omitempty"`
	Disable  bool         `yaml:"disable,omitempty"`
	Name     string       `yaml:"name,omitempty"`
	URL      string       `yaml:"url,omitempty"`
	Examples ExamplesYAML `yaml:"examples,omitempty"`
	Related  []RelatedYAML `yaml:"related,omitempty"`
}

// ExamplesYAML is the examples block in YAML.
type ExamplesYAML struct {
	Bad  []string `yaml:"bad,omitempty"`
	Good []string `yaml:"good,omitempty"`
}

// RelatedYAML is a related link in YAML.
type RelatedYAML struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
}

// DetectNodeYAML is a node in the detection method tree.
// Supports both old flat format (method + inline fields) and new nested format (method + args).
type DetectNodeYAML struct {
	Method string          `yaml:"method"`
	Args   []DetectNodeYAML `yaml:"args,omitempty"`

	// Inline fields — backward compatibility with flat format
	Pattern       string   `yaml:"pattern,omitempty"`
	Words         []string `yaml:"words,omitempty"`
	List          []string `yaml:"list,omitempty"`
	CaseSensitive bool     `yaml:"case_sensitive,omitempty"`
	Value         string   `yaml:"value,omitempty"`
	Left          string   `yaml:"left,omitempty"`
	Right         string   `yaml:"right,omitempty"`
	Min           int      `yaml:"min,omitempty"`
	Max           int      `yaml:"max,omitempty"`
	Mode          string   `yaml:"mode,omitempty"`
	From          int      `yaml:"from,omitempty"`
	To            int      `yaml:"to,omitempty"`
}

// FixNodeYAML is a node in the fix method tree.
type FixNodeYAML struct {
	Method string        `yaml:"method"`
	Args   []FixNodeYAML `yaml:"args,omitempty"`

	// Inline fields — backward compatibility
	With        string  `yaml:"with,omitempty"`
	Pattern     string  `yaml:"pattern,omitempty"`
	Replacement string  `yaml:"replacement,omitempty"`
	Prefix      string  `yaml:"prefix,omitempty"`
	Suffix      string  `yaml:"suffix,omitempty"`
	Suggestion  string  `yaml:"suggestion,omitempty"`
	Autofix     *string `yaml:"autofix,omitempty"`
}

// RulesFile wraps the top-level "rules" key in a YAML file.
type RulesFile struct {
	Rules []RuleYAML `yaml:"rules"`
}

// ---------------------------------------------------------------------------
//  Helpers: YAML node → args map for registry builders
// ---------------------------------------------------------------------------

func (n *DetectNodeYAML) buildArgs() map[string]interface{} {
	args := make(map[string]interface{})
	if n.Pattern != "" {
		args["pattern"] = n.Pattern
	}
	if len(n.Words) > 0 {
		args["words"] = n.Words
	}
	if len(n.List) > 0 {
		args["list"] = n.List
	}
	if n.CaseSensitive {
		args["case_sensitive"] = true
	}
	if n.Value != "" {
		args["value"] = n.Value
	}
	if n.Left != "" {
		args["left"] = n.Left
	}
	if n.Right != "" {
		args["right"] = n.Right
	}
	if n.Min > 0 {
		args["min"] = n.Min
	}
	if n.Max > 0 {
		args["max"] = n.Max
	}
	if n.Mode != "" {
		args["mode"] = n.Mode
	}
	if n.From > 0 {
		args["from"] = n.From
	}
	if n.To > 0 {
		args["to"] = n.To
	}
	return args
}

func (n *FixNodeYAML) buildArgs() map[string]interface{} {
	args := make(map[string]interface{})
	if n.With != "" {
		args["with"] = n.With
	}
	if n.Pattern != "" {
		args["pattern"] = n.Pattern
	}
	if n.Replacement != "" {
		args["replacement"] = n.Replacement
	}
	if n.Prefix != "" {
		args["prefix"] = n.Prefix
	}
	if n.Suffix != "" {
		args["suffix"] = n.Suffix
	}
	return args
}

// ---------------------------------------------------------------------------
//  Build detect/fix trees
// ---------------------------------------------------------------------------

// buildDetectNode recursively builds a detect tree from YAML node.
// Returns (detectNode, detectMethod, error).
// For server-side methods (llm/plugin), detectNode is nil.
func buildDetectNode(ny *DetectNodeYAML) (detect.Node, string, error) {
	method := ny.Method

	// Backward compat: infer method from fields
	if method == "" && len(ny.Words) > 0 {
		method = "wordlist"
	}
	if method == "" && len(ny.List) > 0 {
		method = "wordlist"
	}
	if method == "" && ny.Pattern != "" {
		method = "regex"
	}

	// Server-side methods return nil node
	switch method {
	case "llm", "plugin", "ner", "pos", "function", "expr":
		return nil, method, nil
	}

	if method == "" {
		// No method and no inferrable fields — treat as noop (returns nil)
		return nil, "", nil
	}

	// Build child nodes
	children := make([]detect.Node, 0, len(ny.Args))
	for i := range ny.Args {
		child, _, err := buildDetectNode(&ny.Args[i])
		if err != nil {
			return nil, "", &Error{Op: "buildDetectNode", Message: "build child " + ny.Args[i].Method, Err: err}
		}
		if child != nil {
			children = append(children, child)
		}
	}

	args := ny.buildArgs()
	node, err := detect.Build(method, args, children)
	if err != nil {
		return nil, "", &Error{Op: "buildDetectNode", Message: "build " + method, Err: err}
	}

	return node, method, nil
}

// buildFixNode builds a fix tree from YAML node.
// Returns nil when no fix method is specified.
func buildFixNode(ny *FixNodeYAML) (fix.Node, error) {
	method := ny.Method

	// Backward compat: old format used autofix field directly
	if method == "" && ny.Autofix != nil {
		// autofix: nil → no fix, &"" → delete, &"text" → replace
		with := *ny.Autofix
		return &fix.ReplaceNode{With: with}, nil
	}

	// Backward compat: old FixYAML used autofix/suggestion fields
	// Old "fix:" with just "suggestion" and no method = no fix tree
	if method == "" {
		// If no method and no args, no fix tree
		if ny.With == "" && ny.Pattern == "" {
			return nil, nil
		}
		// Inline with="" means "replace with empty" (remove)
		// Inline with="text" means "replace with text"
		// Set method to "replace" for the with field
		method = "replace"
	}

	// Build child nodes
	children := make([]fix.Node, 0, len(ny.Args))
	for i := range ny.Args {
		child, err := buildFixNode(&ny.Args[i])
		if err != nil {
			return nil, &Error{Op: "buildFixNode", Message: "build child " + ny.Args[i].Method, Err: err}
		}
		if child != nil {
			children = append(children, child)
		}
	}

	args := ny.buildArgs()
	return fix.Build(method, args, children)
}

// ---------------------------------------------------------------------------
//  ParseYAML
// ---------------------------------------------------------------------------

// ParseYAML parses raw YAML bytes into domain Rule objects.
// Supports both old flat format and new nested method tree format (SPEC §8, A26/Q33).
func ParseYAML(data []byte) ([]*model.Rule, error) {
	var rf RulesFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, &Error{Op: "ParseYAML", Message: "invalid YAML", Err: err}
	}

	var rules []*model.Rule
	for _, ry := range rf.Rules {
		rule, err := ruleFromYAML(ry)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// ruleFromYAML converts a RuleYAML to a domain Rule.
func ruleFromYAML(ry RuleYAML) (*model.Rule, error) {
	// Handle disable-only overrides
	if ry.Disable {
		if _, err := model.RuleIDFromString(ry.ID); err != nil {
			return nil, &Error{Op: "ParseYAML", Message: "disable without valid id: " + ry.ID, Err: err}
		}
		rule, err := model.NewRule(ry.ID, 5, "cleanliness", "regex", nil, nil, "", "", "", model.Examples{}, nil)
		if err != nil {
			return nil, &Error{Op: "ParseYAML", Message: "create disabled rule stub", Err: err}
		}
		return rule.Disable(), nil
	}

	// Build detect tree
	detectNode, detectMethod, err := buildDetectNode(&ry.Detect)
	if err != nil {
		return nil, &Error{Op: "ParseYAML", Message: "build detect for " + ry.ID, Err: err}
	}

	// Build fix tree
	var fixNode fix.Node
	if ry.Fix.Method != "" || ry.Fix.With != "" || ry.Fix.Pattern != "" || ry.Fix.Autofix != nil {
		fixNode, err = buildFixNode(&ry.Fix)
		if err != nil {
			return nil, &Error{Op: "ParseYAML", Message: "build fix for " + ry.ID, Err: err}
		}
	}

	examples := model.Examples{
		Bad:  ry.Examples.Bad,
		Good: ry.Examples.Good,
	}
	var related []model.Related
	for _, r := range ry.Related {
		related = append(related, model.Related{Name: r.Name, URL: r.URL})
	}

	rule, err := model.NewRule(
		ry.ID, ry.Severity, ry.Category,
		detectMethod, detectNode, fixNode,
		ry.Fix.Suggestion, ry.Name, ry.URL,
		examples, related,
	)
	if err != nil {
		return nil, &Error{Op: "ParseYAML", Message: "create rule " + ry.ID, Err: err}
	}

	// Apply enabled override (default true)
	if ry.Enabled != nil && !*ry.Enabled {
		rule = rule.Disable()
	}

	return rule, nil
}
