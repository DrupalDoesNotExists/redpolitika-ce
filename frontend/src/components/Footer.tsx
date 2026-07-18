export default function Footer() {
  return (
    <footer className="mt-12 pt-6 pb-12 border-t border-outline text-center">
      <p className="text-[13px] leading-5 text-[#6b645a]">
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
