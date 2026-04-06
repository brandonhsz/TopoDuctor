import type { Worktree } from "../git/porcelain.js";

/** Límites de runes visibles en tarjeta (nombre / rama). */
export const cardNameMaxRunes = 20;
export const cardBranchMaxRunes = 18;
export const cardOuterW = 26;
export const gridColGap = 2;
export const marqueeTickMs = 200;

export function truncateRunes(s: string, max: number): string {
  const r = [...s];
  if (r.length <= max) {
    return s;
  }
  if (max <= 1) {
    return "…";
  }
  return r.slice(0, max - 1).join("") + "…";
}

export function runeLen(s: string): number {
  return [...s].length;
}

export function truncates(s: string, max: number): boolean {
  return runeLen(s) > max;
}

export function marqueeWindow(s: string, width: number, phase: number): string {
  const r = [...s];
  if (r.length <= width) {
    return s;
  }
  if (width < 1) {
    return "";
  }
  const gap = [...("  ")];
  const loop = [...gap, ...r, ...gap];
  const period = loop.length;
  const double = [...loop, ...loop];
  const shift = phase % period;
  return double.slice(shift, shift + width).join("");
}

export function folderName(wt: Worktree): string {
  return wt.path.split(/[/\\]/).pop() ?? wt.path;
}

export function branchLabel(wt: Worktree): string {
  if (!wt.branch) {
    return "detached";
  }
  return wt.branch;
}

export function selectedNeedsMarquee(wts: Worktree[], cursor: number): boolean {
  if (cursor < 0 || cursor >= wts.length) {
    return false;
  }
  const wt = wts[cursor];
  const fn = folderName(wt);
  const br = branchLabel(wt);
  return (
    truncates(fn, cardNameMaxRunes) || truncates(br, cardBranchMaxRunes)
  );
}

export function cardNameText(
  wt: Worktree,
  selected: boolean,
  marqueePhase: number
): string {
  const fn = folderName(wt);
  if (!selected || !truncates(fn, cardNameMaxRunes)) {
    return truncateRunes(fn, cardNameMaxRunes);
  }
  return marqueeWindow(fn, cardNameMaxRunes, marqueePhase);
}

export function cardBranchText(
  wt: Worktree,
  selected: boolean,
  marqueePhase: number
): string {
  const br = branchLabel(wt);
  if (!selected || !truncates(br, cardBranchMaxRunes)) {
    return truncateRunes(br, cardBranchMaxRunes);
  }
  return marqueeWindow(br, cardBranchMaxRunes, marqueePhase);
}

export function gridTotalWidth(cols: number): number {
  if (cols < 1) {
    return cardOuterW;
  }
  return cols * cardOuterW + (cols - 1) * gridColGap;
}
