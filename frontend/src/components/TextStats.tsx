"use client";

import type { FlagState } from "@/lib/store";

interface TextStatsProps {
  wordCount: number;
  sentenceCount: number;
  charCount: number;
  flags: Record<string, FlagState>;
}

export default function TextStats({
  wordCount,
  sentenceCount,
  charCount,
  flags,
}: TextStatsProps) {
  const totalViolations = Object.values(flags).filter(
    (f) => f.status !== "rejected" && f.status !== "applied",
  ).length;

  return (
    <div className="text-stats text-right">
      {wordCount} слов · {sentenceCount} предложений · {charCount.toLocaleString("ru-RU")} знаков
      <span className="violations">
        {totalViolations} {totalViolations === 1 ? "нарушение" : totalViolations >= 2 && totalViolations <= 4 ? "нарушения" : "нарушений"}
      </span>
    </div>
  );
}
