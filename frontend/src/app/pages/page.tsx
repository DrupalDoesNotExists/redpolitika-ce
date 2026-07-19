"use client";

import { useQuery } from "@tanstack/react-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Link from "next/link";

interface PageSummary {
  slug: string;
  title: string;
  description?: string;
  language?: string;
}

async function fetchPages(): Promise<PageSummary[]> {
  const res = await fetch("/api/pages");
  if (!res.ok) {
    throw new Error(`Failed to fetch pages: ${res.status}`);
  }
  return res.json() as Promise<PageSummary[]>;
}

const queryClient = new QueryClient();

function PagesContent() {
  const { data: pages, isLoading, error } = useQuery({
    queryKey: ["pages"],
    queryFn: fetchPages,
    staleTime: 5 * 60 * 1000,
  });

  return (
    <div className="mx-auto max-w-[980px] px-8 editor-area">
      <h1 className="mb-9 font-serif text-[28px] leading-[36px] tracking-[-.01em]">
        Страницы
      </h1>

      {isLoading && (
        <div className="animate-pulse space-y-3">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-6 w-64 rounded bg-outline" />
          ))}
        </div>
      )}

      {error && (
        <p className="text-[17px] leading-[30px] text-on-surface-variant">
          Не удалось загрузить список страниц.
        </p>
      )}

      {pages && pages.length === 0 && (
        <p className="text-[17px] leading-[30px] text-on-surface-variant">
          Нет доступных страниц.
        </p>
      )}

      {pages && pages.length > 0 && (
        <ul className="space-y-4">
          {pages.map((page) => (
            <li key={page.slug}>
              <div className="flex items-baseline gap-2">
                <Link
                  href={`/pages/${page.slug}/`}
                  className="text-[17px] leading-[30px] text-link-blue underline underline-offset-2 transition-colors hover:text-primary"
                >
                  {page.title}
                </Link>
                {page.language && (
                  <span className="inline-block rounded border border-outline px-2 py-0.5 text-[11px] font-medium uppercase leading-[18px] tracking-[0.06em] text-on-surface-variant">
                    {page.language}
                  </span>
                )}
              </div>
              {page.description && (
                <p className="mt-0.5 text-[15px] leading-[24px] text-on-surface-variant">
                  {page.description}
                </p>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export default function PagesPage() {
  return (
    <QueryClientProvider client={queryClient}>
      <PagesContent />
    </QueryClientProvider>
  );
}
