import type { Metadata } from "next";
import { Geist, Geist_Mono, PT_Serif } from "next/font/google";
import AppShell from "@/components/AppShell";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

const ptSerif = PT_Serif({
  weight: ["400", "700"],
  variable: "--font-pt-serif",
  subsets: ["latin", "cyrillic"],
});

export const metadata: Metadata = {
  title: "Редполитика β",
  description: "Проверка текста по правилам редполитики. Чистота и читаемость текста.",
  icons: { icon: "/favicon.svg" },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ru" className={`${geistSans.variable} ${geistMono.variable} ${ptSerif.variable}`}>
      <body className="min-h-screen bg-surface text-on-surface antialiased">
        <AppShell>{children}</AppShell>
      </body>
    </html>
  );
}
