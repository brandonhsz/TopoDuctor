import { spawn } from "node:child_process";
import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

export type ExitPayload = {
  kind: "cd" | "cursor" | "claude" | "custom";
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
    case "claude":
      return `(cd ${JSON.stringify(payload.path)} && claude)`;
    case "custom":
      return expandPathTemplate(payload.customTpl ?? "", payload.path);
  }
}

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

function escapeForAppleScriptDoubleQuoted(s: string): string {
  return s.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}

function runDetachedZeroExit(
  cmd: string,
  args: string[],
  options?: { shell?: boolean }
): Promise<void> {
  return new Promise((resolve, reject) => {
    const p = spawn(cmd, args, {
      stdio: "ignore",
      detached: true,
      windowsHide: process.platform === "win32",
      shell: options?.shell ?? false,
    });
    p.on("error", reject);
    p.on("close", (code) => {
      p.unref();
      if (code === 0) {
        resolve();
      } else {
        reject(
          new Error(`${cmd} terminó con código ${code ?? "desconocido"}`)
        );
      }
    });
  });
}

/** Opens Claude Code in a separate terminal window so Ink keeps this tty. */
export async function launchClaudeInExternalTerminal(
  dir: string
): Promise<void> {
  await assertClaudeOnPath();
  if (process.platform === "darwin") {
    const inner = `cd ${JSON.stringify(dir)} && claude`;
    const escaped = escapeForAppleScriptDoubleQuoted(inner);
    await runDetachedZeroExit("osascript", [
      "-e",
      'tell application "Terminal" to activate',
      "-e",
      `tell application "Terminal" to do script "${escaped}"`,
    ]);
    return;
  }
  if (process.platform === "win32") {
    try {
      await runDetachedZeroExit("wt.exe", ["-d", dir, "claude"], {
        shell: true,
      });
      return;
    } catch {
      /* try start cmd */
    }
    await runDetachedZeroExit("cmd.exe", [
      "/c",
      "start",
      "",
      "cmd",
      "/k",
      `cd /d ${JSON.stringify(dir)} && claude`,
    ]);
    return;
  }
  const tpl = process.env.TOPODUCTOR_TERMINAL?.trim();
  if (tpl) {
    const line = tpl.replaceAll("{dir}", JSON.stringify(dir));
    await runDetachedZeroExit("sh", ["-lc", line]);
    return;
  }
  try {
    await execFileAsync("which", ["gnome-terminal"], { encoding: "utf8" });
    await runDetachedZeroExit("gnome-terminal", [
      "--working-directory",
      dir,
      "--",
      "claude",
    ]);
    return;
  } catch {
    /* try xterm */
  }
  try {
    await execFileAsync("which", ["xterm"], { encoding: "utf8" });
    const inner = `cd ${JSON.stringify(dir)} && exec claude`;
    await runDetachedZeroExit("xterm", ["-e", "bash", "-lc", inner]);
    return;
  } catch {
    /* */
  }
  throw new Error(
    "No se pudo abrir una terminal externa para Claude. En Linux instala gnome-terminal o xterm, o define TOPODUCTOR_TERMINAL (orden shell; usa {dir} para la carpeta, p. ej. gnome-terminal --working-directory={dir} -- claude)."
  );
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
    case "claude":
      await launchClaudeInExternalTerminal(targetPath);
      return;
    case "custom": {
      const line = expandPathTemplate(payload.customTpl ?? "", targetPath);
      await runShellLine(line);
      return;
    }
  }
}

/**
 * Clears the visible terminal and resets the cursor so Ink can redraw cleanly after
 * a child used stdio: "inherit" (shell, cursor CLI, etc.).
 */
export function clearTerminalAfterChildProcess(): void {
  if (!process.stdout.isTTY) {
    return;
  }
  // ED 2: erase entire display; CUP 1,1: cursor to home
  process.stdout.write("\u001b[2J\u001b[H");
}

export async function runExitActionInApp(payload: ExitPayload): Promise<void> {
  let clearAfter = true;
  try {
    const { path: targetPath, kind } = payload;

    switch (kind) {
      case "cd":
        await chdirAndExecShellInApp(targetPath);
        return;
      case "cursor":
        await openInCursor(targetPath);
        return;
      case "claude":
        await launchClaudeInExternalTerminal(targetPath);
        clearAfter = false;
        return;
      case "custom": {
        const line = expandPathTemplate(payload.customTpl ?? "", targetPath);
        await runShellLine(line);
        return;
      }
    }
  } finally {
    if (clearAfter) {
      clearTerminalAfterChildProcess();
    }
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

function win32ChdirHint(targetPath: string): void {
  process.stderr.write(
    "En Windows no se puede reemplazar el shell desde el programa. Usa --print-only y copia el comando, o:\n"
  );
  process.stdout.write(`cd /d ${JSON.stringify(targetPath)}\n`);
}

async function spawnInteractiveShellInWorktreeUnix(
  targetPath: string
): Promise<number | null> {
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
  return await new Promise((resolve, reject) => {
    const child = spawn(shellPath, args, {
      cwd: targetPath,
      stdio: "inherit",
      env: process.env,
    });
    child.on("error", reject);
    child.on("close", (code) => resolve(code));
  });
}

export async function chdirAndExecShell(targetPath: string): Promise<void> {
  if (process.platform === "win32") {
    win32ChdirHint(targetPath);
    return;
  }
  const code = await spawnInteractiveShellInWorktreeUnix(targetPath);
  process.exit(code ?? 0);
}

async function chdirAndExecShellInApp(targetPath: string): Promise<void> {
  if (process.platform === "win32") {
    win32ChdirHint(targetPath);
    return;
  }
  const code = await spawnInteractiveShellInWorktreeUnix(targetPath);
  if (code !== 0 && code !== null) {
    throw new Error(`shell exited with code ${code}`);
  }
}
