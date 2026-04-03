import fs from "node:fs/promises";
import path from "node:path";

export const ProjectConfigDirName = ".topoductor";
export const ProjectConfigFileName = "project.json";
const LegacyProjectConfigFileName = "topoductor.project.json";

export type ProjectScripts = {
  setup?: string;
  run?: string;
  archive?: string;
};

type ProjectConfigFile = {
  scripts: ProjectScripts;
};

function projectConfigPath(repoRoot: string): string {
  return path.join(path.normalize(repoRoot), ProjectConfigDirName, ProjectConfigFileName);
}

function legacyProjectConfigPath(repoRoot: string): string {
  return path.join(path.normalize(repoRoot), LegacyProjectConfigFileName);
}

async function readScriptsFrom(p: string): Promise<ProjectScripts> {
  const data = await fs.readFile(p, "utf8");
  const f = JSON.parse(data) as ProjectConfigFile;
  return f.scripts ?? {};
}

export async function readProjectConfig(
  repoRoot: string
): Promise<ProjectScripts> {
  const p = projectConfigPath(repoRoot);
  try {
    return await readScriptsFrom(p);
  } catch (e) {
    const err = e as NodeJS.ErrnoException;
    if (err.code !== "ENOENT") {
      throw e;
    }
  }
  const legacy = legacyProjectConfigPath(repoRoot);
  try {
    return await readScriptsFrom(legacy);
  } catch (e) {
    const err = e as NodeJS.ErrnoException;
    if (err.code === "ENOENT") {
      return {};
    }
    throw e;
  }
}

export async function saveProjectScripts(
  repoRoot: string,
  s: ProjectScripts
): Promise<void> {
  const root = path.normalize(repoRoot);
  let setup = (s.setup ?? "").trim();
  let run = (s.run ?? "").trim();
  let archive = (s.archive ?? "").trim();
  const p = projectConfigPath(root);
  const dir = path.join(root, ProjectConfigDirName);
  try {
    await fs.unlink(legacyProjectConfigPath(root));
  } catch {
    /* ignore */
  }
  if (!setup && !run && !archive) {
    try {
      await fs.unlink(p);
    } catch (e) {
      const err = e as NodeJS.ErrnoException;
      if (err.code !== "ENOENT") {
        throw e;
      }
    }
    try {
      await fs.rmdir(dir);
    } catch {
      /* ignore */
    }
    return;
  }
  await fs.mkdir(dir, { recursive: true, mode: 0o755 });
  const data = JSON.stringify(
    { scripts: { setup, run, archive } } satisfies ProjectConfigFile,
    null,
    2
  );
  await fs.writeFile(p, data, { mode: 0o644 });
}

export function expandScriptPlaceholders(
  script: string,
  worktreePath: string
): string {
  const q = JSON.stringify(worktreePath);
  return script.replaceAll("{path}", q);
}
