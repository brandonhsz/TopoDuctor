import path from "node:path";
import { gitTopLevel } from "./gitMeta.js";

export async function shouldShowLobby(
  seed: string,
  paths: string[]
): Promise<boolean> {
  if (paths.length === 0) {
    return true;
  }
  let absSeed: string;
  try {
    absSeed = path.resolve(seed);
  } catch {
    return true;
  }
  let top: string;
  try {
    top = await gitTopLevel(absSeed);
  } catch {
    return true;
  }
  const cleanTop = path.normalize(top);
  for (const p of paths) {
    if (path.normalize(p) === cleanTop) {
      return false;
    }
  }
  return true;
}
