import { spawn } from "node:child_process";
import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

async function findBrew(): Promise<string> {
  try {
    const { stdout } = await execFileAsync("which", ["brew"], {
      encoding: "utf8",
    });
    const p = stdout.trim().split("\n")[0];
    if (p) {
      return p;
    }
  } catch {
    /* empty */
  }
  throw new Error("no se encontró brew en PATH");
}

function runBrew(
  brew: string,
  args: string[],
  signal?: AbortSignal
): Promise<{ out: string; code: number | null }> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    const child = spawn(brew, args, {
      env: process.env,
      stdio: ["ignore", "pipe", "pipe"],
      signal,
    });
    child.stdout?.on("data", (d: Buffer) => chunks.push(d));
    child.stderr?.on("data", (d: Buffer) => chunks.push(d));
    child.on("error", reject);
    child.on("close", (code) => {
      resolve({ out: Buffer.concat(chunks).toString("utf8"), code });
    });
  });
}

/** Homebrew formula (Node CLI), not a cask. */
export async function brewUpgradeFormula(
  formula = "topoductor",
  signal?: AbortSignal
): Promise<string> {
  const brew = await findBrew();
  const parts: string[] = [];

  const u = await runBrew(brew, ["update"], signal);
  parts.push(u.out);
  if (u.code !== 0) {
    const tail = u.out.trim();
    const msg =
      tail.length > 800 ? tail.slice(-800) : tail;
    throw new Error(`brew update: exit ${u.code}: ${msg}`);
  }
  parts.push("\n");
  const g = await runBrew(brew, ["upgrade", formula], signal);
  parts.push(g.out);
  if (g.code !== 0) {
    const tail = g.out.trim();
    const msg =
      tail.length > 800 ? tail.slice(-800) : tail;
    throw new Error(`brew upgrade ${formula}: exit ${g.code}: ${msg}`);
  }
  return parts.join("");
}
