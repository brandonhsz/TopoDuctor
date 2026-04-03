import os from "node:os";
import path from "node:path";

/** Mirrors Go's os.UserConfigDir() for config file location. */
export function userConfigDir(): string {
  const home = os.homedir();
  switch (process.platform) {
    case "darwin":
      return path.join(home, "Library", "Application Support");
    case "win32":
      return process.env.APPDATA || path.join(home, "AppData", "Roaming");
    default:
      return process.env.XDG_CONFIG_HOME || path.join(home, ".config");
  }
}
