"use client";

import type { FlagState } from "@/lib/store";

interface CodifierProps {
  flags: Record<string, FlagState>;
  onApplyAll?: () => void;
  isApplyingAll?: boolean;
  acceptedCount?: number;
}

export default function Codifier({ flags, onApplyAll, isApplyingAll, acceptedCount }: CodifierProps) {
  const activeFlags = Object.values(flags).filter(
    (f) => f.status !== "applied",
  );

  // Deduplicate by ruleId, keeping first occurrence
  const seen = new Set<string>();
  const uniqueRules: { ruleId: string; ruleName: string; ruleUrl?: string; allRejected: boolean }[] = [];
  for (const f of activeFlags) {
    if (!seen.has(f.ruleId)) {
      seen.add(f.ruleId);
      // Check all flags for this rule — if ALL are rejected, mark as greyed out
      const allForRule = Object.values(flags).filter(
        (x) => x.ruleId === f.ruleId && x.status !== "applied",
      );
      const allRejected = allForRule.length > 0 && allForRule.every((x) => x.status === "rejected");
      uniqueRules.push({ ruleId: f.ruleId, ruleName: f.ruleName, ruleUrl: f.ruleUrl, allRejected });
    }
  }

  if (uniqueRules.length === 0) return null;

  return (
    <div className="codifier">
      <span className="what">Что не так: </span>
      <span className="list">
        {uniqueRules.map((rule, i) => (
          <span key={rule.ruleId}>
            {rule.allRejected ? (
              <span className="opacity-50">{rule.ruleName}</span>
            ) : rule.ruleUrl ? (
              <a
                href={rule.ruleUrl}
                title={rule.ruleId}
              >
                {rule.ruleName}
              </a>
            ) : (
              <span>{rule.ruleName}</span>
            )}
            {i < uniqueRules.length - 1 && ", "}
          </span>
        ))}
      </span>

      {/* Apply All — inline text link after list */}
      {onApplyAll && acceptedCount != null && acceptedCount > 0 && (
        <>
          <span className="text-outline mx-1">·</span>
          <button
            type="button"
            onClick={onApplyAll}
            disabled={isApplyingAll}
            className="text-terra underline underline-offset-2 hover:no-underline bg-none border-none cursor-pointer text-[13px] font-medium disabled:opacity-50"
          >
            {isApplyingAll
              ? "Применение…"
              : `Применить все (${acceptedCount})`}
          </button>
        </>
      )}
    </div>
  );
}
