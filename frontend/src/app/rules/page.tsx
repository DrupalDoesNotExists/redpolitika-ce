"use client";

import { useQuery } from "@tanstack/react-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Link from "next/link";
import { fetchClientRules, type ClientRule } from "@/lib/client-engine";

const queryClient = new QueryClient();

function categoryLabel(cat?: string): string {
  return cat === "readability" ? "Читаемость" : "Чистота";
}

function RuleRow({ rule }: { rule: ClientRule }) {
  const severityClass =
    rule.severity >= 7
      ? "bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300"
      : rule.severity >= 4
        ? "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300"
        : "bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300";

  return (
    <div className="rounded-xl border border-outline bg-surface-container p-4 transition-colors hover:bg-surface-secondary">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1.5">
          {/* Rule ID */}
          <div className="flex items-center gap-2">
            <span className="rounded bg-surface-secondary px-1.5 py-0.5 font-mono text-[11px] font-medium text-on-surface-variant">
              {rule.id}
            </span>
            <span className="rounded bg-surface-secondary px-1.5 py-0.5 text-[10px] text-on-surface-variant/60">
              {rule.type}
            </span>
          </div>

          {/* Description */}
          {rule.description && (
            <p className="text-sm text-on-surface">{rule.description}</p>
          )}

          {/* Pattern / words preview */}
          {rule.type === "regex" && rule.pattern && (
            <p className="font-mono text-[11px] text-on-surface-variant">
              {rule.pattern.length > 60
                ? rule.pattern.slice(0, 60) + "…"
                : rule.pattern}
            </p>
          )}
          {rule.type === "wordlist" && rule.words && (
            <p className="text-[11px] text-on-surface-variant">
              {rule.words.slice(0, 5).join(", ")}
              {rule.words.length > 5 && ` … +${rule.words.length - 5}`}
            </p>
          )}
        </div>

        {/* Badges */}
        <div className="flex shrink-0 flex-wrap items-center gap-1.5">
          <span className={`rounded px-1.5 py-0.5 font-mono text-[10px] ${severityClass}`}>
            {rule.severity}/10
          </span>
          <span className="rounded bg-surface-secondary px-1.5 py-0.5 text-[10px] text-on-surface-variant">
            {categoryLabel(rule.category)}
          </span>
        </div>
      </div>
    </div>
  );
}

function RulesContent() {
  const { data: rules, isLoading, error } = useQuery({
    queryKey: ["client-rules"],
    queryFn: fetchClientRules,
    staleTime: 5 * 60 * 1000,
  });

  return (
    <div className="mx-auto max-w-[720px] px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <Link
          href="/"
          className="mb-4 inline-flex items-center gap-1 text-xs text-on-surface-variant hover:text-on-surface"
        >
          <svg className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="m15 18-6-6 6-6" />
          </svg>
          Назад к редактору
        </Link>
        <h1 className="text-xl font-bold text-on-surface">Правила проверки</h1>
        <p className="mt-1 text-sm text-on-surface-variant">
          Загруженные клиентские правила для проверки текста
        </p>
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="py-16 text-center">
          <p className="text-sm text-on-surface-variant/50">Загрузка правил…</p>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300">
          Не удалось загрузить правила: {(error as Error).message}
        </div>
      )}

      {/* Rules list */}
      {rules && (
        <div className="space-y-3">
          <p className="text-xs text-on-surface-variant/60">
            Всего правил: {rules.length}
          </p>
          {rules.map((rule) => (
            <RuleRow key={rule.id} rule={rule} />
          ))}
        </div>
      )}
    </div>
  );
}

export default function RulesPage() {
  return (
    <QueryClientProvider client={queryClient}>
      <RulesContent />
    </QueryClientProvider>
  );
}
