package model

import "fmt"

// RuleSet — immutable snapshot of merged rule configuration (A24).
type RuleSet struct {
	rules      []*Rule
	configHash ConfigHash
}

// NewRuleSet from raw values.
func NewRuleSet(rules []*Rule, configHash uint64) *RuleSet {
	return &RuleSet{
		rules:      rules,
		configHash: ConfigHashFromUint64(configHash),
	}
}

func (rs *RuleSet) Rules() []*Rule         { return rs.rules }
func (rs *RuleSet) ConfigHash() ConfigHash { return rs.configHash }

func (rs *RuleSet) ForCategory(category Category) []*Rule {
	var out []*Rule
	for _, r := range rs.rules {
		if r.Category() == category {
			out = append(out, r)
		}
	}
	return out
}

func (rs *RuleSet) ClientRules() []*Rule {
	var out []*Rule
	for _, r := range rs.rules {
		if r.IsClientSide() {
			out = append(out, r)
		}
	}
	return out
}

func (rs *RuleSet) ServerRules() []*Rule {
	var out []*Rule
	for _, r := range rs.rules {
		if !r.IsClientSide() {
			out = append(out, r)
		}
	}
	return out
}

// Merge deep-merges another RuleSet into this one (A24).
func (rs *RuleSet) Merge(other *RuleSet) *RuleSet {
	idx := make(map[RuleID]int)
	var merged []*Rule

	for _, r := range rs.rules {
		idx[r.ID()] = len(merged)
		merged = append(merged, r)
	}
	for _, r := range other.rules {
		if i, ok := idx[r.ID()]; ok {
			if r.IsEnabled() {
				merged[i] = r
			} else {
				merged = append(merged[:i], merged[i+1:]...)
				idx = make(map[RuleID]int)
				for j, mr := range merged {
					idx[mr.ID()] = j
				}
			}
		} else {
			idx[r.ID()] = len(merged)
			merged = append(merged, r)
		}
	}

	h := hashCombine(rs.configHash.Value(), other.configHash.Value())
	return NewRuleSet(merged, h)
}

func hashCombine(a, b uint64) uint64 {
	h := a ^ b
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

func (rs *RuleSet) String() string {
	return fmt.Sprintf("RuleSet{hash=%016x, rules=%d}", rs.configHash.Value(), len(rs.rules))
}
