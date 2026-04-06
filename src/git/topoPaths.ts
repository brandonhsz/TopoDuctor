import crypto from "node:crypto";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { sanitizeWorktreeLabel } from "./sanitize.js";

const topoDuctorDir = ".topoDuctor";

async function topoDuctorRoot(): Promise<string> {
  return path.join(os.homedir(), topoDuctorDir);
}

export function projectSegmentName(repoTop: string): string {
  let base = path.basename(repoTop);
  let slug = sanitizeWorktreeLabel(base);
  if (!slug) {
    slug = "repo";
  }
  const sum = crypto.createHash("sha256").update(path.normalize(repoTop)).digest();
  const short = sum.subarray(0, 4).toString("hex");
  return `${slug}-${short}`;
}

/** ~/.topoDuctor/projects/<segment>/worktree/<wtSlug> */
export async function checkoutPathForNewWorktree(
  repoTop: string,
  wtSlug: string
): Promise<string> {
  const root = await topoDuctorRoot();
  const seg = projectSegmentName(repoTop);
  return path.join(root, "projects", seg, "worktree", wtSlug);
}

export async function ensureDirForFile(filePath: string): Promise<void> {
  await fs.mkdir(path.dirname(filePath), { recursive: true, mode: 0o755 });
}
