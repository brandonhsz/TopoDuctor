import path from "node:path";
import { gitOutput } from "./spawn.js";
import {
  checkoutPathForNewWorktree,
  ensureDirForFile,
} from "./topoPaths.js";
import { sanitizeWorktreeLabel } from "./sanitize.js";
import {
  absGitCommonDir,
  clearManagedIfMatches,
  syncManagedPath,
} from "./topoductorJson.js";
export { listWorktrees, type Worktree } from "./porcelain.js";

async function assertInsideRepo(dir: string): Promise<void> {
  const out = await gitOutput(dir, ["rev-parse", "--is-inside-work-tree"]);
  if (out.trim() !== "true") {
    throw new Error("not a git repository");
  }
}

async function absGitOutput(gitCwd: string, args: string[]): Promise<string> {
  const out = await gitOutput(gitCwd, args);
  const s = out.trim();
  if (!s) {
    throw new Error("empty git output");
  }
  return path.resolve(s);
}

/** Lists local + remote-tracking branch short names, sorted. */
export async function listBranches(gitCwd: string): Promise<string[]> {
  const out = await gitOutput(gitCwd, [
    "for-each-ref",
    "--sort=refname",
    "--format=%(refname:short)",
    "refs/heads",
    "refs/remotes",
  ]);
  const lines = out
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
  const seen = new Set<string>();
  const res: string[] = [];
  for (const line of lines) {
    if (line.endsWith("/HEAD")) {
      continue;
    }
    if (seen.has(line)) {
      continue;
    }
    seen.add(line);
    res.push(line);
  }
  return res;
}

export async function addUserWorktree(
  gitCwd: string,
  baseRef: string,
  label: string
): Promise<void> {
  await assertInsideRepo(gitCwd);
  baseRef = baseRef.trim();
  if (!baseRef) {
    throw new Error("indica la rama base (ej. main)");
  }
  try {
    await gitOutput(gitCwd, ["rev-parse", "--verify", baseRef]);
  } catch (e) {
    throw new Error(
      `rama base no válida: ${e instanceof Error ? e.message : String(e)}`
    );
  }
  const slug = sanitizeWorktreeLabel(label);
  if (!slug) {
    throw new Error("nombre inválido: usa letras, números, -, _ o .");
  }
  const top = await absGitOutput(gitCwd, ["rev-parse", "--show-toplevel"]);
  const newPath = await checkoutPathForNewWorktree(top, slug);
  await ensureDirForFile(newPath);
  try {
    await gitOutput(gitCwd, [
      "worktree",
      "add",
      "-b",
      slug,
      newPath,
      baseRef,
    ]);
  } catch (e) {
    throw new Error(
      `git worktree add: ${e instanceof Error ? e.message : String(e)}`
    );
  }
}

export async function moveWorktree(
  gitCwd: string,
  oldPath: string,
  newBasename: string
): Promise<void> {
  await assertInsideRepo(gitCwd);
  const slug = sanitizeWorktreeLabel(newBasename);
  if (!slug) {
    throw new Error("nombre inválido");
  }
  const newPath = path.join(path.dirname(oldPath), slug);
  if (path.normalize(oldPath) === path.normalize(newPath)) {
    return;
  }
  const top = await absGitOutput(gitCwd, ["rev-parse", "--show-toplevel"]);
  try {
    await gitOutput(top, ["worktree", "move", oldPath, newPath]);
  } catch (e) {
    throw new Error(
      `git worktree move: ${e instanceof Error ? e.message : String(e)}`
    );
  }
  let commonGit: string;
  try {
    commonGit = await absGitCommonDir(gitCwd);
  } catch {
    return;
  }
  try {
    await syncManagedPath(commonGit, oldPath, newPath);
  } catch (e) {
    throw new Error(
      `actualizar estado del orquestador: ${e instanceof Error ? e.message : String(e)}`
    );
  }
}

export async function removeWorktree(
  gitCwd: string,
  wtPath: string
): Promise<void> {
  await assertInsideRepo(gitCwd);
  const top = await absGitOutput(gitCwd, ["rev-parse", "--show-toplevel"]);
  const commonGit = await absGitOutput(gitCwd, ["rev-parse", "--git-common-dir"]);
  const commonAbs = path.normalize(commonGit);

  try {
    await gitOutput(top, ["worktree", "remove", wtPath]);
  } catch {
    try {
      await gitOutput(top, ["worktree", "remove", "--force", wtPath]);
    } catch (e) {
      throw new Error(
        `git worktree remove: ${e instanceof Error ? e.message : String(e)}`
      );
    }
  }
  await clearManagedIfMatches(commonAbs, wtPath);
}
