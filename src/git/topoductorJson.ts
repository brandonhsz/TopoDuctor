import fs from "node:fs/promises";
import path from "node:path";
import { gitOutput } from "./spawn.js";

type OrchestratorState = {
  managed_worktree_path: string;
};

export async function absGitCommonDir(gitCwd: string): Promise<string> {
  let raw = (await gitOutput(gitCwd, ["rev-parse", "--git-common-dir"])).trim();
  if (!path.isAbsolute(raw)) {
    raw = path.resolve(gitCwd, raw);
  }
  return path.normalize(raw);
}

async function readState(file: string): Promise<OrchestratorState | null> {
  try {
    const data = await fs.readFile(file, "utf8");
    return JSON.parse(data) as OrchestratorState;
  } catch {
    return null;
  }
}

async function writeState(file: string, st: OrchestratorState): Promise<void> {
  const data = JSON.stringify(st, null, 2);
  await fs.writeFile(file, data, { mode: 0o644 });
}

export async function syncManagedPath(
  commonGit: string,
  oldPath: string,
  newPath: string
): Promise<void> {
  const statePath = path.join(commonGit, "topoductor.json");
  const st = await readState(statePath);
  if (!st) {
    return;
  }
  if (path.normalize(st.managed_worktree_path) !== path.normalize(oldPath)) {
    return;
  }
  st.managed_worktree_path = newPath;
  await writeState(statePath, st);
}

export async function clearManagedIfMatches(
  commonGit: string,
  wtPath: string
): Promise<void> {
  const statePath = path.join(commonGit, "topoductor.json");
  const st = await readState(statePath);
  if (!st) {
    return;
  }
  if (path.normalize(st.managed_worktree_path) !== path.normalize(wtPath)) {
    return;
  }
  st.managed_worktree_path = "";
  await writeState(statePath, st);
}
