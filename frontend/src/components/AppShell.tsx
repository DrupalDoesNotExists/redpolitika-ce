import TopBar from "@/components/TopBar";
import Footer from "@/components/Footer";

export default function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex flex-col bg-[#f4f1ea] text-[#1a1a1a]">
      <TopBar />
      <main className="flex-1">{children}</main>
      <Footer />
    </div>
  );
}
