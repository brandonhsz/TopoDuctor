import path from "node:path";

export function normalizePreferredBranchNames(v: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of v) {
    const s = raw.trim();
    if (!s || seen.has(s)) {
      continue;
    }
    seen.add(s);
    out.push(s);
    if (out.length >= 3) {
      break;
    }
  }
  return out;
}

export function normalizePreferredBranchesMap(
  m: Record<string, string[]> | undefined
): Record<string, string[]> {
  if (!m || Object.keys(m).length === 0) {
    return {};
  }
  const out: Record<string, string[]> = {};
  for (const [k, v] of Object.entries(m)) {
    const abs = path.resolve(k);
    const clean = path.normalize(abs);
    const nv = normalizePreferredBranchNames(v);
    if (nv.length === 0) {
      continue;
    }
    out[clean] = nv;
  }
  return out;
}

/** Preferred branch names first (exact ref match), then the rest in original order. */
export function applyPreferredFirst(
  all: string[],
  preferred: string[]
): string[] {
  if (preferred.length === 0) {
    return [...all];
  }
  const seen = new Set<string>();
  const head: string[] = [];
  for (const p of preferred) {
    const q = p.trim();
    if (!q) {
      continue;
    }
    for (const a of all) {
      if (a === q && !seen.has(a)) {
        head.push(a);
        seen.add(a);
        break;
      }
    }
  }
  const rest = all.filter((a) => !seen.has(a));
  return [...head, ...rest];
}
