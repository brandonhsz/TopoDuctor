import semver from "semver";

const defaultOwner = "brandonhsz";
const defaultRepo = "TopoDuctor";

export type ReleaseTag = { tag: string; url: string };

type GithubRelease = {
  tag_name: string;
  html_url: string;
};

export async function fetchLatestRelease(
  owner = defaultOwner,
  repo = defaultRepo,
  signal?: AbortSignal
): Promise<ReleaseTag> {
  const url = `https://api.github.com/repos/${owner}/${repo}/releases/latest`;
  const res = await fetch(url, {
    signal,
    headers: {
      Accept: "application/vnd.github+json",
      "User-Agent": "topoductor-update-check",
    },
  });
  if (!res.ok) {
    throw new Error(`GitHub API: ${res.status} ${res.statusText}`);
  }
  const body = (await res.json()) as GithubRelease;
  const tag = (body.tag_name ?? "").trim();
  if (!tag) {
    throw new Error("respuesta sin tag_name");
  }
  return { tag, url: body.html_url };
}

export function normalizeSemver(v: string): string {
  v = v.trim().replace(/^v/i, "");
  if (!v || v.toLowerCase() === "dev") {
    return "v0.0.0";
  }
  const i = v.indexOf("-");
  if (i >= 0) {
    v = v.slice(0, i);
  }
  const withV = v.startsWith("v") ? v : `v${v}`;
  if (!semver.valid(withV)) {
    return "v0.0.0";
  }
  return withV;
}

export function isNewerThan(current: string, latest: string): boolean {
  return semver.lt(normalizeSemver(current), normalizeSemver(latest));
}
