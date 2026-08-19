import Logo from "./Logo";
import Nav from "./Nav";

export default function AppHeader({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <header className="mb-8 flex flex-wrap items-center justify-between gap-4">
      <div className="flex items-center gap-3">
        <Logo />
        <div>
          <h1 className="text-lg font-semibold leading-tight tracking-tight text-[color:var(--text-primary)]">
            {title}
          </h1>
          <p className="text-xs text-[color:var(--text-secondary)]">{subtitle}</p>
        </div>
      </div>
      <Nav />
    </header>
  );
}
