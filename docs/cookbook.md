---
title: Рецепты правил
description: Готовые композируемые паттерны для типовых задач редполитики
weight: 80
language: ru
---

# Рецепты правил

Готовые композируемые паттерны для типовых задач. Полный справочник — [guide-rules.md](guide-rules.md).

---

## 1. Исключение из списка (`exclude` / and+not)

Флаги слов с «ё» кроме исключений.

```yaml
rules:
  - id: "typography/yo-outside-whitelist"
    severity: 3
    category: "readability"
    name: "Буква ё вне словаря"
    detect:
      exclude:
        match:
          regex:
            pattern: "[а-яА-ЯёЁ]*ё[а-яА-ЯёЁ]*"
            case_sensitive: true
        without:
          - wordlist:
              list: ["ёлка", "ёжик", "Ельцин"]
    fix:
      suggestion: "Проверьте написание через «ё»"
```

Без сахара:

```yaml
detect:
  and:
    - regex: { pattern: "[а-яА-ЯёЁ]*ё[а-яА-ЯёЁ]*", case_sensitive: true }
    - not:
        wordlist: { list: ["ёлка", "ёжик"] }
```

---

## 2. Концентрация (`threshold`)

Флаг, когда слово-паразит встречается 3+ раз на 100 слов.

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
    fix:
      suggestion: "Слишком много хеджирования на участке текста"
```

---

## 3. Контекст: before / after

Строчная после двоеточия.

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

## 4. Парные штампы (`near`)

Два канцеляризма в одном предложении.

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
    fix:
      suggestion: "Два канцеляризма рядом — уберите один"
```

---

## 5. Capture groups в fix

Перестановка слов.

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

## 6. Условный fix (`when`)

Замена ё→е только для слов в нижнем регистре (пропуск имён собственных).

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

## Связанное

- [guide-rules.md](guide-rules.md) — полный справочник методов
