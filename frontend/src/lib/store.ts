/* ------------------------------------------------------------------ */
/*  Zustand store — flags, scores, session, WS connection              */
/* ------------------------------------------------------------------ */

import { create } from "zustand";
import { textHash, anchorToSpan } from "./client-engine";

/* ── Types ── */

export type FlagStatus =
  | "tentative"
  | "confirmed"
  | "accepted"
  | "rejected"
  | "applied";

export type Category = "cleanliness" | "readability";

export type ConnectionStatus = "online" | "reconnecting" | "offline";

export interface Anchor {
  paragraphIndex: number;
  occurrence: number;
  matchText: string;
}

export interface FlagState {
  id: string;
  ruleId: string;
  ruleName: string;
  category: Category;
  severity: number;
  message: string;
  suggestion?: string;
  autoFix?: string;
  anchor: Anchor;
  status: FlagStatus;
  span: { from: number; to: number };
  examples?: { bad: string[]; good: string[] };
  related?: { name: string; url?: string }[];
  ruleUrl?: string;
}

export interface Scores {
  cleanliness: number;
  readability: number;
}

/** Server flag shape (snake_case wire format). */
export interface ServerFlag {
  id: string;
  rule_id: string;
  rule_name?: string;
  category: Category;
  severity: number;
  message: string;
  suggestion?: string;
  auto_fix?: string;
  anchor: {
    paragraph_index: number;
    occurrence: number;
    match_text: string;
  };
  examples?: { bad: string[]; good: string[] };
  related?: { name: string; url?: string }[];
  rule_url?: string;
}

/* ── Convert server flag to internal FlagState ── */

export function serverFlagToState(f: ServerFlag): FlagState {
  return {
    id: f.id,
    ruleId: f.rule_id,
    ruleName: f.rule_name ?? f.rule_id,
    category: f.category,
    severity: f.severity,
    message: f.message,
    suggestion: f.suggestion,
    autoFix: f.auto_fix,
    anchor: {
      paragraphIndex: f.anchor.paragraph_index,
      occurrence: f.anchor.occurrence,
      matchText: f.anchor.match_text,
    },
    status: "confirmed",
    span: { from: 0, to: 0 },
    examples: f.examples,
    related: f.related,
    ruleUrl: f.rule_url,
  };
}

/* ── Stable selectors / helpers ── */

export function wordCount(text: string): number {
  return text.trim() ? text.trim().split(/\s+/).filter(Boolean).length : 0;
}

export function sentenceCount(text: string): number {
  if (!text.trim()) return 0;
  const trimmed = text.trim();
  // Count sentence-ending punctuation
  const matches = trimmed.match(/[.!?…]+(\s|$)/g);
  return matches ? matches.length : (trimmed.length > 0 ? 1 : 0);
}

export function charCount(text: string): number {
  return text.length;
}

export function getFlagsByCategory(
  flags: Record<string, FlagState>,
  category: Category,
): FlagState[] {
  return Object.values(flags)
    .filter(
      (f) =>
        f.category === category &&
        f.status !== "rejected" &&
        f.status !== "applied",
    )
    .sort((a, b) => b.severity - a.severity);
}

export function getActiveFlags(
  flags: Record<string, FlagState>,
): FlagState[] {
  return Object.values(flags)
    .filter((f) => f.status !== "rejected" && f.status !== "applied")
    .sort((a, b) => b.severity - a.severity);
}

export function getAcceptedFlags(
  flags: Record<string, FlagState>,
): FlagState[] {
  return Object.values(flags)
    .filter((f) => f.status === "accepted")
    .sort((a, b) => b.severity - a.severity);
}

export function getCategoryCounts(
  flags: Record<string, FlagState>,
): { cleanliness: number; readability: number } {
  let c = 0,
    r = 0;
  for (const f of Object.values(flags)) {
    if (f.status === "rejected" || f.status === "applied") continue;
    if (f.category === "cleanliness") c++;
    else r++;
  }
  return { cleanliness: c, readability: r };
}

/* ── Helpers ── */

/** Clean up whitespace/punctuation after text deletion (autofix=""). */
function cleanDelete(text: string): string {
  return text
    .replace(/  +/g, ' ')            // collapse multiple spaces
    .replace(/ ([.,!?;:…])/g, '$1') // space before punctuation
    .replace(/([.,!?;:…])\1+/g, '$1') // collapse doubled punctuation
    .replace(/^[,.\s…]+/, '')        // trim leading punctuation/space/dots
    .replace(/[,.\s…]+$/, '');       // trim trailing punctuation/space/dots
}

/** Re-resolve spans from anchors after text mutation; drop unresolvable. */
function reresolveFlags(
  flags: Record<string, FlagState>,
  text: string,
): Record<string, FlagState> {
  const next = { ...flags };
  for (const id of Object.keys(next)) {
    const f = next[id];
    if (f.status === "applied") continue;
    const span = anchorToSpan(text, f.anchor);
    if (span) {
      next[id] = { ...f, span };
    } else {
      delete next[id];
    }
  }
  return next;
}

/* ── Store ── */

interface StoreState {
  text: string;
  flags: Record<string, FlagState>;
  scores: Scores;
  sessionId: string | null;
  connectionStatus: ConnectionStatus;
  full: boolean;
  wordCount: number;
  textHash: string;
  drawerOpen: boolean;
  sentenceCount: number;
  charCount: number;

  setText: (text: string) => void;
  mergeServerFlags: (serverFlags: FlagState[]) => void;
  mergeFlagsFromServerWire: (serverFlags: ServerFlag[]) => void;
  addTentativeFlags: (tentativeFlags: FlagState[]) => void;
  acceptFlag: (id: string) => void;
  rejectFlag: (id: string) => void;
  applyFlagFix: (id: string) => string | null;
  applyAllAccepted: () => string | null;
  updateTentativeScores: () => void;
  setScores: (scores: Scores) => void;
  setSessionId: (id: string | null) => void;
  setConnectionStatus: (status: ConnectionStatus) => void;
  setFull: (full: boolean) => void;
  setDrawerOpen: (open: boolean) => void;
}

export const useStore = create<StoreState>()((set, get) => ({
  text: "",
  flags: {},
  scores: { cleanliness: 10, readability: 10 },
  sessionId: null,
  connectionStatus: "offline",
  full: true,
  wordCount: 0,
  textHash: "",
  drawerOpen: false,
  sentenceCount: 0,
  charCount: 0,

  setText: (text) => {
    const hash = textHash(text);
    const wc = wordCount(text);
    const sc = sentenceCount(text);
    const cc = charCount(text);
    const flags = reresolveFlags(get().flags, text);

    set({ text, textHash: hash, wordCount: wc, flags, sentenceCount: sc, charCount: cc });
  },

  mergeServerFlags: (incoming) => {
    const flags = { ...get().flags };
    const text = get().text;

    for (const sf of incoming) {
      const existing = flags[sf.id];
      const status =
        existing && (existing.status === "accepted" || existing.status === "rejected")
          ? existing.status
          : "confirmed";
      flags[sf.id] = { ...sf, status };
      const span = anchorToSpan(text, sf.anchor);
      if (span) flags[sf.id].span = span;
    }

    // Remove flags not in server response unless accepted/rejected
    const incomingIds = new Set(incoming.map((f) => f.id));
    for (const id of Object.keys(flags)) {
      if (!incomingIds.has(id)) {
        const f = flags[id];
        if (f.status === "tentative" || f.status === "confirmed") {
          delete flags[id];
        }
      }
    }

    set({ flags });
  },

  mergeFlagsFromServerWire: (serverFlags) => {
    const converted = serverFlags.map(serverFlagToState);
    get().mergeServerFlags(converted);
  },

  addTentativeFlags: (tentative) => {
    const flags = { ...get().flags };
    const text = get().text;

    for (const tf of tentative) {
      if (!flags[tf.id]) {
        const span = anchorToSpan(text, tf.anchor);
        if (span) tf.span = span;
        flags[tf.id] = tf;
      }
    }

    set({ flags });
  },

  acceptFlag: (id) => {
    set((state) => {
      const flags = { ...state.flags };
      if (flags[id]) {
        flags[id] = { ...flags[id], status: "accepted" };
      }
      return { flags };
    });
  },

  rejectFlag: (id) => {
    set((state) => {
      const flags = { ...state.flags };
      if (flags[id]) {
        flags[id] = { ...flags[id], status: "rejected" };
      }
      return { flags };
    });
  },

  applyFlagFix: (id) => {
    const state = get();
    const flag = state.flags[id];
    if (!flag || flag.autoFix === undefined) return null;

    const { from, to } = flag.span;
    if (from < 0 || to > state.text.length || from >= to) return null;

    let newText: string;
    if (flag.autoFix === "") {
      // Delete matched text and clean surrounding whitespace/punctuation
      newText = state.text.slice(0, from) + state.text.slice(to);
      newText = cleanDelete(newText);
    } else {
      newText =
        state.text.slice(0, from) + flag.autoFix + state.text.slice(to);
    }

    const flags = { ...state.flags };
    delete flags[id];

    set({
      text: newText,
      textHash: textHash(newText),
      wordCount: wordCount(newText),
      sentenceCount: sentenceCount(newText),
      charCount: charCount(newText),
      // Keep remaining highlights valid until the next WS check_result
      flags: reresolveFlags(flags, newText),
    });
    return newText;
  },

  applyAllAccepted: () => {
    const state = get();
    const accepted = Object.values(state.flags)
      .filter((f) => f.status !== "rejected" && f.status !== "applied" && f.autoFix !== undefined)
      .sort((a, b) => b.span.from - a.span.from);

    if (accepted.length === 0) return null;

    let newText = state.text;
    const flags = { ...state.flags };

    for (const f of accepted) {
      const { from, to } = f.span;
      if (from < 0 || to > newText.length || from >= to) continue;
      if (f.autoFix === "") {
        newText = newText.slice(0, from) + newText.slice(to);
      } else {
        newText = newText.slice(0, from) + f.autoFix + newText.slice(to);
      }
      delete flags[f.id];
    }

    newText = cleanDelete(newText);
    set({
      text: newText,
      textHash: textHash(newText),
      wordCount: wordCount(newText),
      sentenceCount: sentenceCount(newText),
      charCount: charCount(newText),
      flags: reresolveFlags(flags, newText),
    });
    return newText;
  },

  updateTentativeScores: () => {
    const state = get();
    if (state.wordCount === 0) {
      set({ scores: { cleanliness: 10, readability: 10 } });
      return;
    }

    let cleanSum = 0;
    let readSum = 0;

    for (const f of Object.values(state.flags)) {
      if (f.status === "rejected" || f.status === "applied") continue;
      if (f.category === "cleanliness") cleanSum += f.severity;
      else readSum += f.severity;
    }

    const clamp = (v: number) => Math.max(0, Math.min(10, v));

    set({
      scores: {
        cleanliness: 10 - clamp((cleanSum * 100) / state.wordCount),
        readability: 10 - clamp((readSum * 100) / state.wordCount),
      },
    });
  },

  setScores: (scores) => set({ scores }),
  setSessionId: (id) => set({ sessionId: id }),
  setConnectionStatus: (status) => set({ connectionStatus: status }),
  setFull: (full) => set({ full }),
  setDrawerOpen: (open) => set({ drawerOpen: open }),
}));
