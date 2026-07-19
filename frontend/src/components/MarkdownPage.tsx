"use client";

import { useQuery } from "@tanstack/react-query";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import Link from "next/link";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface PageData {
  slug: string;
  title: string;
  description?: string;
  language?: string;
  content_markdown: string;
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

/** Strip leading `# Title` heading from markdown to avoid duplicate. */
function stripTitleFromMarkdown(markdown: string, title: string): string {
  const escaped = title.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const headingRegex = new RegExp(`^#\\s+${escaped}\\s*$`, "m");
  return markdown.replace(headingRegex, "").trimStart();
}

/* ------------------------------------------------------------------ */
/*  API                                                                */
/* ------------------------------------------------------------------ */

async function fetchPage(slug: string): Promise<PageData> {
  const res = await fetch(`/api/pages/${encodeURIComponent(slug)}`);
  if (!res.ok) {
    if (res.status === 404) throw new Error("not-found");
    throw new Error(`Failed to fetch page: ${res.status}`);
  }
  return res.json() as Promise<PageData>;
}

/* ------------------------------------------------------------------ */
/*  Loading skeleton                                                   */
/* ------------------------------------------------------------------ */

function PageSkeleton() {
  return (
    <div className="mx-auto max-w-[980px] px-8 editor-area animate-pulse">
      <div className="mb-6 h-4 w-28 rounded bg-outline" />
      <div className="mb-2 h-9 w-72 rounded bg-outline" />
      <div className="mb-6 h-5 w-96 rounded bg-outline" />
      <div className="space-y-3">
        <div className="h-5 w-full rounded bg-outline" />
        <div className="h-5 w-5/6 rounded bg-outline" />
        <div className="h-5 w-4/6 rounded bg-outline" />
        <div className="h-5 w-full rounded bg-outline" />
        <div className="h-5 w-3/4 rounded bg-outline" />
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Error / not-found                                                  */
/* ------------------------------------------------------------------ */

function PageNotFound({ slug }: { slug: string }) {
  return (
    <div className="mx-auto max-w-[980px] px-8 editor-area">
      <h1 className="mb-4 font-serif text-[28px] leading-[36px] tracking-[-.01em]">
        Страница не найдена
      </h1>
      <p className="text-[17px] leading-[30px] text-on-surface-variant">
        Страница «{slug}» не существует.
      </p>
      <Link
        href="/pages/"
        className="link-blue mt-6 inline-block text-[15px] underline underline-offset-2 transition-colors hover:text-primary"
      >
        ← Все страницы
      </Link>
    </div>
  );
}

function PageError() {
  return (
    <div className="mx-auto max-w-[980px] px-8 editor-area">
      <h1 className="mb-4 font-serif text-[28px] leading-[36px] tracking-[-.01em]">
        Ошибка загрузки
      </h1>
      <p className="text-[17px] leading-[30px] text-on-surface-variant">
        Не удалось загрузить страницу. Проверьте соединение и попробуйте снова.
      </p>
      <Link
        href="/pages/"
        className="link-blue mt-6 inline-block text-[15px] underline underline-offset-2 transition-colors hover:text-primary"
      >
        ← Все страницы
      </Link>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  MarkdownPage component                                             */
/* ------------------------------------------------------------------ */

export default function MarkdownPage({ slug }: { slug: string }) {
  const {
    data: page,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["page", slug],
    queryFn: () => fetchPage(slug),
    staleTime: 5 * 60 * 1000,
    retry: 1,
  });

  if (isLoading) return <PageSkeleton />;

  if (error?.message === "not-found") return <PageNotFound slug={slug} />;
  if (error || !page) return <PageError />;

  const cleanedMarkdown = page.title
    ? stripTitleFromMarkdown(page.content_markdown, page.title)
    : page.content_markdown;

  return (
    <div className="mx-auto max-w-[980px] px-8 editor-area">
      <Link
        href="/pages/"
        className="mb-6 inline-block text-[15px] text-link-blue underline underline-offset-2 transition-colors hover:text-primary"
      >
        ← Все страницы
      </Link>

      <h1 className="mb-2 font-serif text-[28px] leading-[36px] tracking-[-.01em]">
        {page.title}
      </h1>

      {page.description && (
        <p className="mb-4 text-[17px] leading-[26px] text-on-surface-variant">
          {page.description}
        </p>
      )}

      {page.language && (
        <div className="mb-6">
          <span className="inline-block rounded border border-outline px-2 py-0.5 text-[11px] font-medium uppercase leading-[18px] tracking-[0.06em] text-on-surface-variant">
            {page.language}
          </span>
        </div>
      )}

      <article className="markdown-content">
        <ReactMarkdown remarkPlugins={[remarkGfm]}>
          {cleanedMarkdown}
        </ReactMarkdown>
      </article>
    </div>
  );
}
