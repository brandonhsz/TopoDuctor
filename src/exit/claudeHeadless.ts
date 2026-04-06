import { spawn } from "node:child_process";
import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

const maxCaptureChars = 512 * 1024;

async function assertClaudeOnPath(): Promise<void> {
  try {
    if (process.platform === "win32") {
      await execFileAsync("where", ["claude"], { encoding: "utf8" });
    } else {
      await execFileAsync("which", ["claude"], { encoding: "utf8" });
    }
  } catch {
    throw new Error(
      'No se encontró "claude" en PATH (instala la CLI de Claude Code)'
    );
  }
}

export type ClaudeHeadlessResult = {
  stdout: string;
  stderr: string;
  exitCode: number | null;
};

/**
 * Runs `claude -p` with piped stdio (Agent SDK / "headless" mode).
 * This is not the full-screen interactive Claude Code TUI; it's the supported way
 * to capture output without owning the terminal.
 */
export async function runClaudeHeadlessPrompt(
  cwd: string,
  prompt: string,
  signal?: AbortSignal
): Promise<ClaudeHeadlessResult> {
  const p = prompt.trim();
  if (!p) {
    throw new Error("El prompt no puede estar vacío.");
  }
  await assertClaudeOnPath();

  return await new Promise((resolve, reject) => {
    let settled = false;
    const outChunks: Buffer[] = [];
    const errChunks: Buffer[] = [];

    const child = spawn("claude", ["-p", p, "--output-format", "text"], {
      cwd,
      env: process.env,
      stdio: ["ignore", "pipe", "pipe"],
    });

    const finish = (fn: () => void) => {
      if (settled) {
        return;
      }
      settled = true;
      if (signal) {
        signal.removeEventListener("abort", onAbort);
      }
      fn();
    };

    const onAbort = () => {
      child.kill("SIGTERM");
    };

    if (signal) {
      if (signal.aborted) {
        finish(() => reject(new Error("Cancelado")));
        return;
      }
      signal.addEventListener("abort", onAbort);
    }

    child.stdout?.on("data", (b: Buffer) => {
      outChunks.push(b);
    });
    child.stderr?.on("data", (b: Buffer) => {
      errChunks.push(b);
    });

    child.on("error", (err) => {
      finish(() => {
        reject(
          err instanceof Error && "code" in err && err.code === "ENOENT"
            ? new Error(
                'No se encontró "claude" en PATH (instala la CLI de Claude Code)'
              )
            : err instanceof Error
              ? err
              : new Error(String(err))
        );
      });
    });

    child.on("close", (exitCode) => {
      finish(() => {
        if (signal?.aborted) {
          reject(new Error("Cancelado"));
          return;
        }
        let stdout = Buffer.concat(outChunks).toString("utf8");
        if (stdout.length > maxCaptureChars) {
          stdout =
            stdout.slice(0, maxCaptureChars) + "\n… [salida truncada]";
        }
        let stderr = Buffer.concat(errChunks).toString("utf8");
        if (stderr.length > maxCaptureChars) {
          stderr =
            stderr.slice(0, maxCaptureChars) + "\n… [stderr truncado]";
        }
        resolve({
          stdout,
          stderr,
          exitCode,
        });
      });
    });
  });
}
