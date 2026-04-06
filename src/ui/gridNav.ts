import { cardOuterW, gridColGap } from "./cardText.js";

export { cardOuterW, gridColGap };

export function gridCols(termW: number): number {
  let tw = termW < 1 ? 80 : termW;
  const usable = tw - 12;
  if (usable < cardOuterW) {
    return 1;
  }
  let c = Math.floor(usable / (cardOuterW + gridColGap));
  if (c < 1) {
    c = 1;
  }
  if (c > 6) {
    c = 6;
  }
  return c;
}

export function withGridCursor(
  cursor: number,
  n: number,
  cols: number,
  dx: number,
  dy: number
): number {
  if (n === 0) {
    return cursor;
  }
  if (cols < 1) {
    cols = 1;
  }
  const row = Math.floor(cursor / cols);
  const col = cursor % cols;

  if (dx < 0 && col > 0) {
    return cursor - 1;
  }
  if (dx > 0 && col < cols - 1 && cursor + 1 < n) {
    return cursor + 1;
  }
  if (dy < 0 && row > 0) {
    const next = cursor - cols;
    return next < 0 ? 0 : next;
  }
  if (dy > 0) {
    const next = cursor + cols;
    if (next < n) {
      return next;
    }
  }
  return cursor;
}
