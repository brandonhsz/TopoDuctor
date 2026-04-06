/** Normaliza etiquetas de carpeta/rama para worktrees. */
export function sanitizeWorktreeLabel(s: string): string {
  s = s.trim();
  if (!s) {
    return "";
  }
  const out: string[] = [];
  for (const ch of s) {
    if (/[a-zA-Z0-9]/.test(ch)) {
      out.push(ch);
    } else if (ch === "-" || ch === "_" || ch === ".") {
      out.push(ch);
    } else if (ch === " ") {
      out.push("-");
    } else if (/\p{L}|\p{N}/u.test(ch)) {
      out.push(ch);
    } else {
      out.push("-");
    }
  }
  let r = out.join("").replace(/^[-._]+|[-._]+$/g, "");
  while (r.includes("--")) {
    r = r.replaceAll("--", "-");
  }
  return r.replace(/^-+|-+$/g, "");
}
