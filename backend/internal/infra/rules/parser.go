package rules

import (
	"fmt"
	"strings"

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
	ID       string         `yaml:"id"`
	Severity int            `yaml:"severity"`
	Priority int            `yaml:"priority,omitempty"`
	Category string         `yaml:"category"`
	Enabled  *bool          `yaml:"enabled,omitempty"`
	Detect   DetectNodeYAML `yaml:"detect"`
	Fix      FixNodeYAML    `yaml:"fix,omitempty"`
	Disable  bool           `yaml:"disable,omitempty"`
	Name     string         `yaml:"name,omitempty"`
	URL      string         `yaml:"url,omitempty"`
	Examples ExamplesYAML   `yaml:"examples,omitempty"`
	Related  []RelatedYAML  `yaml:"related,omitempty"`
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
	Type          string   `yaml:"type,omitempty"`
	Count         int      `yaml:"count,omitempty"`
	Per           string   `yaml:"per,omitempty"`
	Window        int      `yaml:"window,omitempty"`
	WindowStr     string   `yaml:"-"` // "sentence" or empty; set by UnmarshalYAML

	// Nested detection tree children — populated by UnmarshalYAML
	PatternNode *DetectNodeYAML
	NearNode    *DetectNodeYAML // second pattern for near
	ChildNode   *DetectNodeYAML
	IfNode      *DetectNodeYAML
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

	// When-method sub-nodes (cross-domain: detect condition + then fix)
	WhenDetect *DetectNodeYAML
	ThenFix    *FixNodeYAML
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
	if n.Type != "" {
		args["type"] = n.Type
	}
	if n.Count > 0 {
		args["count"] = n.Count
	}
	if n.Per != "" {
		args["per"] = n.Per
	}
	if n.WindowStr != "" {
		args["window"] = n.WindowStr
	} else if n.Window > 0 {
		args["window"] = n.Window
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

// detectNodeArgFields is the set of YAML keys that are detect node arguments
// (not method names or child nodes). Used by UnmarshalYAML to distinguish
// inline args from child detection nodes in the key-as-method format.
var detectNodeArgFields = map[string]bool{
	"pattern": true, "words": true, "list": true, "value": true,
	"case_sensitive": true, "left": true, "right": true,
	"min": true, "max": true, "max_chars": true, "mode": true,
	"from": true, "to": true, "type": true,
	"child": true, "if": true, "suggestion": true,
	"count": true, "per": true, "window": true,
}

// knownDetectMethods is the set of all detect method names the parser knows.
// Used to distinguish method-as-key from unknown args in key-as-method format.
var knownDetectMethods = map[string]bool{
	"regex": true, "wordlist": true, "contains": true, "eq": true,
	"prefix": true, "suffix": true,
	"sentence_start": true, "sentence_end": true,
	"paragraph_start": true, "paragraph_end": true,
	"word_boundary": true,
	"before": true, "after": true, "surrounded_by": true,
	"length": true, "case": true, "position": true,
	"and": true, "or": true, "not": true,
	"threshold": true, "near": true, "exclude": true,
	"any": true,
	"llm": true, "plugin": true, "ner": true, "pos": true, "function": true, "expr": true,
	"ref": true,
}

// unresolvedRef captures a RefNode that needs resolution after all rules are parsed.
type unresolvedRef struct {
	Node   *detect.RefNode
	RuleID string // owning rule ID (for error messages and cycle detection)
}

// ParseResult holds the results of parsing YAML.
type ParseResult struct {
	Rules []*model.Rule
	Refs  []unresolvedRef
}

// flatDetectNode is a shadow type without UnmarshalYAML to avoid infinite recursion.
type flatDetectNode DetectNodeYAML

// UnmarshalYAML handles three YAML formats for detection nodes:
//
//  1. Scalar → regex shorthand: `regex: "pattern"` or standalone `"pattern"`
//  2. Sequence → method children: `and: [child1, child2]`
//  3. Mapping → flat format (`method: ..., pattern: ...`) OR
//     key-as-method (`before: { pattern: ..., child: ... }`)
func (n *DetectNodeYAML) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		n.Method = "regex"
		n.Pattern = s
		return nil

	case yaml.SequenceNode:
		return value.Decode(&n.Args)

	case yaml.MappingNode:
		// Check for explicit "method" key → flat format
		for i := 0; i < len(value.Content); i += 2 {
			if value.Content[i].Value == "method" {
				return value.Decode((*flatDetectNode)(n))
			}
		}

		// Key-as-method format: first key is method name
		if len(value.Content) < 2 {
			return nil
		}
		n.Method = value.Content[0].Value
		valNode := value.Content[1]

		switch valNode.Kind {
		case yaml.ScalarNode:
			var s string
			if err := valNode.Decode(&s); err != nil {
				return err
			}
			// Store in appropriate field based on method
			n.Pattern = s

		case yaml.SequenceNode:
			return valNode.Decode(&n.Args)

		case yaml.MappingNode:
			for j := 0; j+1 < len(valNode.Content); j += 2 {
				key := valNode.Content[j].Value
				subVal := valNode.Content[j+1]

				switch {
				case key == "pattern":
					if subVal.Kind == yaml.MappingNode || subVal.Kind == yaml.SequenceNode {
						n.PatternNode = &DetectNodeYAML{}
						if err := subVal.Decode(n.PatternNode); err != nil {
							return err
						}
					} else {
						if err := subVal.Decode(&n.Pattern); err != nil {
							return err
						}
					}
				case key == "child":
					n.ChildNode = &DetectNodeYAML{}
					if err := subVal.Decode(n.ChildNode); err != nil {
						return err
					}
				case key == "if":
					n.IfNode = &DetectNodeYAML{}
					if err := subVal.Decode(n.IfNode); err != nil {
						return err
					}
				case key == "list":
					if err := subVal.Decode(&n.List); err != nil {
						return err
					}
				case key == "words":
					if err := subVal.Decode(&n.Words); err != nil {
						return err
					}
				case key == "value":
					if err := subVal.Decode(&n.Value); err != nil {
						return err
					}
				case key == "case_sensitive":
					if err := subVal.Decode(&n.CaseSensitive); err != nil {
						return err
					}
				case key == "left":
					if err := subVal.Decode(&n.Left); err != nil {
						return err
					}
				case key == "right":
					if err := subVal.Decode(&n.Right); err != nil {
						return err
					}
				case key == "min":
					if err := subVal.Decode(&n.Min); err != nil {
						return err
					}
				case key == "max" || key == "max_chars":
					if err := subVal.Decode(&n.Max); err != nil {
						return err
					}
				case key == "mode":
					if err := subVal.Decode(&n.Mode); err != nil {
						return err
					}
				case key == "from":
					if err := subVal.Decode(&n.From); err != nil {
						return err
					}
				case key == "to":
					if err := subVal.Decode(&n.To); err != nil {
						return err
					}
				case key == "type":
					if err := subVal.Decode(&n.Type); err != nil {
						return err
					}
				case key == "count":
					if err := subVal.Decode(&n.Count); err != nil {
						return err
					}
				case key == "per":
					if err := subVal.Decode(&n.Per); err != nil {
						return err
					}
				case key == "window":
					var asInt int
					if err := subVal.Decode(&asInt); err == nil {
						n.Window = asInt
						break
					}
					var asStr string
					if err := subVal.Decode(&asStr); err != nil {
						return err
					}
					switch {
					case asStr == "sentence":
						n.WindowStr = "sentence"
					case strings.HasPrefix(asStr, "chars:"):
						var chars int
						if _, err := fmt.Sscanf(asStr, "chars:%d", &chars); err == nil {
							n.Window = chars
						} else {
							n.WindowStr = asStr
						}
					default:
						var chars int
						if _, err := fmt.Sscanf(asStr, "%d", &chars); err == nil {
							n.Window = chars
						} else {
							n.WindowStr = asStr
						}
					}
				case key == "near":
					n.NearNode = &DetectNodeYAML{}
					if err := subVal.Decode(n.NearNode); err != nil {
						return err
					}
				default:
					// Unknown key: could be a child method node
					if knownDetectMethods[key] {
						child := &DetectNodeYAML{Method: key}
						if err := decodeChildValue(child, subVal); err != nil {
							return err
						}
						if n.PatternNode == nil {
							n.PatternNode = child
						} else if n.ChildNode == nil {
							n.ChildNode = child
						}
					}
					// else: unknown non-method key → silently skip
				}
			}
		}
		return nil
	}
	return nil
}

// decodeChildValue populates a DetectNodeYAML from a sub-value node
// (the value side of a method-as-key in key-as-method format).
// Handles scalar (regex shorthand), sequence (and/or children), and
// mapping (flat args) formats.
func decodeChildValue(child *DetectNodeYAML, valNode *yaml.Node) error {
	switch valNode.Kind {
	case yaml.ScalarNode:
		var s string
		if err := valNode.Decode(&s); err != nil {
			return err
		}
		if child.Method == "regex" {
			child.Pattern = s
		} else {
			child.Value = s
		}
		return nil

	case yaml.SequenceNode:
		return valNode.Decode(&child.Args)

	case yaml.MappingNode:
		return valNode.Decode((*flatDetectNode)(child))
	}
	return nil
}

// ---------------------------------------------------------------------------
//  FixNodeYAML — custom UnmarshalYAML for key-as-method format
// ---------------------------------------------------------------------------

// flatFixNode is a shadow type without UnmarshalYAML to avoid infinite recursion.
type flatFixNode FixNodeYAML

// UnmarshalYAML handles key-as-method format for fix nodes:
//
//	replace: { with: "..." }    → method=replace, With="..."
//	remove: {}                  → method=remove
//	regex_replace: { ... }      → method=regex_replace
//
// Sibling meta fields under fix: are also accepted:
//
//	fix:
//	  replace: { with: "…" }
//	  suggestion: "Замените три точки…"
func (n *FixNodeYAML) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		return value.Decode(&n.Args)
	case yaml.MappingNode:
		// Check for explicit "method" key → flat format
		for i := 0; i < len(value.Content); i += 2 {
			if value.Content[i].Value == "method" {
				return value.Decode((*flatFixNode)(n))
			}
		}
		// Key-as-method format: first non-meta key is the method;
		// sibling keys (suggestion/autofix) sit alongside it under fix:.
		if len(value.Content) < 2 {
			return nil
		}
		for i := 0; i+1 < len(value.Content); i += 2 {
			key := value.Content[i].Value
			valNode := value.Content[i+1]
			switch key {
			case "suggestion":
				if err := valNode.Decode(&n.Suggestion); err != nil {
					return err
				}
			case "autofix":
				if err := valNode.Decode(&n.Autofix); err != nil {
					return err
				}
			default:
				// First method key wins; ignore extra method siblings
				if n.Method != "" {
					continue
				}
				n.Method = key
				switch valNode.Kind {
				case yaml.SequenceNode:
					if err := valNode.Decode(&n.Args); err != nil {
						return err
					}
				case yaml.MappingNode:
					for j := 0; j+1 < len(valNode.Content); j += 2 {
						argKey := valNode.Content[j].Value
						subVal := valNode.Content[j+1]
						switch argKey {
						case "with":
							if err := subVal.Decode(&n.With); err != nil {
								return err
							}
						case "pattern":
							if err := subVal.Decode(&n.Pattern); err != nil {
								return err
							}
						case "replacement":
							if err := subVal.Decode(&n.Replacement); err != nil {
								return err
							}
						case "prefix":
							if err := subVal.Decode(&n.Prefix); err != nil {
								return err
							}
						case "suffix":
							if err := subVal.Decode(&n.Suffix); err != nil {
								return err
							}
						case "suggestion":
							// Also allow suggestion nested inside the method map
							if err := subVal.Decode(&n.Suggestion); err != nil {
								return err
							}
						case "autofix":
							if err := subVal.Decode(&n.Autofix); err != nil {
								return err
							}
						case "detect":
							n.WhenDetect = &DetectNodeYAML{}
							if err := subVal.Decode(n.WhenDetect); err != nil {
								return err
							}
						case "then":
							n.ThenFix = &FixNodeYAML{}
							if err := subVal.Decode(n.ThenFix); err != nil {
								return err
							}
						}
					}
				}
			}
		}
		return nil
	}
	return nil
}

// ---------------------------------------------------------------------------
//  Build detect/fix trees
// ---------------------------------------------------------------------------

// buildDetectNode recursively builds a detect tree from YAML node.
// Returns (detectNode, detectMethod, error).
// For server-side methods (llm/plugin), detectNode is nil.
func buildDetectNode(ny *DetectNodeYAML, ruleID string, refs *[]unresolvedRef) (detect.Node, string, error) {
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

	// ref method — creates a RefNode that will be resolved phase 2
	if method == "ref" {
		refID := ny.Pattern
		if refID == "" {
			return nil, "", &Error{Op: "buildDetectNode", Message: "ref method requires a rule ID"}
		}
		refNode := detect.NewRefNode(refID)
		if refs != nil {
			*refs = append(*refs, unresolvedRef{Node: refNode, RuleID: ruleID})
		}
		return refNode, method, nil
	}

	if method == "" {
		// No method and no inferrable fields — treat as noop (returns nil)
		return nil, "", nil
	}

	// Build child nodes from all sub-node fields
	var children []detect.Node

	// 1. Args array (and/or logical operators)
	for i := range ny.Args {
		child, _, err := buildDetectNode(&ny.Args[i], ruleID, refs)
		if err != nil {
			return nil, "", &Error{Op: "buildDetectNode", Message: "build child " + ny.Args[i].Method, Err: err}
		}
		if child != nil {
			children = append(children, child)
		}
	}

	// 2. ChildNode (inner filter for length/case/position/contextual)
	if ny.ChildNode != nil {
		child, _, err := buildDetectNode(ny.ChildNode, ruleID, refs)
		if err != nil {
			return nil, "", &Error{Op: "buildDetectNode", Message: "build child " + ny.ChildNode.Method, Err: err}
		}
		if child != nil {
			children = append(children, child)
		}
	}

	// 3. PatternNode (anchor pattern for before/after/near)
	if ny.PatternNode != nil {
		child, _, err := buildDetectNode(ny.PatternNode, ruleID, refs)
		if err != nil {
			return nil, "", &Error{Op: "buildDetectNode", Message: "build child " + ny.PatternNode.Method, Err: err}
		}
		if child != nil {
			children = append(children, child)
		}
	}

	// 4. NearNode (second pattern for near)
	if ny.NearNode != nil {
		child, _, err := buildDetectNode(ny.NearNode, ruleID, refs)
		if err != nil {
			return nil, "", &Error{Op: "buildDetectNode", Message: "build near " + ny.NearNode.Method, Err: err}
		}
		if child != nil {
			children = append(children, child)
		}
	}

	// 5. IfNode (conditional detection)
	if ny.IfNode != nil {
		child, _, err := buildDetectNode(ny.IfNode, ruleID, refs)
		if err != nil {
			return nil, "", &Error{Op: "buildDetectNode", Message: "build child " + ny.IfNode.Method, Err: err}
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

	// Handle "when" method (cross-domain: detect condition + fix)
	if method == "when" {
		var condition detect.Node
		if ny.WhenDetect != nil {
			var detectMethod string // not used
			var err error
			condition, detectMethod, err = buildDetectNode(ny.WhenDetect, "", nil)
			if err != nil {
				return nil, &Error{Op: "buildFixNode", Message: "build when condition", Err: err}
			}
			_ = detectMethod
		}
		var thenNode fix.Node
		if ny.ThenFix != nil {
			var err error
			thenNode, err = buildFixNode(ny.ThenFix)
			if err != nil {
				return nil, &Error{Op: "buildFixNode", Message: "build when then", Err: err}
			}
		}
		return &fix.WhenNode{Condition: condition, Then: thenNode}, nil
	}

	args := ny.buildArgs()
	return fix.Build(method, args, children)
}

// ---------------------------------------------------------------------------
//  ParseYAML
// ---------------------------------------------------------------------------

// ParseYAML parses raw YAML bytes into domain Rule objects.
// Supports both old flat format and new nested method tree format (SPEC §8, A26/Q33).
// Returns a ParseResult which includes refs that need resolution phase 2.
func ParseYAML(data []byte) (ParseResult, error) {
	var rf RulesFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return ParseResult{}, &Error{Op: "ParseYAML", Message: "invalid YAML", Err: err}
	}

	var rules []*model.Rule
	var refs []unresolvedRef
	for _, ry := range rf.Rules {
		rule, err := ruleFromYAML(ry, &refs)
		if err != nil {
			return ParseResult{}, err
		}
		rules = append(rules, rule)
	}

	return ParseResult{Rules: rules, Refs: refs}, nil
}

// ruleFromYAML converts a RuleYAML to a domain Rule.
func ruleFromYAML(ry RuleYAML, refs *[]unresolvedRef) (*model.Rule, error) {
	// Handle disable-only overrides
	if ry.Disable {
		if _, err := model.RuleIDFromString(ry.ID); err != nil {
			return nil, &Error{Op: "ParseYAML", Message: "disable without valid id: " + ry.ID, Err: err}
		}
		rule, err := model.NewRule(ry.ID, 5, "cleanliness", "regex", nil, nil, 0, "", "", "", model.Examples{}, nil)
		if err != nil {
			return nil, &Error{Op: "ParseYAML", Message: "create disabled rule stub", Err: err}
		}
		return rule.Disable(), nil
	}

	// Build detect tree
	detectNode, detectMethod, err := buildDetectNode(&ry.Detect, ry.ID, refs)
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
		detectMethod, detectNode, fixNode, ry.Priority,
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
