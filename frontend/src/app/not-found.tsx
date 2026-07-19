import Link from "next/link";

export default function NotFound() {
  return (
    <div className="mx-auto max-w-[980px] px-8 editor-area">
      <h1 className="font-serif text-[28px] leading-[36px] mb-9 tracking-[-.01em]">
        Страница не найдена
      </h1>
      <p className="text-[17px] leading-[30px] text-on-surface-variant mb-9">
        Запрашиваемая страница не существует или была удалена.
      </p>
      <Link
        href="/"
        className="text-link-blue underline underline-offset-2 transition-colors hover:text-primary"
      >
        На главную
      </Link>
    </div>
  );
}
