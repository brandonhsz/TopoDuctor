import fs from "node:fs/promises";
import path from "node:path";
import { userConfigDir } from "../lib/userConfigDir.js";
import type { ProjectsFile } from "./types.js";
import { normalizePreferredBranchesMap } from "./branches.js";

export async function defaultConfigPath(): Promise<string> {
  const dir = path.join(userConfigDir(), "topoductor");
  await fs.mkdir(dir, { recursive: true });
  return path.join(dir, "projects.json");
}

export function normalizePaths(paths: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const p of paths) {
    const abs = path.resolve(p);
    const clean = path.normalize(abs);
    if (seen.has(clean)) {
      continue;
    }
    seen.add(clean);
    out.push(clean);
  }
  return out;
}

export async function loadProjects(configPath: string): Promise<ProjectsFile> {
  try {
    const data = await fs.readFile(configPath, "utf8");
    const raw = JSON.parse(data) as ProjectsFile;
    return {
      paths: Array.isArray(raw.paths) ? raw.paths : [],
      active: typeof raw.active === "string" ? raw.active : "",
      preferred_branches: raw.preferred_branches,
    };
  } catch (e) {
    const err = e as NodeJS.ErrnoException;
    if (err.code === "ENOENT") {
      return { paths: [], active: "" };
    }
    throw e;
  }
}

export async function saveProjects(
  configPath: string,
  f: ProjectsFile
): Promise<void> {
  const normalized: ProjectsFile = {
    paths: normalizePaths(f.paths),
    active: f.active,
  };
  const pref = normalizePreferredBranchesMap(f.preferred_branches);
  if (Object.keys(pref).length > 0) {
    normalized.preferred_branches = pref;
  }
  const data = JSON.stringify(normalized, null, 2);
  await fs.writeFile(configPath, data, { mode: 0o644 });
}
