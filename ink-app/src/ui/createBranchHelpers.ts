import { applyPreferredFirst } from "../projects/branches.js";

export const createBranchVisible = 3;

export function filterBranchNames(all: string[], query: string): string[] {
  const q = query.trim().toLowerCase();
  if (!q) {
    return [...all];
  }
  return all.filter((b) => b.toLowerCase().includes(q));
}

export function adjustBranchScroll(
  cursor: number,
  scroll: number,
  window: number,
  total: number
): number {
  if (total <= 0) {
    return 0;
  }
  if (total <= window) {
    return 0;
  }
  let c = cursor;
  if (c < 0) {
    c = 0;
  }
  if (c >= total) {
    c = total - 1;
  }
  if (c < scroll) {
    return c;
  }
  if (c >= scroll + window) {
    return c - window + 1;
  }
  return scroll;
}

export function filteredCreateBranches(
  all: string[],
  filterQ: string,
  preferred: string[] | undefined
): string[] {
  const sub = filterBranchNames(all, filterQ);
  return applyPreferredFirst(sub, preferred ?? []);
}
