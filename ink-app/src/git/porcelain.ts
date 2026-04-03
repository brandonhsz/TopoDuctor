import { gitOutput } from "./spawn.js";

export type Worktree = {
  path: string;
  head: string;
  branch: string;
};

function branchShortName(ref: string): string {
  const s = ref.trim();
  const p = "refs/heads/";
  if (s.startsWith(p)) {
    return s.slice(p.length);
  }
  return s;
}

/** Parses `git worktree list --porcelain` (aligned with internal/worktree/git/runner.go). */
export function parsePorcelain(raw: string): Worktree[] {
  const s = raw.trim();
  if (!s) {
    return [];
  }
  const lines = s.split("\n");
  const out: Worktree[] = [];
  let cur: Worktree | null = null;

  const flush = () => {
    if (cur && cur.path) {
      out.push(cur);
    }
    cur = null;
  };

  for (const line of lines) {
    const row = line.replace(/\r$/, "");
    if (row === "") {
      flush();
      continue;
    }
    const sp = row.indexOf(" ");
    const key = sp < 0 ? row : row.slice(0, sp);
    const val = sp < 0 ? "" : row.slice(sp + 1).trim();

    switch (key) {
      case "worktree":
        flush();
        cur = { path: val, head: "", branch: "" };
        break;
      case "HEAD":
        if (cur) {
          cur.head = val;
        }
        break;
      case "branch":
        if (cur) {
          cur.branch = branchShortName(val);
        }
        break;
      case "detached":
        if (cur) {
          cur.branch = "";
        }
        break;
      default:
        break;
    }
  }
  flush();
  return out;
}

export async function listWorktrees(cwd: string): Promise<Worktree[]> {
  const out = await gitOutput(cwd, ["worktree", "list", "--porcelain"]);
  return parsePorcelain(out);
}
