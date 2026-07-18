"use client";

import type { FlagState } from "@/lib/store";

/* ── Props ── */

interface FlagCardProps {
  flag: FlagState;
  text?: string;
  position: { x: number; y: number };
  onAccept: (id: string) => void;
  onReject: (id: string) => void;
  onClose: () => void;
}

function getContext(text: string, from: number, to: number): { before: string; after: string } {
  const beforeLen = Math.min(40, from);
  const afterLen = Math.min(40, text.length - to);
  return {
    before: text.slice(from - beforeLen, from),
    after: text.slice(to, to + afterLen),
  };
}

/* ── Category label ── */

function categoryLabel(cat: string): string {
  return cat === "cleanliness" ? "чистота" : "читаемость";
}

/* ── Component ── */

export default function FlagCard({
  flag,
  text,
  position,
  onAccept,
  onReject,
  onClose,
}: FlagCardProps) {
  const isActionable =
    flag.status === "tentative" || flag.status === "confirmed" || flag.status === "rejected";

  const hasBadExamples = flag.examples?.bad && flag.examples.bad.length > 0;
  const hasGoodExamples = flag.examples?.good && flag.examples.good.length > 0;
  const maxExamples = Math.max(
    flag.examples?.bad?.length ?? 0,
    flag.examples?.good?.length ?? 0,
  );

  return (
    <div
      className="floating-card"
      style={{
        position: "fixed",
        left: `${position.x}px`,
        top: `${position.y}px`,
      }}
    >
      <div className="inner">
        {/* Header */}
        <div className="flex items-baseline gap-2 mb-2.5">
          <span className="font-serif text-xl leading-[26px] text-[#1a1a1a]">
            «{flag.ruleName}»
          </span>
          <span className="text-[10px] leading-[14px] text-[#6b645a] tracking-[0.06em] uppercase">
            {categoryLabel(flag.category)}
          </span>
        </div>

        {/* Context */}
        {text ? (
          <p className="text-sm leading-[22px] text-[#6b645a] italic mb-4">
            {(() => {
              const ctx = getContext(text, flag.span.from, flag.span.to);
              const showBeforeEllipsis = flag.span.from > 40;
              const showAfterEllipsis = text.length - flag.span.to > 40;
              return (
                <>
                  {showBeforeEllipsis && <span>&hellip;</span>}
                  <span>{ctx.before}</span>
                  <span className="not-italic px-0.5 rounded-sm bg-[#ffe89a]">{flag.anchor.matchText}</span>
                  <span>{ctx.after}</span>
                  {showAfterEllipsis && <span>&hellip;</span>}
                </>
              );
            })()}
          </p>
        ) : (
          <p className="text-sm leading-[22px] text-[#6b645a] italic mb-4">
            &hellip;<span className="not-italic px-0.5 rounded-sm bg-[#ffe89a]">{flag.anchor.matchText}</span>&hellip;
          </p>
        )}

        {/* Suggestion */}
        {flag.suggestion && (
          <p className="text-sm leading-[22px] text-[#6b645a] mb-3.5">
            {flag.suggestion}
          </p>
        )}

        {/* Accept / Reject */}
        {isActionable && (
          <div className="text-[15px] leading-[22px] mb-3.5 pb-3.5 border-b border-outline">
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                if (flag.autoFix !== undefined) onAccept(flag.id);
              }}
              disabled={flag.autoFix === undefined}
              className={"bg-none border-none cursor-pointer font-medium " + (flag.autoFix !== undefined
                ? "text-terra hover:border-b hover:border-terra border-b border-transparent"
                : "text-outline cursor-default")}
            >
              Принять
            </button>
            <span className="text-outline mx-2">·</span>
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onReject(flag.id);
              }}
              className="text-terra no-underline font-medium hover:border-b hover:border-terra border-b border-transparent bg-none cursor-pointer"
            >
              Отклонить
            </button>
          </div>
        )}

        {/* Examples grid + auto suggestion row */}
        {(hasBadExamples || hasGoodExamples || flag.autoFix !== undefined) && (
          <div
            className="grid gap-x-6 gap-y-3.5 mb-3.5"
            style={{ gridTemplateColumns: "1fr 1fr" }}
          >
            <div className="col">
              <h4 className="text-xs leading-[18px] uppercase tracking-[0.04em] font-bold text-[#1a1a1a] mb-1.5">
                Нет
              </h4>
              {Array.from({ length: maxExamples }).map((_, i) => (
                <p
                  key={i}
                  className="text-sm leading-[22px] text-[#1a1a1a] mb-1.5 last:mb-0"
                >
                  {flag.examples?.bad?.[i] ?? "\u00A0"}
                </p>
              ))}
              {flag.autoFix !== undefined && (
                <p className="text-[13px] leading-[22px] text-terra font-medium mb-1.5 last:mb-0">
                  {flag.anchor.matchText}
                </p>
              )}
            </div>
            <div className="col">
              <h4 className="text-xs leading-[18px] uppercase tracking-[0.04em] font-bold text-[#1a1a1a] mb-1.5">
                Да
              </h4>
              {Array.from({ length: maxExamples }).map((_, i) => (
                <p
                  key={i}
                  className="text-sm leading-[22px] text-[#1a1a1a] mb-1.5 last:mb-0"
                >
                  {flag.examples?.good?.[i] ?? "\u00A0"}
                </p>
              ))}
              {flag.autoFix !== undefined && (
                <p className="text-[13px] leading-[22px] text-terra font-medium mb-1.5 last:mb-0">
                  {flag.autoFix === "" ? (<span className="opacity-60">удалить</span>) : flag.autoFix}
                </p>
              )}
            </div>
          </div>
        )}

        {/* See also */}
        {flag.related && flag.related.length > 0 && (
          <div className="text-[13px] leading-[20px] text-[#1a1a1a]">
            См. также:{" "}
            {flag.related.map((r, i) => (
              <span key={r.name}>
                {r.url ? (
                  <a
                    href={r.url}
                    className="text-link-blue underline underline-offset-2"
                  >
                    {r.name}
                  </a>
                ) : (
                  <span>{r.name}</span>
                )}
                {i < flag.related!.length - 1 && ", "}
              </span>
            ))}
          </div>
        )}

        {/* Close button */}
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onClose();
          }}
          className="absolute top-3 right-3 w-5 h-5 flex items-center justify-center text-[#6b645a] hover:text-[#1a1a1a] bg-none border-none cursor-pointer"
          aria-label="Закрыть"
        >
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
            <path d="M2 2l10 10M12 2L2 12" />
          </svg>
        </button>
      </div>
    </div>
  );
}
