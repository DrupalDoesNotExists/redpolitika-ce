"use client";

import { useEffect, useRef } from "react";
import { basicSetup } from "codemirror";
import {
  Decoration,
  type DecorationSet,
  EditorView,
  gutter,
  GutterMarker,
} from "@codemirror/view";
import {
  StateEffect,
  StateField,
  type Range,
} from "@codemirror/state";
import type { FlagState, Category } from "@/lib/store";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface DecorationPayload {
  flags: FlagState[];
  text: string;
}

/* ------------------------------------------------------------------ */
/*  Decoration effect                                                  */
/* ------------------------------------------------------------------ */

const flagEffect = StateEffect.define<DecorationPayload>();
const flagDataEffect = StateEffect.define<FlagState[]>();

/* ------------------------------------------------------------------ */
/*  Build decorations from flags — category + status + severity         */
/* ------------------------------------------------------------------ */

function buildDecos(payload: DecorationPayload): DecorationSet {
  const { flags, text } = payload;
  const decos: Range<Decoration>[] = [];

  for (const f of flags) {
    if (f.status === "applied") continue;

    const { from, to } = f.span;
    if (from < 0 || to > text.length || from >= to) continue;

    const categoryClass =
      f.category === "cleanliness"
        ? "category-cleanliness"
        : "category-readability";

    const severityClass =
      f.severity >= 7
        ? "severity-high"
        : f.severity >= 4
          ? "severity-mid"
          : "severity-low";

    const statusClass =
      f.status === "tentative"
        ? "status-tentative"
        : f.status === "accepted"
          ? "status-accepted"
          : f.status === "rejected"
            ? "status-rejected"
            : "";

    decos.push(
      Decoration.mark({
        class: `cm-flag ${categoryClass} ${severityClass} ${statusClass}`,
        attributes: { "data-flag-id": f.id },
      }).range(from, to),
    );
  }

  return Decoration.set(decos, true);
}

/* ------------------------------------------------------------------ */
/*  Flag data state field (for tooltip access)                         */
/* ------------------------------------------------------------------ */

const flagDataField = StateField.define<FlagState[]>({
  create() {
    return [];
  },
  update(flags, tr) {
    for (const ef of tr.effects) {
      if (ef.is(flagDataEffect)) {
        return ef.value;
      }
    }
    return flags;
  },
});

/* ------------------------------------------------------------------ */
/*  Decoration state field                                             */
/* ------------------------------------------------------------------ */

const flagField = StateField.define<DecorationSet>({
  create() {
    return Decoration.none;
  },
  update(decos, tr) {
    for (const ef of tr.effects) {
      if (ef.is(flagEffect)) {
        return buildDecos(ef.value);
      }
    }
    return decos.map(tr.changes);
  },
  provide: (f) => EditorView.decorations.from(f),
});

/* ------------------------------------------------------------------ */
/*  Gutter marker — dot on flagged lines                              */
/* ------------------------------------------------------------------ */

class FlagGutterMarker extends GutterMarker {
  constructor(private category: Category) {
    super();
  }
  toDOM() {
    const dot = document.createElement("div");
    dot.className = `cm-flag-dot ${this.category === "readability" ? "readability" : ""}`;
    return dot;
  }
}

function flagGutter(dataField: StateField<FlagState[]>) {
  return gutter({
    class: "cm-flag-gutter",
    lineMarker: (view, line) => {
      const flags = view.state.field(dataField);
      try {
        const docLine = view.state.doc.lineAt(line.from);
        const lineNum = docLine.number;
        for (const f of flags) {
          if (f.status === "rejected" || f.status === "applied") continue;
          const flagDocLine = view.state.doc.lineAt(f.span.from);
          if (flagDocLine.number === lineNum) {
            return new FlagGutterMarker(f.category);
          }
        }
      } catch {
        // line may be out of bounds during update
      }
      return null;
    },
    initialSpacer: null,
  });
}

/* ------------------------------------------------------------------ */
/*  Props                                                              */
/* ------------------------------------------------------------------ */

interface Props {
  text: string;
  flags: FlagState[];
  onChange: (text: string) => void;
  onFlagClick?: (flagId: string) => void;
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export default function CodeMirrorEditor({
  text,
  flags,
  onChange,
  onFlagClick,
}: Props) {
  const editorRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  const onFlagClickRef = useRef(onFlagClick);
  // Skip onChange when applying store→editor sync (accept/applyAll), otherwise
  // a full-doc replace re-enters setText and can drop remaining flags.
  const suppressChangeRef = useRef(false);

  // Sync refs after render
  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  useEffect(() => {
    onFlagClickRef.current = onFlagClick;
  }, [onFlagClick]);

  /* ---------- Create editor once ---------- */
  useEffect(() => {
    const el = editorRef.current;
    if (!el) return;

    // Click handler on the editor for flag clicks
    const clickHandler = EditorView.domEventHandlers({
      mousedown: (event, view) => {
        const target = event.target as HTMLElement;
        const flagEl = target.closest("[data-flag-id]") as HTMLElement | null;
        if (flagEl) {
          const flagId = flagEl.getAttribute("data-flag-id");
          if (flagId && onFlagClickRef.current) {
            onFlagClickRef.current(flagId);
          }
        }
      },
    });

    const view = new EditorView({
      doc: text,
      extensions: [
        basicSetup,
        EditorView.lineWrapping,
        flagField,
        flagDataField,
        flagGutter(flagDataField),
        clickHandler,
        EditorView.updateListener.of((update) => {
          if (update.docChanged && !suppressChangeRef.current) {
            onChangeRef.current(update.state.doc.toString());
          }
        }),
      ],
      parent: el,
    });

    viewRef.current = view;

    // Auto-focus editor so cursor blinks on page load
    view.focus();

    return () => {
      view.destroy();
      viewRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  /* ---------- Sync external text changes ---------- */
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    const cur = view.state.doc.toString();
    if (cur !== text) {
      suppressChangeRef.current = true;
      view.dispatch({ changes: { from: 0, to: cur.length, insert: text } });
      suppressChangeRef.current = false;
    }
  }, [text]);

  /* ---------- Sync flag decorations ---------- */
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;

    const activeFlags = flags.filter(
      (f) => f.status !== "applied",
    );

    view.dispatch({
      effects: [
        flagEffect.of({ flags, text }),
        flagDataEffect.of(activeFlags),
      ],
    });
  }, [flags, text]);

  return (
    <div
      ref={editorRef}
      className="w-full"
    />
  );
}
