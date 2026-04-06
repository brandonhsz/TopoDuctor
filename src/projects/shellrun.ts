import { spawn } from "node:child_process";
import path from "node:path";
import { expandScriptPlaceholders } from "./projectConfig.js";

function shellPath(): string {
  return process.env.SHELL?.trim() || "/bin/sh";
}

export async function runScriptInDir(dir: string, script: string): Promise<void> {
  if (process.platform === "win32") {
    throw new Error("los scripts de proyecto no están soportados en Windows");
  }
  let line = script.trim();
  if (!line) {
    return;
  }
  line = expandScriptPlaceholders(line, dir);
  const absDir = path.resolve(dir);
  const sh = shellPath();
  await new Promise<void>((resolve, reject) => {
    const chunks: Buffer[] = [];
    const child = spawn(sh, ["-lc", line], {
      cwd: absDir,
      env: process.env,
      stdio: ["ignore", "pipe", "pipe"],
    });
    child.stdout?.on("data", (d: Buffer) => chunks.push(d));
    child.stderr?.on("data", (d: Buffer) => chunks.push(d));
    child.on("error", reject);
    child.on("close", (code) => {
      const s = Buffer.concat(chunks).toString("utf8").trim();
      if (code === 0) {
        resolve();
      } else {
        reject(s ? new Error(`exit ${code}: ${s}`) : new Error(`script failed (${code})`));
      }
    });
  });
}

export async function runScriptCapture(
  dir: string,
  script: string
): Promise<string> {
  if (process.platform === "win32") {
    throw new Error("los scripts de proyecto no están soportados en Windows");
  }
  let line = script.trim();
  if (!line) {
    return "";
  }
  const absDir = path.resolve(dir);
  line = expandScriptPlaceholders(line, absDir);
  const sh = shellPath();
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    const child = spawn(sh, ["-lc", line], {
      cwd: absDir,
      env: process.env,
      stdio: ["ignore", "pipe", "pipe"],
    });
    child.stdout?.on("data", (d: Buffer) => chunks.push(d));
    child.stderr?.on("data", (d: Buffer) => chunks.push(d));
    child.on("error", reject);
    child.on("close", (code) => {
      const s = Buffer.concat(chunks).toString("utf8");
      if (code === 0) {
        resolve(s);
      } else {
        reject(new Error(s.trim() ? s.trim() : `exit ${code}`));
      }
    });
  });
}
