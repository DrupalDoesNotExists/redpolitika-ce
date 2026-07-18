/* ------------------------------------------------------------------ */
/*  Client rule engine — tentative rules, FNV-1a, paragraph anchoring  */
/* ------------------------------------------------------------------ */

import { z } from "zod";
import type { FlagState, Anchor, Category } from "./store";

/* ------------------------------------------------------------------ */
/*  Zod schemas                                                        */
/* ------------------------------------------------------------------ */

/** Wire format from GET /api/client-rules (docs/api.md). */
const ClientRuleWireSchema = z.object({
  id: z.string(),
  name: z.string().optional(),
  method: z.enum(["regex", "wordlist"]),
  category: z.enum(["cleanliness", "readability"]).optional().default("cleanliness"),
  pattern: z.string().optional(),
  words: z.array(z.string()).optional(),
  severity: z.number().min(1).max(10),
  case_sensitive: z.boolean().optional(),
  suggestion: z.string().optional(),
  auto_fix: z.string().nullable().optional(),
  message: z.string().optional(),
  description: z.string().optional(),
  engine: z.string().optional(),
});

export type ClientRule = {
  id: string;
  name?: string;
  type: "regex" | "wordlist";
  category: "cleanliness" | "readability";
  pattern?: string;
  words?: string[];
  severity: number;
  case_sensitive?: boolean;
  suggestion?: string;
  autofix?: string;
  message?: string;
  description?: string;
};

function wireToClientRule(w: z.infer<typeof ClientRuleWireSchema>): ClientRule {
  return {
    id: w.id,
    name: w.name,
    type: w.method,
    category: w.category,
    pattern: w.pattern,
    words: w.words,
    severity: w.severity,
    case_sensitive: w.case_sensitive,
    suggestion: w.suggestion,
    autofix: w.auto_fix ?? undefined,
    message: w.message,
    description: w.description,
  };
}

/* ------------------------------------------------------------------ */
/*  FNV-1a 64-bit hash → base36 string (per A23)                       */
/* ------------------------------------------------------------------ */

const FNV_OFFSET = 0xcbf29ce484222325n;
const FNV_PRIME = 0x100000001b3n;
const FNV_MASK = 0xffffffffffffffffn;

export function fnv1a64(input: string): string {
  let hash = FNV_OFFSET;
  for (let i = 0; i < input.length; i++) {
    hash ^= BigInt(input.charCodeAt(i));
    hash = (hash * FNV_PRIME) & FNV_MASK;
  }
  return hash.toString(36);
}

/** Quick text hash for stale-response detection (FNV-1a of full text). */
export function textHash(input: string): string {
  return fnv1a64(input);
}

/* ------------------------------------------------------------------ */
/*  Paragraph utilities                                                */
/* ------------------------------------------------------------------ */

export interface ParagraphRange {
  text: string;
  start: number; // offset in full text
  end: number;   // offset in full text (exclusive)
}

/** Split text into paragraphs by \n\n, preserving offsets. */
export function splitParagraphs(text: string): ParagraphRange[] {
  const paras: ParagraphRange[] = [];
  let start = 0;
  while (start < text.length) {
    const delim = text.indexOf("\n\n", start);
    if (delim === -1) {
      paras.push({ text: text.slice(start), start, end: text.length });
      break;
    }
    paras.push({ text: text.slice(start, delim), start, end: delim });
    start = delim + 2;
  }
  // If text is empty, return one empty paragraph
  if (paras.length === 0 && text.length === 0) {
    paras.push({ text: "", start: 0, end: 0 });
  }
  return paras;
}

/** Map anchor to offset span in current text. Returns null if anchor cannot be resolved. */
export function anchorToSpan(
  text: string,
  anchor: Anchor,
): { from: number; to: number } | null {
  const paras = splitParagraphs(text);
  if (anchor.paragraphIndex >= paras.length) return null;

  const para = paras[anchor.paragraphIndex];

  // Find occurrence of matchText in paragraph
  let found = -1;
  let searchFrom = 0;
  for (let occ = 0; occ <= anchor.occurrence; occ++) {
    const idx = para.text.indexOf(anchor.matchText, searchFrom);
    if (idx === -1) return null;
    found = idx;
    searchFrom = idx + anchor.matchText.length;
  }

  if (found === -1) return null;

  return {
    from: para.start + found,
    to: para.start + found + anchor.matchText.length,
  };
}

/* ------------------------------------------------------------------ */
/*  Client rule evaluation — paragraph-based anchoring                 */
/* ------------------------------------------------------------------ */

export function evaluateRuleOnText(
  rule: ClientRule,
  text: string,
): FlagState[] {
  const flags: FlagState[] = [];
  const paras = splitParagraphs(text);

  for (let paraIdx = 0; paraIdx < paras.length; paraIdx++) {
    const para = paras[paraIdx];

    if (rule.type === "regex" && rule.pattern) {
      try {
        const re = new RegExp(rule.pattern, "g" + (rule.pattern.includes("(?i)") ? "" : ""));
        let match: RegExpExecArray | null;
        let occurrence = 0;
        while ((match = re.exec(para.text)) !== null) {
          if (match[0].length === 0) {
            re.lastIndex++;
            continue;
          }

          const anchor: Anchor = {
            paragraphIndex: paraIdx,
            occurrence,
            matchText: match[0],
          };

          // A23: flagId = FNV-1a 64 from rule_id ‖ match_text ‖ paragraph_index ‖ occurrence
          const raw = `${rule.id}\0${match[0]}\0${paraIdx}\0${occurrence}`;
          flags.push(createTentativeFlag(raw, rule, anchor));

          occurrence++;
          if (match[0].length === 0) break;
        }
      } catch {
        // invalid regex — skip silently
      }
    }

    if (rule.type === "wordlist" && rule.words) {
      const caseSensitive = rule.case_sensitive ?? false;
      const searchText = caseSensitive ? para.text : para.text.toLowerCase();
      for (const word of rule.words) {
        const searchWord = caseSensitive ? word : word.toLowerCase();
        let occurrence = 0;
        let pos = 0;
        while (pos < para.text.length) {
          const idx = searchText.indexOf(searchWord, pos);
          if (idx === -1) break;

          // Word boundary check — prevent "кат" matching "категория"
          if (idx > 0 && /\p{L}/u.test(searchText[idx - 1])) {
            pos = idx + 1;
            continue;
          }
          const end = idx + word.length;
          if (end < searchText.length && /\p{L}/u.test(searchText[end])) {
            pos = idx + 1;
            continue;
          }

          const anchor: Anchor = {
            paragraphIndex: paraIdx,
            occurrence,
            matchText: para.text.slice(idx, idx + word.length),
          };

          const raw = `${rule.id}\0${word}\0${paraIdx}\0${occurrence}`;
          flags.push(createTentativeFlag(raw, rule, anchor));

          occurrence++;
          pos = end;
        }
      }
    }
  }

  return flags;
}

function createTentativeFlag(
  rawKey: string,
  rule: ClientRule,
  anchor: Anchor,
): FlagState {
  return {
    id: fnv1a64(rawKey),
    ruleId: rule.id,
    ruleName: rule.name ?? rule.id,
    category: (rule.category ?? "cleanliness") as Category,
    severity: rule.severity,
    message: rule.message ?? `Нарушение правила «${rule.id}»`,
    suggestion: rule.suggestion,
    autoFix: rule.autofix,
    anchor,
    status: "tentative",
    span: { from: 0, to: 0 }, // will be resolved by caller
  };
}

/* ------------------------------------------------------------------ */
/*  Run all client rules                                               */
/* ------------------------------------------------------------------ */

export function runAllClientRules(
  rules: ClientRule[],
  text: string,
): FlagState[] {
  const seen = new Set<string>();
  const all: FlagState[] = [];

  for (const rule of rules) {
    const result = evaluateRuleOnText(rule, text);
    for (const flag of result) {
      if (!seen.has(flag.id)) {
        // Resolve span from anchor
        const span = anchorToSpan(text, flag.anchor);
        if (span) {
          flag.span = span;
        }
        seen.add(flag.id);
        all.push(flag);
      }
    }
  }

  return all;
}

/* ------------------------------------------------------------------ */
/*  Tentative scores — per A25/Q32                                     */
/*  score = 10 − clamp((Σseverity × 100) / word_count, 0, 10)         */
/* ------------------------------------------------------------------ */

export function calculateTentativeScores(
  flags: FlagState[],
  wordCount: number,
): { cleanliness: number; readability: number } {
  if (wordCount === 0) return { cleanliness: 10, readability: 10 };

  let cleanlinessSum = 0;
  let readabilitySum = 0;

  for (const f of flags) {
    if (f.status === "rejected") continue;
    if (f.category === "cleanliness") {
      cleanlinessSum += f.severity;
    } else {
      readabilitySum += f.severity;
    }
  }

  const clamp = (v: number) => Math.max(0, Math.min(10, v));

  return {
    cleanliness: 10 - clamp((cleanlinessSum * 100) / wordCount),
    readability: 10 - clamp((readabilitySum * 100) / wordCount),
  };
}

/* ------------------------------------------------------------------ */
/*  Fetch rules from API                                               */
/* ------------------------------------------------------------------ */

export async function fetchClientRules(): Promise<ClientRule[]> {
  const res = await fetch("/api/client-rules");
  if (!res.ok) {
    throw new Error(`Failed to fetch client rules: ${res.status}`);
  }
  const json: unknown = await res.json();
  return z.array(ClientRuleWireSchema).parse(json).map(wireToClientRule);
}
