"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Link from "next/link";

interface PageSummary {
  slug: string;
  title: string;
  description?: string;
  language?: string;
  weight?: number;
  updated_at?: string;
  is_index?: boolean;
}

interface PageGroup {
  dir: string;
  pages: PageSummary[];
}

async function fetchPages(): Promise<PageSummary[]> {
  const res = await fetch("/api/pages");
  if (!res.ok) {
    throw new Error(`Failed to fetch pages: ${res.status}`);
  }
  return res.json() as Promise<PageSummary[]>;
}

const queryClient = new QueryClient();

/* ------------------------------------------------------------------ */
/*  Grouping + sorting helpers                                         */
/* ------------------------------------------------------------------ */

function extractDir(slug: string): string {
  const slashIdx = slug.lastIndexOf("/");
  return slashIdx === -1 ? "" : slug.slice(0, slashIdx);
}

/** Sort pages within a group: weighted (weight > 0) before unweighted. */
function sortPages(pages: PageSummary[]): PageSummary[] {
  return [...pages].sort((a, b) => {
    const aW = a.weight && a.weight > 0 ? a.weight : null;
    const bW = b.weight && b.weight > 0 ? b.weight : null;

    // Weighted pages come before unweighted
    if (aW !== null && bW === null) return -1;
    if (aW === null && bW !== null) return 1;
    // Both weighted → sort by weight asc, then slug
    if (aW !== null && bW !== null) {
      if (aW !== bW) return aW - bW;
    }
    // Fallback: alphabetical by slug
    return a.slug.localeCompare(b.slug);
  });
}

interface GroupSortKey {
  hasWeight: boolean;
  weight: number;
  name: string;
}

function groupSortKey(
  group: PageGroup,
  indexPages: Map<string, PageSummary>,
): GroupSortKey {
  const idx = indexPages.get(group.dir);
  if (idx && idx.weight && idx.weight > 0) {
    return { hasWeight: true, weight: idx.weight, name: group.dir };
  }
  const first = group.pages[0];
  if (first && first.weight && first.weight > 0) {
    return { hasWeight: true, weight: first.weight, name: group.dir };
  }
  return { hasWeight: false, weight: 0, name: group.dir };
}

function sortGroups(
  groups: PageGroup[],
  indexPages: Map<string, PageSummary>,
): PageGroup[] {
  return [...groups].sort((a, b) => {
    const ka = groupSortKey(a, indexPages);
    const kb = groupSortKey(b, indexPages);

    if (ka.hasWeight !== kb.hasWeight) {
      return ka.hasWeight ? -1 : 1; // weighted sections first
    }
    if (ka.hasWeight) {
      if (ka.weight !== kb.weight) return ka.weight - kb.weight;
    }
    return ka.name.localeCompare(kb.name);
  });
}

function buildGroups(
  pages: PageSummary[],
): { indexPages: Map<string, PageSummary>; groups: PageGroup[] } {
  const indexPages = new Map<string, PageSummary>();
  const regular: PageSummary[] = [];

  for (const page of pages) {
    if (page.is_index) {
      const dir = page.slug.endsWith("/_index")
        ? page.slug.slice(0, -7)
        : "";
      indexPages.set(dir, page);
    } else {
      regular.push(page);
    }
  }

  if (regular.length === 0) return { indexPages, groups: [] };

  // Group by directory
  const groupMap = new Map<string, PageSummary[]>();
  for (const page of regular) {
    const dir = extractDir(page.slug);
    if (!groupMap.has(dir)) groupMap.set(dir, []);
    groupMap.get(dir)!.push(page);
  }

  // Sort pages inside each group
  const groups: PageGroup[] = [];
  for (const [dir, dirPages] of groupMap) {
    groups.push({ dir, pages: sortPages(dirPages) });
  }

  return { indexPages, groups: sortGroups(groups, indexPages) };
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

function PagesContent() {
  const { data: pages, isLoading, error } = useQuery({
    queryKey: ["pages"],
    queryFn: fetchPages,
    staleTime: 5 * 60 * 1000,
  });

  const { indexPages, groups } = useMemo(
    () => (pages ? buildGroups(pages) : { indexPages: new Map(), groups: [] }),
    [pages],
  );

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

      {pages && pages.filter((p) => !p.is_index).length === 0 && (
        <p className="text-[17px] leading-[30px] text-on-surface-variant">
          Нет доступных страниц.
        </p>
      )}

      {groups.length > 0 && (
        <div className="space-y-8">
          {groups.map((group) => {
            const indexPage = indexPages.get(group.dir);
            return (
              <section key={group.dir}>
                {indexPage && (
                  <div className="mb-4">
                    <h2 className="font-serif text-[22px] leading-[28px] tracking-[-.015em]">
                      {indexPage.title}
                    </h2>
                    {indexPage.description && (
                      <p className="mt-1 text-[15px] leading-[24px] text-on-surface-variant">
                        {indexPage.description}
                      </p>
                    )}
                  </div>
                )}

                <ul className="space-y-4">
                  {group.pages.map((page) => (
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
              </section>
            );
          })}
        </div>
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
