"use client";

import Link from "next/link";

export default function TopBar() {
  return (
    <header className="border-b border-outline">
      <div className="mx-auto flex h-14 max-w-[980px] items-center gap-3 px-8">
        {/* Logo — serif font, black text, terracotta β */}
        <Link href="/" className="inline-flex items-center no-underline">
          <span className="font-serif text-[22px] leading-[28px] text-[#1a1a1a] tracking-[-0.01em]">
            Редполитика<sup className="text-terra text-[0.55em] font-semibold leading-none align-super ml-[1px]">β</sup>
          </span>
        </Link>
      </div>
    </header>
  );
}
