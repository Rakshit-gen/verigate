import Link from "next/link";
import Logo from "@/components/Logo";
import DocsSidebar from "@/components/docs/DocsSidebar";

export default function DocsLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen">
      <header className="border-b border-[color:var(--border)]">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <Link href="/" className="flex items-center gap-2.5">
            <Logo size={20} />
            <span className="text-sm font-semibold tracking-tight text-[color:var(--text-primary)]">
              Verigate <span className="font-normal text-[color:var(--text-tertiary)]">docs</span>
            </span>
          </Link>
          <Link
            href="/dashboard"
            className="rounded-full px-4 py-1.5 text-xs font-medium transition-colors"
            style={{ background: "var(--accent-soft)", color: "var(--accent)" }}
          >
            Live dashboard →
          </Link>
        </div>
      </header>

      <div className="mx-auto flex max-w-6xl gap-10 px-6 py-10">
        <aside className="hidden w-52 shrink-0 lg:block">
          <div className="sticky top-10">
            <DocsSidebar />
          </div>
        </aside>
        <main className="min-w-0 flex-1 pb-20">{children}</main>
      </div>
    </div>
  );
}
