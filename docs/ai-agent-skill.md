# AI Agent Skill: Writing redpolitika Rules

This skill teaches AI coding agents how to write YAML rules for redpolitika — the editorial policy text checker.

For full format reference, see [rules.md](rules.md).

---

## Rule structure

```yaml
rules:
  - id: "category/descriptive-name"   # kebab-case, namespace/id
    severity: 5                        # 1-10
    category: "cleanliness"            # "cleanliness" | "readability"
    name: "Human readable"
    suggestion: "Что делать"
    detect:
      # detection tree
    fix:
      # optional fix tree
    examples:
      bad: ["Wrong"]
      good: ["Right"]
```

## Best practices

- **ID naming:** `category/descriptive-name` — e.g. `typography/ellipsis`, `redundancy/very`
- **Severity:** 8-10 for slurs/obscene, 5-7 for annoying, 3-4 for style, 1-2 for informational
- **Suggestion:** Always actionable. Tell what to do, in target language
- **Fix:** Include when replacement is unambiguous (typo→correct). Omit when context-dependent (style choice)
- **Examples:** Always both `bad` and `good`. Mirror real usage

### Detection rules of thumb

| Prefer | Over |
|--------|------|
| `wordlist` for word-level checks | `regex` for simple word lists |
| `contains` for fixed phrases | `regex` for literal strings |
| `and`/`or`/`not` composition | Giant alternation regex |
| `before`/`after` for context | Regex lookaround (RE2 doesn't support it) |

### What NOT to include

- Regex backreferences or lookaround — RE2 rejects them at load time
- Flat `method:` syntax (old format) — use tree-based `detect:` instead

## Common mistakes

| Mistake | Fix |
|---------|-----|
| `severity: 5` on everything | Match severity to actual impact |
| No `suggestion` | Always tell user what to do |
| `regex` where `wordlist` works | Use `wordlist` with `case_sensitive` |
| Lookahead/lookbehind | Use `before`/`after` contextual wrappers |

## Example

Task: Flag "очень" before "важно"/"необходимо"/"нужно".

```yaml
rules:
  - id: "redundancy/very"
    severity: 5
    category: "cleanliness"
    name: "Избыточный усилитель «очень»"
    url: "/rules/redundant-very"
    detect:
      regex: "очень\\s+(важно|необходимо|нужно)"
    suggestion: "Уберите «очень» — фраза сильнее без него"
    examples:
      bad:
        - "Это очень важно"
        - "Мне очень нужно"
      good:
        - "Это важно"
        - "Мне нужно"
```

## Verification

1. Validate YAML: `python3 -c "import yaml; yaml.safe_load(open('file.yaml'))"`
2. Start redpolitika — it validates all rules on load; invalid rules block startup
3. Test: `curl -X POST http://localhost:8080/api/analyze -d '{"text":"test"}'`

## System prompt fragment

```
You write rules for redpolitika — editorial policy text checker.
YAML, composable detect/fix method trees.
Only tree-based methods: regex, wordlist, contains, eq, prefix, suffix,
sentence_start/end, paragraph_start/end, word_boundary, length, case,
before, after, surrounded_by, position, and, or, not.
RE2 regex — no backreferences/lookaround. 
severity 1-10, category "cleanliness" or "readability".
Always include suggestion + examples.
```

## Related

- [Full rules reference](rules.md)
- [API reference](api.md)
