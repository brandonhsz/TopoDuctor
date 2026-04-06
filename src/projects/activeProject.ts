export function pickActiveProject(paths: string[], active: string): string {
  if (paths.length === 0) {
    return "";
  }
  for (const p of paths) {
    if (p === active) {
      return active;
    }
  }
  return paths[0] ?? "";
}

export function projectIndex(active: string, paths: string[]): number {
  const i = paths.indexOf(active);
  return i >= 0 ? i : 0;
}
