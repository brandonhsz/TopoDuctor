import path from "node:path";
import { gitOutput } from "../git/spawn.js";

export async function isGitRepo(dir: string): Promise<boolean> {
  try {
    const out = await gitOutput(dir, ["rev-parse", "--is-inside-work-tree"]);
    return out.trim() === "true";
  } catch {
    return false;
  }
}

export async function gitTopLevel(dir: string): Promise<string> {
  const out = await gitOutput(dir, ["rev-parse", "--show-toplevel"]);
  const top = out.trim();
  if (!top) {
    throw new Error("git toplevel vacío");
  }
  return path.resolve(top);
}
