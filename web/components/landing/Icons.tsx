// Small, consistent line icons (20x20, 1.6 stroke) — geometric and plain
// rather than a full icon library, kept in one file since there are only six.
const common = { width: 20, height: 20, viewBox: "0 0 20 20", fill: "none", stroke: "currentColor", strokeWidth: 1.6, strokeLinecap: "round" as const, strokeLinejoin: "round" as const };

export function IconEval() {
  return (
    <svg {...common}>
      <path d="M2 14 L6 8 L9 12 L13 5 L18 10" />
      <circle cx="13" cy="5" r="1.4" fill="currentColor" stroke="none" />
    </svg>
  );
}

export function IconCache() {
  return (
    <svg {...common}>
      <ellipse cx="10" cy="5" rx="6.5" ry="2.5" />
      <path d="M3.5 5 V15 C3.5 16.4 6.4 17.5 10 17.5 C13.6 17.5 16.5 16.4 16.5 15 V5" />
      <path d="M3.5 10 C3.5 11.4 6.4 12.5 10 12.5 C13.6 12.5 16.5 11.4 16.5 10" />
    </svg>
  );
}

export function IconShield() {
  return (
    <svg {...common}>
      <path d="M10 2 L17 5 V10 C17 14 14 16.5 10 18 C6 16.5 3 14 3 10 V5 Z" />
      <path d="M7 10 L9 12 L13.5 7.5" />
    </svg>
  );
}

export function IconLock() {
  return (
    <svg {...common}>
      <rect x="4" y="9" width="12" height="8" rx="1.5" />
      <path d="M6.5 9 V6.5 a3.5 3.5 0 0 1 7 0 V9" />
    </svg>
  );
}

export function IconUsers() {
  return (
    <svg {...common}>
      <circle cx="7" cy="7" r="2.8" />
      <path d="M2 17 C2 13.5 4.2 11.5 7 11.5 C9.8 11.5 12 13.5 12 17" />
      <circle cx="15" cy="8" r="2.2" />
      <path d="M13.5 11.5 C16.2 11.7 18 13.6 18 17" />
    </svg>
  );
}

export function IconPulse() {
  return (
    <svg {...common}>
      <path d="M2 10 H6 L8 4 L12 16 L14 10 H18" />
    </svg>
  );
}
