import { clamp } from "./clamp.js";

/** Like Go `modalMaxWidth` / `wrapModal` content width. */
export function modalMaxWidth(termW: number): number {
  const tw = termW < 1 ? 80 : termW;
  return clamp(tw - 6, 36, 78);
}
