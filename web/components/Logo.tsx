// A small geometric mark: two gate-posts with a signal pulse between them —
// "traffic passing through and being read," which is the whole product.
// Inline SVG rather than an icon font/emoji so it renders crisply and
// matches the accent token exactly.
export default function Logo({ size = 22 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <rect x="1" y="3" width="3" height="18" rx="1" fill="var(--accent)" />
      <rect x="20" y="3" width="3" height="18" rx="1" fill="var(--accent)" />
      <path
        d="M6 12 L9.5 12 L11.5 7 L14 17 L16 12 L18 12"
        stroke="var(--accent)"
        strokeWidth="1.75"
        strokeLinecap="round"
        strokeLinejoin="round"
        opacity="0.9"
      />
    </svg>
  );
}
