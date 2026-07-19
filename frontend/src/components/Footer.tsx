import Link from "next/link";

export default function Footer() {
  return (
    <footer className="mt-12 border-t border-outline pt-6 pb-12 text-center">
      <p className="text-[13px] leading-5 text-[#6b645a]">
        <Link
          href="/about/"
          className="text-[#6b645a] no-underline hover:text-[#1a1a1a]"
        >
          О программе
        </Link>
        <span className="mx-2 text-[#c0b8a8]" aria-hidden>
          ·
        </span>
        Работает на движке{" "}
        <a
          href="https://github.com/drupaldoesnotexists/redpolitika-ce"
          target="_blank"
          rel="noopener noreferrer"
          className="footer-brand no-underline"
        >
          Редполитика<sup>β</sup>
        </a>
      </p>
    </footer>
  );
}
