"use client";

import { useQuery } from "@tanstack/react-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Link from "next/link";

const queryClient = new QueryClient();

interface VersionInfo {
  version: string;
  commit?: string;
  build_time?: string;
  license?: string;
}

async function fetchVersion(): Promise<VersionInfo> {
  const res = await fetch("/api/version");
  if (!res.ok) throw new Error(`Failed to fetch version: ${res.status}`);
  return res.json();
}

function AboutContent() {
  const { data: version, isLoading, error } = useQuery({
    queryKey: ["version"],
    queryFn: fetchVersion,
    staleTime: 60 * 60 * 1000, // 1 hour
  });

  return (
    <div className="mx-auto max-w-[600px] px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <Link
          href="/"
          className="mb-4 inline-flex items-center gap-1 text-xs text-on-surface-variant hover:text-on-surface"
        >
          <svg
            className="h-3.5 w-3.5"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="m15 18-6-6 6-6" />
          </svg>
          Назад к редактору
        </Link>
        <h1 className="text-xl font-bold text-on-surface">О программе</h1>
      </div>

      {/* Brand */}
      <div className="mb-8 flex items-center gap-4">
        <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary text-base font-bold text-white">
          R
        </div>
        <div>
          <h2 className="text-lg font-semibold text-on-surface">
            Redpolitika CE
          </h2>
          <p className="text-sm text-on-surface-variant">
            Проверка текста по правилам редполитики
          </p>
        </div>
      </div>

      {/* Version */}
      <div className="space-y-4 rounded-xl border border-outline bg-surface-container p-6">
        <h3 className="text-sm font-semibold text-on-surface">
          Информация о версии
        </h3>

        {isLoading && (
          <p className="text-sm text-on-surface-variant/50">Загрузка…</p>
        )}

        {error && (
          <p className="text-sm text-red-600 dark:text-red-400">
            Не удалось загрузить информацию о версии
          </p>
        )}

        {version && (
          <dl className="space-y-3">
            <div className="flex justify-between">
              <dt className="text-sm text-on-surface-variant">Версия</dt>
              <dd className="text-sm font-mono text-on-surface">
                {version.version}
              </dd>
            </div>
            {version.commit && (
              <div className="flex justify-between">
                <dt className="text-sm text-on-surface-variant">Commit</dt>
                <dd className="text-sm font-mono text-on-surface">
                  {version.commit.slice(0, 8)}
                </dd>
              </div>
            )}
            {version.build_time && (
              <div className="flex justify-between">
                <dt className="text-sm text-on-surface-variant">Сборка</dt>
                <dd className="text-sm text-on-surface">
                  {version.build_time}
                </dd>
              </div>
            )}
            {version.license && (
              <div className="flex justify-between">
                <dt className="text-sm text-on-surface-variant">Лицензия</dt>
                <dd className="text-sm text-on-surface">{version.license}</dd>
              </div>
            )}
          </dl>
        )}
      </div>

      {/* License info */}
      <div className="mt-6 rounded-xl border border-outline bg-surface-container p-6">
        <h3 className="mb-2 text-sm font-semibold text-on-surface">
          Лицензия
        </h3>
        <p className="text-xs leading-relaxed text-on-surface-variant">
          Redpolitika CE распространяется по лицензии BSL (Business Source
          License). Дополнительные условия использования — в файле LICENSE.
        </p>
        <p className="mt-3 text-xs leading-relaxed text-on-surface-variant">
          Версии Enterprise Edition доступны по отдельной лицензии.
        </p>
      </div>

      {/* Footer link */}
      <div className="mt-8 text-center">
        <Link
          href="/"
          className="text-xs text-on-surface-variant/50 hover:text-on-surface-variant"
        >
          &larr; Вернуться в редактор
        </Link>
      </div>
    </div>
  );
}

export default function AboutPage() {
  return (
    <QueryClientProvider client={queryClient}>
      <AboutContent />
    </QueryClientProvider>
  );
}
