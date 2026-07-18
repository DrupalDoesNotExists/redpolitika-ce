# Rule cookbook

Composable patterns for common editorial checks. Full reference: [rules.md](rules.md).

---

## 1. Whitelist exclusion (`exclude` / and+not)

Flag words with «ё» except proper names / exceptions.

```yaml
rules:
  - id: "typography/yo-outside-whitelist"
    severity: 3
    category: "readability"
    name: "Буква ё вне словаря"
    detect:
      exclude:
        list: ["ёлка", "ёжик", "Ельцин"]
        child:
          regex:
            pattern: "[а-яА-ЯёЁ]*ё[а-яА-ЯёЁ]*"
            case_sensitive: true
    suggestion: "Проверьте написание через «ё»"
```

Same without sugar:

```yaml
detect:
  and:
    - regex: { pattern: "[а-яА-ЯёЁ]*ё[а-яА-ЯёЁ]*", case_sensitive: true }
    - not:
        wordlist: { list: ["ёлка", "ёжик"] }
```

---

## 2. Concentration (`threshold`)

Flag when a hedge word appears 3+ times per 100 words.

```yaml
rules:
  - id: "tone/hedge-concentration"
    severity: 4
    category: "readability"
    detect:
      threshold:
        count: 3
        per: words
        window: 100
        child:
          wordlist:
            list: ["вроде", "как бы", "типа"]
    suggestion: "Слишком много хеджирования на участке текста"
```

---

## 3. Contextual before/after

Lowercase after a colon (child must be uppercase letter right after `: `).

```yaml
rules:
  - id: "capitalization/after-colon"
    severity: 4
    category: "readability"
    detect:
      after:
        pattern:
          contains: { value: ": ", case_sensitive: false }
        max_chars: 80
        child:
          regex:
            pattern: "^[А-ЯЁ]"
            case_sensitive: true
    fix:
      lowercase: {}
    suggestion: "После двоеточия — строчная, если это не новое предложение"
```

---

## 4. Paired stamps (`near`)

Two clichés in the same sentence.

```yaml
rules:
  - id: "cliches/paired"
    severity: 5
    category: "cleanliness"
    detect:
      near:
        pattern:
          wordlist: { list: ["на сегодняшний день"] }
        near:
          wordlist: { list: ["в рамках"] }
        window: sentence
    suggestion: "Два канцеляризма рядом — уберите один"
```

---

## 5. Capture groups in fix

```yaml
rules:
  - id: "style/reorder"
    severity: 2
    category: "readability"
    detect:
      regex: "(у нас) (есть)"
    fix:
      regex_replace:
        pattern: ".*"
        replacement: "$2 у нас"
    suggestion: "Переставьте порядок слов"
```

---

## 6. Conditional fix (`when`)

Replace ё→е only for all-lowercase matches (skip likely proper names).

```yaml
rules:
  - id: "typography/yo-to-e"
    severity: 2
    category: "readability"
    detect:
      regex: "[а-яё]*ё[а-яё]*"
    fix:
      when:
        detect:
          case: { mode: all_lower }
        then:
          regex_replace:
            pattern: "ё"
            replacement: "е"
```

---

## Related

- [rules.md](rules.md) — full method reference
- [ai-agent-skill.md](ai-agent-skill.md) — writing rules with an AI agent
