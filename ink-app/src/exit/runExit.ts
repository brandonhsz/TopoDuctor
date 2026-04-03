import { spawn } from "node:child_process";
import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

export type ExitPayload = {
  kind: "cd" | "cursor" | "custom";
  path: string;
  customTpl?: string;
};

export function expandPathTemplate(tpl: string, p: string): string {
  return tpl.replaceAll("{path}", JSON.stringify(p));
}

export function cursorPrintLine(p: string): string {
  /* deferred check in async */
  return `cursor ${JSON.stringify(p)}`;
}

async function hasCursorCli(): Promise<boolean> {
  try {
    await execFileAsync("which", ["cursor"], { encoding: "utf8" });
    return true;
  } catch {
    return false;
  }
}

export function printOnlyLine(payload: ExitPayload): string {
  switch (payload.kind) {
    case "cd":
      return `cd ${JSON.stringify(payload.path)}`;
    case "cursor":
      return cursorPrintLine(payload.path);
    case "custom":
      return expandPathTemplate(payload.customTpl ?? "", payload.path);
    default:
      return `cd ${JSON.stringify(payload.path)}`;
  }
}

export async function runExitAction(payload: ExitPayload): Promise<void> {
  const { path: targetPath, kind } = payload;

  switch (kind) {
    case "cd":
      await chdirAndExecShell(targetPath);
      return;
    case "cursor":
      await openInCursor(targetPath);
      return;
    case "custom": {
      const line = expandPathTemplate(payload.customTpl ?? "", targetPath);
      await runShellLine(line);
      return;
    }
    default:
      await chdirAndExecShell(targetPath);
  }
}

async function openInCursor(dir: string): Promise<void> {
  if (await hasCursorCli()) {
    await new Promise<void>((resolve, reject) => {
      const p = spawn("cursor", [dir], {
        stdio: "inherit",
        env: process.env,
      });
      p.on("error", reject);
      p.on("close", (code) =>
        code === 0 ? resolve() : reject(new Error(`cursor exit ${code}`))
      );
    });
    return;
  }
  if (process.platform === "darwin") {
    await new Promise<void>((resolve, reject) => {
      const p = spawn("open", ["-a", "Cursor", dir], {
        stdio: "inherit",
        env: process.env,
      });
      p.on("error", reject);
      p.on("close", (code) =>
        code === 0 ? resolve() : reject(new Error(`open exit ${code}`))
      );
    });
    return;
  }
  throw new Error(
    'no se encontró el comando "cursor" en PATH (instala la CLI de Cursor)'
  );
}

async function runShellLine(line: string): Promise<void> {
  if (process.platform === "win32") {
    process.stderr.write(
      "Ejecuta el comando a mano o usa --print-only. Comando:\n" + line + "\n"
    );
    return;
  }
  const shell = process.env.SHELL || "/bin/sh";
  let shellPath = shell;
  try {
    const { stdout } = await execFileAsync("which", [shell], {
      encoding: "utf8",
    });
    const w = stdout.trim().split("\n")[0];
    if (w) {
      shellPath = w;
    }
  } catch {
    shellPath = "/bin/sh";
  }
  await new Promise<void>((resolve, reject) => {
    const p = spawn(shellPath, ["-lc", line], {
      stdio: "inherit",
      env: process.env,
    });
    p.on("error", reject);
    p.on("close", (code) =>
      code === 0 ? resolve() : reject(new Error(`shell exit ${code}`))
    );
  });
}

export async function chdirAndExecShell(targetPath: string): Promise<void> {
  if (process.platform === "win32") {
    process.stderr.write(
      "En Windows no se puede reemplazar el shell desde el programa. Usa --print-only y copia el comando, o:\n"
    );
    process.stdout.write(`cd /d ${JSON.stringify(targetPath)}\n`);
    return;
  }
  let shell = process.env.SHELL;
  if (!shell) {
    shell = process.platform === "darwin" ? "/bin/zsh" : "/bin/bash";
  }
  let shellPath = shell;
  try {
    const { stdout } = await execFileAsync("which", [shell], {
      encoding: "utf8",
    });
    const w = stdout.trim().split("\n")[0];
    if (w) {
      shellPath = w;
    }
  } catch {
    shellPath = "/bin/sh";
  }
  const useInteractive =
    shellPath.includes("bash") || shellPath.includes("zsh");
  const args = useInteractive ? ["-i"] : [];
  await new Promise<void>((resolve, reject) => {
    const child = spawn(shellPath, args, {
      cwd: targetPath,
      stdio: "inherit",
      env: process.env,
    });
    child.on("error", reject);
    child.on("close", (code) => {
      process.exit(code ?? 0);
      resolve();
    });
  });
}
