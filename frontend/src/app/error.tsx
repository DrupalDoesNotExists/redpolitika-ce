"use client";

import Link from "next/link";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="mx-auto max-w-[980px] px-8 editor-area">
      <h1 className="font-serif text-[28px] leading-[36px] mb-9 tracking-[-.01em]">
        Что-то пошло не так
      </h1>
      <p className="text-[17px] leading-[30px] text-on-surface-variant mb-9">
        Произошла непредвиденная ошибка. Попробуйте обновить страницу.
      </p>
      <div className="flex items-center gap-4">
        <button
          type="button"
          onClick={reset}
          className="inline-block rounded bg-[#b65a37] px-5 py-2 text-sm font-medium text-white transition-colors hover:bg-[#9a4a2d]"
        >
          Попробовать снова
        </button>
        <Link
          href="/"
          className="text-link-blue underline underline-offset-2 transition-colors hover:text-primary"
        >
          На главную
        </Link>
      </div>
    </div>
  );
}
