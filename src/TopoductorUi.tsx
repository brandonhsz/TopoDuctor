import React, { useEffect, useRef, useState } from "react";
import { Box, Text, useApp, useInput, useStdout } from "ink";
import path from "node:path";
import os from "node:os";
import {
  listWorktrees,
  listBranches,
  addUserWorktree,
  moveWorktree,
  removeWorktree,
  type Worktree,
} from "./git/operations.js";
import {
  defaultConfigPath,
  loadProjects,
  normalizePaths,
  saveProjects,
} from "./projects/store.js";
import {
  normalizePreferredBranchesMap,
  normalizePreferredBranchNames,
} from "./projects/branches.js";
import { shouldShowLobby } from "./projects/lobby.js";
import { isGitRepo } from "./projects/gitMeta.js";
import {
  pickActiveProject,
  projectIndex,
} from "./projects/activeProject.js";
import type { ProjectsFile } from "./projects/types.js";
import {
  readProjectConfig,
  saveProjectScripts,
} from "./projects/projectConfig.js";
import { runScriptInDir, runScriptCapture } from "./projects/shellrun.js";
import { fetchLatestRelease, isNewerThan } from "./update/github.js";
import { brewUpgradeFormula } from "./update/brew.js";
import {
  runExitActionInApp,
  type ExitPayload,
} from "./exit/runExit.js";
import { runClaudeHeadlessPrompt } from "./exit/claudeHeadless.js";
import { gridCols, withGridCursor } from "./ui/gridNav.js";
import {
  createBranchVisible,
  filteredCreateBranches,
  adjustBranchScroll,
} from "./ui/createBranchHelpers.js";
import { WorktreeGrid } from "./ui/WorktreeGrid.js";
import {
  marqueeTickMs,
  selectedNeedsMarquee,
} from "./ui/cardText.js";
import { clamp } from "./ui/clamp.js";
import { ModalOverlay } from "./ui/ModalOverlay.js";
import { modalMaxWidth } from "./ui/modalMaxWidth.js";

type View =
  | { kind: "bootstrap" }
  | { kind: "lobby" }
  | { kind: "projectPicker" }
  | { kind: "list" };

export type Dialog =
  | { kind: "none" }
  | { kind: "addProjectPath" }
  | {
      kind: "claudePrompt";
      workDir: string;
      promptBuf: string;
      loading: boolean;
      ran: boolean;
      out: string;
      err: string;
      scroll: number;
    }
  | {
      kind: "exit";
      path: string;
      exitCursor: 0 | 1 | 2 | 3;
      customBuf: string;
    }
  | {
      kind: "createPick";
      filter: string;
      branches: string[] | null;
      loadErr: string | null;
      loading: boolean;
      brCursor: number;
      brScroll: number;
    }
  | { kind: "createName"; baseRef: string; nameBuf: string }
  | { kind: "rename"; wtPath: string; buf: string }
  | { kind: "deleteConfirm"; wtPath: string }
  | { kind: "archiveRunConfirm"; wtPath: string; line: string }
  | {
      kind: "branchPrefs";
      repoPath: string;
      focus: 0 | 1 | 2;
      b0: string;
      b1: string;
      b2: string;
    }
  | {
      kind: "scriptEdit";
      focus: 0 | 1 | 2;
      setup: string;
      run: string;
      archive: string;
      loadErr: string | null;
    }
  | {
      kind: "scriptRun";
      title: string;
      workDir: string;
      cmd: string;
      loading: boolean;
      out: string;
      err: string;
      scroll: number;
    }
  | {
      kind: "settings";
      checking: boolean;
      applying: boolean;
      err: string;
      notice: string;
      latest: string;
      releaseURL: string;
      hasNewer: boolean;
    };

type AppState = {
  view: View;
  projectPickerReturn: "lobby" | "list";
  configPath: string;
  seedCwd: string;
  projectPaths: string[];
  activeProject: string;
  preferredBranches: Record<string, string[]>;
  projectCursor: number;
  worktrees: Worktree[];
  listCursor: number;
  listLoading: boolean;
  listError: string | null;
  bootstrapError: string | null;
  banner: string;
  addPathBuffer: string;
  listFetchKey: number;
  dialog: Dialog;
  busy: boolean;
};

const scriptRunVisible = 14;

function initialState(seedCwd: string): AppState {
  return {
    view: { kind: "bootstrap" },
    projectPickerReturn: "list",
    configPath: "",
    seedCwd,
    projectPaths: [],
    activeProject: "",
    preferredBranches: {},
    projectCursor: 0,
    worktrees: [],
    listCursor: 0,
    listLoading: false,
    listError: null,
    bootstrapError: null,
    banner: "",
    addPathBuffer: "",
    listFetchKey: 0,
    dialog: { kind: "none" },
    busy: false,
  };
}

function expandUserPath(raw: string): string {
  const t = raw.trim();
  if (!t.startsWith("~")) {
    return path.resolve(t);
  }
  const home = os.homedir();
  const rest = t.slice(1);
  if (rest.startsWith("/") || rest === "") {
    return path.join(home, rest.replace(/^\//, ""));
  }
  return path.join(home, rest);
}

function persistFile(s: AppState): ProjectsFile {
  const pref =
    Object.keys(s.preferredBranches).length > 0
      ? s.preferredBranches
      : undefined;
  return {
    paths: s.projectPaths,
    active: s.activeProject,
    preferred_branches: pref,
  };
}

function scriptOutLines(s: string): string[] {
  if (!s) {
    return [];
  }
  return s.replaceAll("\r\n", "\n").split("\n");
}

function scriptMaxScroll(lineCount: number): number {
  if (lineCount <= scriptRunVisible) {
    return 0;
  }
  return lineCount - scriptRunVisible;
}

function prefKey(repo: string): string {
  return path.normalize(repo);
}

type Props = {
  cwd: string;
  version: string;
  exitOutcomeRef: React.MutableRefObject<ExitPayload | null | undefined>;
};

export function TopoductorUi({
  cwd,
  version,
  exitOutcomeRef,
}: Props) {
  const { exit } = useApp();
  const { stdout } = useStdout();
  const termW = stdout?.columns ?? 80;
  const termH = stdout?.rows ?? 24;
  const [state, setState] = useState(() => initialState(cwd));
  const [marqueeTick, setMarqueeTick] = useState(0);
  const stateRef = useRef(state);
  stateRef.current = state;
  const claudeAbortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    setMarqueeTick(0);
  }, [state.listCursor, state.worktrees]);

  useEffect(() => {
    if (state.view.kind !== "list") {
      return;
    }
    if (state.dialog.kind !== "none") {
      return;
    }
    if (state.listLoading || state.listError || state.busy) {
      return;
    }
    if (!selectedNeedsMarquee(state.worktrees, state.listCursor)) {
      return;
    }
    const id = setInterval(
      () => setMarqueeTick((t) => t + 1),
      marqueeTickMs
    );
    return () => clearInterval(id);
  }, [
    state.view.kind,
    state.dialog.kind,
    state.listLoading,
    state.listError,
    state.busy,
    state.listCursor,
    state.worktrees,
  ]);

  const runExitChoice = (payload: ExitPayload) => {
    setState((x) => ({
      ...x,
      dialog: { kind: "none" },
      busy: true,
      banner: "",
    }));
    void (async () => {
      try {
        await runExitActionInApp(payload);
      } catch (e) {
        setState((x) => ({
          ...x,
          banner:
            e instanceof Error ? e.message : String(e),
        }));
      } finally {
        setState((x) => ({ ...x, busy: false }));
      }
    })();
  };

  const quitApp = () => {
    exitOutcomeRef.current = null;
    exit();
  };

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const configPath = await defaultConfigPath();
        const file = await loadProjects(configPath);
        const paths = normalizePaths(file.paths);
        const preferred = normalizePreferredBranchesMap(file.preferred_branches);
        const active = pickActiveProject(paths, file.active);
        const lobby = await shouldShowLobby(cwd, paths);
        if (cancelled) {
          return;
        }
        if (lobby) {
          setState((s) => ({
            ...s,
            configPath,
            projectPaths: paths,
            preferredBranches: preferred,
            activeProject: "",
            view: { kind: "lobby" },
            projectPickerReturn: "lobby",
            projectCursor: 0,
          }));
        } else {
          setState((s) => ({
            ...s,
            configPath,
            projectPaths: paths,
            preferredBranches: preferred,
            activeProject: active,
            view: { kind: "list" },
            projectPickerReturn: "list",
            projectCursor: projectIndex(active, paths),
            listLoading: true,
            listError: null,
            listCursor: 0,
            listFetchKey: 1,
          }));
        }
      } catch (e) {
        if (!cancelled) {
          setState((s) => ({
            ...s,
            bootstrapError:
              e instanceof Error ? e.message : String(e),
          }));
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [cwd]);

  useEffect(() => {
    if (state.view.kind !== "list" || !state.activeProject) {
      return;
    }
    let cancelled = false;
    setState((s) => ({
      ...s,
      listLoading: true,
      listError: null,
    }));
    (async () => {
      try {
        const w = await listWorktrees(state.activeProject);
        if (cancelled) {
          return;
        }
        setState((s) => ({
          ...s,
          worktrees: w,
          listCursor: clamp(
            s.listCursor,
            0,
            Math.max(0, w.length - 1)
          ),
          listLoading: false,
          listError: null,
        }));
      } catch (e) {
        if (!cancelled) {
          setState((s) => ({
            ...s,
            listLoading: false,
            listError:
              e instanceof Error ? e.message : String(e),
          }));
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [state.view.kind, state.activeProject, state.listFetchKey]);

  useEffect(() => {
    if (state.dialog.kind !== "settings" || !state.dialog.checking) {
      return;
    }
    const ac = new AbortController();
    (async () => {
      try {
        const rel = await fetchLatestRelease(
          undefined,
          undefined,
          ac.signal
        );
        const has = isNewerThan(version, rel.tag);
        setState((s) => {
          if (s.dialog.kind !== "settings" || !s.dialog.checking) {
            return s;
          }
          let notice: string;
          if (has) {
            notice =
              process.platform === "darwin"
                ? "Hay una versión más reciente. Pulsa i para ejecutar brew update y brew upgrade topoductor."
                : "Hay una versión más reciente. Descarga el binario desde GitHub (enlace abajo).";
          } else {
            const mv = version.trim() || "dev";
            notice = `Estás al día. Local: ${mv} · Release: ${rel.tag}`;
          }
          return {
            ...s,
            dialog: {
              kind: "settings",
              checking: false,
              applying: s.dialog.applying,
              err: "",
              notice,
              latest: rel.tag,
              releaseURL: rel.url,
              hasNewer: has,
            },
          };
        });
      } catch (e) {
        if (ac.signal.aborted) {
          return;
        }
        setState((s) => {
          if (s.dialog.kind !== "settings") {
            return s;
          }
          return {
            ...s,
            dialog: {
              kind: "settings",
              checking: false,
              applying: false,
              err: e instanceof Error ? e.message : String(e),
              notice: "",
              latest: "",
              releaseURL: "",
              hasNewer: false,
            },
          };
        });
      }
    })();
    return () => ac.abort();
  }, [
    state.dialog.kind,
    state.dialog.kind === "settings" ? state.dialog.checking : false,
    version,
  ]);

  const openSettings = () => {
    setState((s) => ({
      ...s,
      dialog: {
        kind: "settings",
        checking: true,
        applying: false,
        err: "",
        notice: "",
        latest: "",
        releaseURL: "",
        hasNewer: false,
      },
    }));
  };

  const startScriptRun = (
    title: string,
    workDir: string,
    scriptLine: string
  ) => {
    const abs = path.resolve(workDir);
    setState((s) => ({
      ...s,
      dialog: {
        kind: "scriptRun",
        title,
        workDir: abs,
        cmd: scriptLine,
        loading: true,
        out: "",
        err: "",
        scroll: 0,
      },
    }));
    void (async () => {
      try {
        const out = await runScriptCapture(abs, scriptLine);
        setState((s) => {
          if (s.dialog.kind !== "scriptRun" || s.dialog.workDir !== abs) {
            return s;
          }
          return {
            ...s,
            dialog: {
              ...s.dialog,
              loading: false,
              out,
              err: "",
              scroll: 0,
            },
          };
        });
      } catch (e) {
        setState((s) => {
          if (s.dialog.kind !== "scriptRun") {
            return s;
          }
          return {
            ...s,
            dialog: {
              ...s.dialog,
              loading: false,
              out: "",
              err: e instanceof Error ? e.message : String(e),
              scroll: 0,
            },
          };
        });
      }
    })();
  };

  useInput((input, key) => {
    const s = stateRef.current;
    const d = s.dialog;

    if (s.bootstrapError && (input === "q" || key.escape)) {
      quitApp();
      return;
    }

    if (d.kind !== "none") {
      if (d.kind === "claudePrompt") {
        const pieces: string[] = [];
        if (d.err) {
          pieces.push(d.err);
        }
        if (d.out) {
          pieces.push(d.out);
        }
        const scrollText = pieces.length > 0 ? pieces.join("\n---\n") : "";
        const lines = scriptOutLines(scrollText || "(sin salida)");
        const maxScr = scriptMaxScroll(lines.length);
        const editing = !d.loading && !d.ran;
        const showing = d.ran && !d.loading;

        if (d.loading && key.escape) {
          claudeAbortRef.current?.abort();
          claudeAbortRef.current = null;
          setState((x) => ({
            ...x,
            dialog:
              x.dialog.kind === "claudePrompt"
                ? {
                    ...x.dialog,
                    loading: false,
                  }
                : x.dialog,
          }));
          return;
        }
        if (showing && (key.escape || key.return)) {
          setState((x) => ({ ...x, dialog: { kind: "none" } }));
          return;
        }
        if (editing && key.escape) {
          setState((x) => ({ ...x, dialog: { kind: "none" } }));
          return;
        }
        if (showing) {
          if (input === "k" || key.upArrow) {
            if (d.scroll > 0) {
              setState((x) => ({
                ...x,
                dialog:
                  x.dialog.kind === "claudePrompt"
                    ? { ...x.dialog, scroll: x.dialog.scroll - 1 }
                    : x.dialog,
              }));
            }
            return;
          }
          if (input === "j" || key.downArrow) {
            if (scrollText && d.scroll < maxScr) {
              setState((x) => ({
                ...x,
                dialog:
                  x.dialog.kind === "claudePrompt"
                    ? { ...x.dialog, scroll: x.dialog.scroll + 1 }
                    : x.dialog,
              }));
            }
            return;
          }
          return;
        }
        if (editing) {
          if (key.return) {
            const buf = d.promptBuf.trim();
            if (!buf) {
              setState((x) => ({
                ...x,
                banner: "Escribe un prompt para Claude.",
              }));
              return;
            }
            const workDir = d.workDir;
            const ac = new AbortController();
            claudeAbortRef.current = ac;
            setState((x) => ({
              ...x,
              dialog:
                x.dialog.kind === "claudePrompt"
                  ? {
                      ...x.dialog,
                      loading: true,
                      out: "",
                      err: "",
                      scroll: 0,
                    }
                  : x.dialog,
              banner: "",
            }));
            void (async () => {
              try {
                const r = await runClaudeHeadlessPrompt(
                  workDir,
                  buf,
                  ac.signal
                );
                const errText =
                  r.exitCode !== 0 && r.exitCode !== null
                    ? [r.stderr, `(código de salida ${r.exitCode})`]
                        .filter(Boolean)
                        .join("\n")
                    : r.stderr;
                setState((x) => {
                  if (x.dialog.kind !== "claudePrompt" || !x.dialog.loading) {
                    return x;
                  }
                  return {
                    ...x,
                    dialog: {
                      ...x.dialog,
                      loading: false,
                      ran: true,
                      out: r.stdout,
                      err: errText,
                      scroll: 0,
                    },
                  };
                });
              } catch (e) {
                const msg = e instanceof Error ? e.message : String(e);
                setState((x) => {
                  if (x.dialog.kind !== "claudePrompt") {
                    return x;
                  }
                  if (msg === "Cancelado") {
                    return {
                      ...x,
                      dialog: {
                        ...x.dialog,
                        loading: false,
                        ran: false,
                        out: "",
                        err: "",
                      },
                    };
                  }
                  return {
                    ...x,
                    dialog: {
                      ...x.dialog,
                      loading: false,
                      ran: false,
                      out: "",
                      err: "",
                    },
                    banner: msg,
                  };
                });
              } finally {
                claudeAbortRef.current = null;
              }
            })();
            return;
          }
          if (key.backspace || key.delete) {
            setState((x) => ({
              ...x,
              dialog:
                x.dialog.kind === "claudePrompt"
                  ? {
                      ...x.dialog,
                      promptBuf: x.dialog.promptBuf.slice(0, -1),
                    }
                  : x.dialog,
            }));
            return;
          }
          if (input && !key.ctrl && input.length >= 1) {
            setState((x) => ({
              ...x,
              dialog:
                x.dialog.kind === "claudePrompt"
                  ? {
                      ...x.dialog,
                      promptBuf: x.dialog.promptBuf + input,
                    }
                  : x.dialog,
            }));
          }
        }
        return;
      }
      if (d.kind === "addProjectPath") {
        if (key.escape) {
          setState((x) => ({
            ...x,
            dialog: { kind: "none" },
            addPathBuffer: "",
            banner: "",
          }));
          return;
        }
        if (key.return) {
          const raw = s.addPathBuffer.trim();
          if (!raw || !s.configPath) {
            return;
          }
          void (async () => {
            try {
              const abs = path.normalize(expandUserPath(raw));
              if (!(await isGitRepo(abs))) {
                setState((x) => ({
                  ...x,
                  banner: "No es un repositorio git válido.",
                }));
                return;
              }
              const cur = stateRef.current;
              if (cur.projectPaths.includes(abs)) {
                setState((x) => ({
                  ...x,
                  banner: "Ese proyecto ya está en la lista.",
                }));
                return;
              }
              const paths = [...cur.projectPaths, abs];
              await saveProjects(cur.configPath, {
                paths,
                active: abs,
                preferred_branches:
                  Object.keys(cur.preferredBranches).length > 0
                    ? cur.preferredBranches
                    : undefined,
              });
              setState((x) => ({
                ...x,
                projectPaths: paths,
                activeProject: abs,
                projectCursor: paths.length - 1,
                view: { kind: "list" },
                projectPickerReturn: "list",
                addPathBuffer: "",
                dialog: { kind: "none" },
                banner: "",
                listLoading: true,
                worktrees: [],
                listCursor: 0,
                listFetchKey: x.listFetchKey + 1,
              }));
            } catch (e) {
              setState((x) => ({
                ...x,
                banner:
                  e instanceof Error ? e.message : String(e),
              }));
            }
          })();
          return;
        }
        if (key.backspace || key.delete) {
          setState((x) => ({
            ...x,
            addPathBuffer: x.addPathBuffer.slice(0, -1),
          }));
          return;
        }
        if (input && !key.ctrl && input.length >= 1) {
          setState((x) => ({
            ...x,
            addPathBuffer: x.addPathBuffer + input,
          }));
        }
        return;
      }
      if (d.kind === "settings") {
        if (d.checking || d.applying) {
          if (key.escape || (key.ctrl && input === "c")) {
            setState((x) => ({ ...x, dialog: { kind: "none" } }));
          }
          return;
        }
        if (key.escape || (key.ctrl && input === "c")) {
          setState((x) => ({ ...x, dialog: { kind: "none" } }));
          return;
        }
        if (input === "u") {
          setState((x) => ({
            ...x,
            dialog: {
              kind: "settings",
              checking: true,
              applying: false,
              err: "",
              notice: "",
              latest: "",
              releaseURL: "",
              hasNewer: false,
            },
          }));
          return;
        }
        if (input === "i" && d.hasNewer) {
          if (process.platform !== "darwin") {
            setState((x) => ({
              ...x,
              dialog: {
                ...d,
                err:
                  "La instalación con Homebrew solo está disponible en macOS.",
              },
            }));
            return;
          }
          setState((x) => ({
            ...x,
            dialog: { ...d, applying: true, err: "", notice: "" },
          }));
          void (async () => {
            try {
              const out = await brewUpgradeFormula();
              void out;
              setState((x) => ({
                ...x,
                dialog: {
                  kind: "settings",
                  checking: false,
                  applying: false,
                  err: "",
                  notice:
                    "Homebrew terminó. Cierra esta app y vuelve a abrirla para usar la nueva versión.",
                  latest: d.latest,
                  releaseURL: d.releaseURL,
                  hasNewer: false,
                },
              }));
            } catch (e) {
              setState((x) => ({
                ...x,
                dialog: {
                  ...d,
                  applying: false,
                  err:
                    e instanceof Error ? e.message : String(e),
                },
              }));
            }
          })();
          return;
        }
        return;
      }

      if (d.kind === "scriptRun") {
        const lines = scriptOutLines(d.out);
        const maxScr = scriptMaxScroll(lines.length);
        if (!d.loading && (key.escape || key.return)) {
          setState((x) => ({ ...x, dialog: { kind: "none" } }));
          return;
        }
        if (input === "k" || key.upArrow) {
          if (!d.loading && d.scroll > 0) {
            setState((x) => ({
              ...x,
              dialog: { ...d, scroll: d.scroll - 1 },
            }));
          }
          return;
        }
        if (input === "j" || key.downArrow) {
          if (!d.loading && d.out && d.scroll < maxScr) {
            setState((x) => ({
              ...x,
              dialog: { ...d, scroll: d.scroll + 1 },
            }));
          }
          return;
        }
        return;
      }

      if (d.kind === "scriptEdit") {
        if (key.escape) {
          setState((x) => ({ ...x, dialog: { kind: "none" }, banner: "" }));
          return;
        }
        if (key.return) {
          void (async () => {
            try {
              await saveProjectScripts(stateRef.current.activeProject, {
                setup: d.setup,
                run: d.run,
                archive: d.archive,
              });
              setState((x) => ({
                ...x,
                dialog: { kind: "none" },
                banner: "Guardado en .topoductor/project.json",
              }));
            } catch (e) {
              setState((x) => ({
                ...x,
                banner:
                  e instanceof Error ? e.message : String(e),
              }));
            }
          })();
          return;
        }
        if (key.tab) {
          const nf = ((d.focus + 1) % 3) as 0 | 1 | 2;
          setState((x) => ({ ...x, dialog: { ...d, focus: nf } }));
          return;
        }
        if (key.backspace || key.delete) {
          const f = d.focus;
          const field = f === 0 ? "setup" : f === 1 ? "run" : "archive";
          const cur = d[field];
          setState((x) => ({
            ...x,
            dialog: { ...d, [field]: cur.slice(0, -1) },
          }));
          return;
        }
        if (input && !key.ctrl && input.length >= 1) {
          const f = d.focus;
          const field = f === 0 ? "setup" : f === 1 ? "run" : "archive";
          const cur = d[field];
          setState((x) => ({
            ...x,
            dialog: { ...d, [field]: cur + input },
          }));
        }
        return;
      }

      if (d.kind === "branchPrefs") {
        if (key.escape) {
          setState((x) => ({
            ...x,
            dialog: { kind: "none" },
            banner: "",
          }));
          return;
        }
        if (key.return) {
          const raw = [d.b0, d.b1, d.b2];
          const names = normalizePreferredBranchNames(raw);
          const keyCl = prefKey(d.repoPath);
          const nextPref = { ...s.preferredBranches };
          if (names.length === 0) {
            delete nextPref[keyCl];
          } else {
            nextPref[keyCl] = names;
          }
          void (async () => {
            try {
              const cur = stateRef.current;
              if (cur.configPath) {
                const merged = { ...cur, preferredBranches: nextPref };
                await saveProjects(cur.configPath, persistFile(merged));
              }
              setState((x) => ({
                ...x,
                preferredBranches: nextPref,
                dialog: { kind: "none" },
                banner: "",
              }));
            } catch (e) {
              setState((x) => ({
                ...x,
                banner:
                  e instanceof Error ? e.message : String(e),
              }));
            }
          })();
          return;
        }
        if (key.tab) {
          const nf = ((d.focus + 1) % 3) as 0 | 1 | 2;
          setState((x) => ({ ...x, dialog: { ...d, focus: nf } }));
          return;
        }
        const f = `b${d.focus}` as "b0" | "b1" | "b2";
        if (key.backspace || key.delete) {
          setState((x) => ({
            ...x,
            dialog: { ...d, [f]: d[f].slice(0, -1) },
          }));
          return;
        }
        if (input && !key.ctrl) {
          setState((x) => ({
            ...x,
            dialog: { ...d, [f]: d[f] + input },
          }));
        }
        return;
      }

      if (d.kind === "archiveRunConfirm") {
        if (input === "n" || key.escape) {
          setState((x) => ({ ...x, dialog: { kind: "none" } }));
          return;
        }
        if (input === "y" || key.return) {
          const { wtPath, line } = d;
          startScriptRun("Archive", wtPath, line);
          return;
        }
        return;
      }

      if (d.kind === "deleteConfirm") {
        if (input === "n" || key.escape) {
          setState((x) => ({ ...x, dialog: { kind: "none" } }));
          return;
        }
        if (input === "y" || key.return) {
          const wtPath = d.wtPath;
          const ap = s.activeProject;
          setState((x) => ({ ...x, dialog: { kind: "none" }, busy: true }));
          void (async () => {
            try {
              let archiveLine = "";
              try {
                const sc = await readProjectConfig(ap);
                archiveLine = (sc.archive ?? "").trim();
              } catch {
                /* ignore */
              }
              if (archiveLine) {
                await runScriptInDir(wtPath, archiveLine);
              }
              await removeWorktree(ap, wtPath);
              const w = await listWorktrees(ap);
              setState((x) => ({
                ...x,
                worktrees: w,
                listCursor: clamp(x.listCursor, 0, Math.max(0, w.length - 1)),
                busy: false,
              }));
            } catch (e) {
              setState((x) => ({
                ...x,
                busy: false,
                banner:
                  e instanceof Error ? e.message : String(e),
              }));
            }
          })();
          return;
        }
        return;
      }

      if (d.kind === "rename") {
        if (key.escape) {
          setState((x) => ({ ...x, dialog: { kind: "none" } }));
          return;
        }
        if (key.return) {
          const name = d.buf.trim();
          if (!name) {
            return;
          }
          const oldP = d.wtPath;
          const ap = s.activeProject;
          setState((x) => ({ ...x, dialog: { kind: "none" }, busy: true }));
          void (async () => {
            try {
              await moveWorktree(ap, oldP, name);
              const w = await listWorktrees(ap);
              setState((x) => ({
                ...x,
                worktrees: w,
                listCursor: clamp(x.listCursor, 0, Math.max(0, w.length - 1)),
                busy: false,
              }));
            } catch (e) {
              setState((x) => ({
                ...x,
                busy: false,
                banner:
                  e instanceof Error ? e.message : String(e),
              }));
            }
          })();
          return;
        }
        if (key.backspace || key.delete) {
          setState((x) => ({
            ...x,
            dialog: { ...d, buf: d.buf.slice(0, -1) },
          }));
          return;
        }
        if (input && !key.ctrl) {
          setState((x) => ({ ...x, dialog: { ...d, buf: d.buf + input } }));
        }
        return;
      }

      if (d.kind === "createName") {
        if (key.escape) {
          setState((x) => ({
            ...x,
            dialog: {
              kind: "createPick",
              filter: "",
              branches: null,
              loadErr: null,
              loading: true,
              brCursor: 0,
              brScroll: 0,
            },
          }));
          void (async () => {
            try {
              const ap = stateRef.current.activeProject;
              const br = await listBranches(ap);
              setState((x) => {
                if (x.dialog.kind !== "createPick" || x.dialog.loading !== true) {
                  return x;
                }
                return {
                  ...x,
                  dialog: {
                    ...x.dialog,
                    branches: br,
                    loading: false,
                    loadErr: null,
                  },
                };
              });
            } catch (e) {
              setState((x) => ({
                ...x,
                dialog: {
                  kind: "createPick",
                  filter: "",
                  branches: null,
                  loadErr:
                    e instanceof Error ? e.message : String(e),
                  loading: false,
                  brCursor: 0,
                  brScroll: 0,
                },
              }));
            }
          })();
          return;
        }
        if (key.return) {
          const name = d.nameBuf.trim();
          if (!name) {
            return;
          }
          const base = d.baseRef;
          const ap = s.activeProject;
          setState((x) => ({ ...x, dialog: { kind: "none" }, busy: true }));
          void (async () => {
            try {
              await addUserWorktree(ap, base, name);
              const w = await listWorktrees(ap);
              setState((x) => ({
                ...x,
                worktrees: w,
                listCursor: clamp(x.listCursor, 0, Math.max(0, w.length - 1)),
                busy: false,
              }));
            } catch (e) {
              setState((x) => ({
                ...x,
                busy: false,
                banner:
                  e instanceof Error ? e.message : String(e),
              }));
            }
          })();
          return;
        }
        if (key.backspace || key.delete) {
          setState((x) => ({
            ...x,
            dialog: { ...d, nameBuf: d.nameBuf.slice(0, -1) },
          }));
          return;
        }
        if (input && !key.ctrl) {
          setState((x) => ({
            ...x,
            dialog: { ...d, nameBuf: d.nameBuf + input },
          }));
        }
        return;
      }

      if (d.kind === "createPick") {
        const prefs =
          s.preferredBranches[prefKey(s.activeProject)] ?? [];
        const all = d.branches ?? [];
        const f = filteredCreateBranches(all, d.filter, prefs);
        if (key.escape || input === "q") {
          if (input === "q") {
            quitApp();
          } else {
            setState((x) => ({ ...x, dialog: { kind: "none" } }));
          }
          return;
        }
        if (d.loading || d.loadErr) {
          return;
        }
        if (key.return) {
          if (f.length === 0) {
            setState((x) => ({
              ...x,
              banner: "No hay ramas que coincidan.",
            }));
            return;
          }
          const c = clamp(d.brCursor, 0, f.length - 1);
          const baseRef = f[c];
          setState((x) => ({
            ...x,
            dialog: {
              kind: "createName",
              baseRef,
              nameBuf: "",
            },
            banner: "",
          }));
          return;
        }
        if (input === "k" || key.upArrow) {
          if (f.length === 0) {
            return;
          }
          const c = clamp(d.brCursor - 1, 0, f.length - 1);
          const sc = adjustBranchScroll(
            c,
            d.brScroll,
            createBranchVisible,
            f.length
          );
          setState((x) => ({
            ...x,
            dialog: { ...d, brCursor: c, brScroll: sc },
          }));
          return;
        }
        if (input === "j" || key.downArrow) {
          if (f.length === 0) {
            return;
          }
          const c = clamp(d.brCursor + 1, 0, f.length - 1);
          const sc = adjustBranchScroll(
            c,
            d.brScroll,
            createBranchVisible,
            f.length
          );
          setState((x) => ({
            ...x,
            dialog: { ...d, brCursor: c, brScroll: sc },
          }));
          return;
        }
        if (key.backspace || key.delete) {
          const nf = d.filter.slice(0, -1);
          setState((x) => ({
            ...x,
            dialog: { ...d, filter: nf, brCursor: 0, brScroll: 0 },
          }));
          return;
        }
        if (input && !key.ctrl && input.length >= 1) {
          setState((x) => ({
            ...x,
            dialog: {
              ...d,
              filter: d.filter + input,
              brCursor: 0,
              brScroll: 0,
            },
          }));
          return;
        }
        return;
      }

      if (d.kind === "exit") {
        if (key.escape) {
          setState((x) => ({ ...x, dialog: { kind: "none" } }));
          return;
        }
        if (input === "q") {
          runExitChoice({ kind: "cd", path: d.path });
          return;
        }
        if (input === "k" || key.upArrow) {
          if (d.exitCursor > 0) {
            const nc = (d.exitCursor - 1) as 0 | 1 | 2 | 3;
            setState((x) => ({
              ...x,
              dialog: { ...d, exitCursor: nc },
            }));
          }
          return;
        }
        if (input === "j" || key.downArrow) {
          if (d.exitCursor < 3) {
            const nc = (d.exitCursor + 1) as 0 | 1 | 2 | 3;
            setState((x) => ({
              ...x,
              dialog: { ...d, exitCursor: nc },
            }));
          }
          return;
        }
        if (key.return) {
          if (d.exitCursor === 3) {
            const v = d.customBuf.trim();
            if (!v) {
              setState((x) => ({
                ...x,
                banner: "Escribe un comando con {path} o elige otra opción.",
              }));
              return;
            }
            runExitChoice({
              kind: "custom",
              path: d.path,
              customTpl: v,
            });
            return;
          }
          if (d.exitCursor === 0) {
            runExitChoice({ kind: "cd", path: d.path });
          } else if (d.exitCursor === 1) {
            runExitChoice({ kind: "cursor", path: d.path });
          } else if (d.exitCursor === 2) {
            setState((x) => ({
              ...x,
              dialog: {
                kind: "claudePrompt",
                workDir: d.path,
                promptBuf: "",
                loading: false,
                ran: false,
                out: "",
                err: "",
                scroll: 0,
              },
              banner: "",
            }));
          }
          return;
        }
        if (d.exitCursor === 3) {
          if (key.backspace || key.delete) {
            setState((x) => ({
              ...x,
              dialog: { ...d, customBuf: d.customBuf.slice(0, -1) },
            }));
            return;
          }
          if (input && !key.ctrl) {
            setState((x) => ({
              ...x,
              dialog: { ...d, customBuf: d.customBuf + input },
            }));
          }
        }
        return;
      }

      return;
    }

    if (s.view.kind === "bootstrap") {
      return;
    }

    if (s.view.kind === "lobby") {
      if (key.ctrl && input === "c") {
        openSettings();
        return;
      }
      if (input === "q") {
        quitApp();
        return;
      }
      if (input === "p" || key.return) {
        setState((x) => ({
          ...x,
          view: { kind: "projectPicker" },
          projectPickerReturn: "lobby",
          projectCursor: projectIndex(x.activeProject, x.projectPaths),
          banner: "",
        }));
      }
      return;
    }

    if (s.view.kind === "projectPicker") {
      if (key.ctrl && input === "c") {
        openSettings();
        return;
      }
      if (input === "b" || input === "B") {
        if (s.projectPaths.length === 0) {
          return;
        }
        const repo = s.projectPaths[s.projectCursor];
        const prefs = s.preferredBranches[prefKey(repo)] ?? [];
        setState((x) => ({
          ...x,
          dialog: {
            kind: "branchPrefs",
            repoPath: repo,
            focus: 0,
            b0: prefs[0] ?? "",
            b1: prefs[1] ?? "",
            b2: prefs[2] ?? "",
          },
          banner: "",
        }));
        return;
      }
      if (input === "q" || key.escape) {
        const ret = s.projectPickerReturn;
        setState((x) => ({
          ...x,
          view: ret === "lobby" ? { kind: "lobby" } : { kind: "list" },
          banner: "",
        }));
        return;
      }
      if (input === "a") {
        setState((x) => ({
          ...x,
          dialog: { kind: "addProjectPath" },
          addPathBuffer: "",
          banner: "",
        }));
        return;
      }
      if (input === "k" || key.upArrow) {
        setState((x) => ({
          ...x,
          projectCursor: clamp(
            x.projectCursor - 1,
            0,
            x.projectPaths.length - 1
          ),
        }));
        return;
      }
      if (input === "j" || key.downArrow) {
        setState((x) => ({
          ...x,
          projectCursor: clamp(
            x.projectCursor + 1,
            0,
            x.projectPaths.length - 1
          ),
        }));
        return;
      }
      if (key.return && s.projectPaths.length > 0) {
        const picked = s.projectPaths[s.projectCursor];
        if (!picked || !s.configPath) {
          return;
        }
        void (async () => {
          try {
            const file = persistFile({
              ...s,
              activeProject: picked,
            });
            await saveProjects(s.configPath, file);
            setState((x) => ({
              ...x,
              activeProject: picked,
              view: { kind: "list" },
              projectPickerReturn: "list",
              listLoading: true,
              worktrees: [],
              listCursor: 0,
              listFetchKey: x.listFetchKey + 1,
              banner: "",
            }));
          } catch (e) {
            setState((x) => ({
              ...x,
              banner:
                e instanceof Error ? e.message : String(e),
            }));
          }
        })();
        return;
      }
      if (input === "d" && s.projectPaths.length > 0) {
        const removed = s.projectPaths[s.projectCursor];
        const nextPaths = s.projectPaths.filter(
          (_, i) => i !== s.projectCursor
        );
        const nextPref = { ...s.preferredBranches };
        delete nextPref[path.normalize(removed)];
        let nextCursor = s.projectCursor;
        if (nextPaths.length === 0) {
          void (async () => {
            try {
              if (s.configPath) {
                await saveProjects(s.configPath, {
                  paths: [],
                  active: "",
                  preferred_branches: undefined,
                });
              }
              setState((x) => ({
                ...x,
                projectPaths: [],
                activeProject: "",
                preferredBranches: {},
                projectCursor: 0,
                view: { kind: "lobby" },
                projectPickerReturn: "lobby",
                worktrees: [],
                listCursor: 0,
                banner: "",
              }));
            } catch (e) {
              setState((x) => ({
                ...x,
                banner:
                  e instanceof Error ? e.message : String(e),
              }));
            }
          })();
          return;
        }
        if (nextCursor >= nextPaths.length) {
          nextCursor = nextPaths.length - 1;
        }
        const removedWasActive = s.activeProject === removed;
        let nextActive = s.activeProject;
        if (removedWasActive) {
          nextActive = nextPaths[nextCursor] ?? "";
        }
        void (async () => {
          try {
            if (s.configPath) {
              await saveProjects(s.configPath, {
                paths: nextPaths,
                active: nextActive,
                preferred_branches:
                  Object.keys(nextPref).length > 0 ? nextPref : undefined,
              });
            }
            setState((x) => ({
              ...x,
              projectPaths: nextPaths,
              preferredBranches: nextPref,
              activeProject: nextActive,
              projectCursor: nextCursor,
              listLoading: removedWasActive,
              worktrees: removedWasActive ? [] : x.worktrees,
              banner: "",
            }));
          } catch (e) {
            setState((x) => ({
              ...x,
              banner:
                e instanceof Error ? e.message : String(e),
            }));
          }
        })();
      }
      return;
    }

    if (s.view.kind === "list") {
      if (key.ctrl && input === "c") {
        openSettings();
        return;
      }
      if (input === "q") {
        quitApp();
        return;
      }
      if (key.ctrl && input === "l") {
        setState((x) => ({
          ...x,
          view: { kind: "lobby" },
          projectPickerReturn: "lobby",
          banner: "",
        }));
        return;
      }
      if (input === "p") {
        setState((x) => ({
          ...x,
          view: { kind: "projectPicker" },
          projectPickerReturn: "list",
          projectCursor: projectIndex(x.activeProject, x.projectPaths),
          banner: "",
        }));
        return;
      }
      if (input === "b" || input === "B") {
        if (!s.activeProject) {
          setState((x) => ({
            ...x,
            banner: "Añade o activa un proyecto (p).",
          }));
          return;
        }
        const repo = s.activeProject;
        const prefs = s.preferredBranches[prefKey(repo)] ?? [];
        setState((x) => ({
          ...x,
          dialog: {
            kind: "branchPrefs",
            repoPath: repo,
            focus: 0,
            b0: prefs[0] ?? "",
            b1: prefs[1] ?? "",
            b2: prefs[2] ?? "",
          },
          banner: "",
        }));
        return;
      }
      if (input === "e") {
        if (!s.activeProject) {
          setState((x) => ({
            ...x,
            banner: "Añade o activa un proyecto (p).",
          }));
          return;
        }
        void (async () => {
          try {
            const sc = await readProjectConfig(stateRef.current.activeProject);
            setState((x) => ({
              ...x,
              dialog: {
                kind: "scriptEdit",
                focus: 0,
                setup: sc.setup ?? "",
                run: sc.run ?? "",
                archive: sc.archive ?? "",
                loadErr: null,
              },
              banner: "",
            }));
          } catch (e) {
            setState((x) => ({
              ...x,
              dialog: {
                kind: "scriptEdit",
                focus: 0,
                setup: "",
                run: "",
                archive: "",
                loadErr:
                  e instanceof Error ? e.message : String(e),
              },
            }));
          }
        })();
        return;
      }
      if (input === "n") {
        if (!s.activeProject) {
          setState((x) => ({
            ...x,
            banner: "Añade un proyecto (p → a) antes de crear worktrees.",
          }));
          return;
        }
        setState((x) => ({
          ...x,
          dialog: {
            kind: "createPick",
            filter: "",
            branches: null,
            loadErr: null,
            loading: true,
            brCursor: 0,
            brScroll: 0,
          },
          banner: "",
        }));
        void (async () => {
          try {
            const br = await listBranches(stateRef.current.activeProject);
            setState((x) => {
              if (x.dialog.kind !== "createPick") {
                return x;
              }
              return {
                ...x,
                dialog: {
                  ...x.dialog,
                  branches: br,
                  loading: false,
                  loadErr: null,
                },
              };
            });
          } catch (e) {
            setState((x) => ({
              ...x,
              dialog: {
                kind: "createPick",
                filter: "",
                branches: null,
                loadErr:
                  e instanceof Error ? e.message : String(e),
                loading: false,
                brCursor: 0,
                brScroll: 0,
              },
            }));
          }
        })();
        return;
      }
      if (input === "r") {
        if (s.worktrees.length === 0) {
          return;
        }
        const wt = s.worktrees[s.listCursor];
        if (!wt) {
          return;
        }
        setState((x) => ({
          ...x,
          dialog: {
            kind: "rename",
            wtPath: wt.path,
            buf: path.basename(wt.path),
          },
          banner: "",
        }));
        return;
      }
      if (input === "d") {
        if (s.worktrees.length <= 1) {
          setState((x) => ({
            ...x,
            banner: "No se puede eliminar el único worktree.",
          }));
          return;
        }
        const wt = s.worktrees[s.listCursor];
        if (!wt) {
          return;
        }
        setState((x) => ({
          ...x,
          dialog: { kind: "deleteConfirm", wtPath: wt.path },
          banner: "",
        }));
        return;
      }
      if (input === "i") {
        if (s.worktrees.length === 0 || !s.activeProject) {
          setState((x) => ({
            ...x,
            banner: "Activa un proyecto con worktrees (p).",
          }));
          return;
        }
        void (async () => {
          try {
            const sc = await readProjectConfig(s.activeProject);
            if (!(sc.setup ?? "").trim()) {
              setState((x) => ({
                ...x,
                banner:
                  "No hay scripts.setup (.topoductor/project.json o editor e).",
              }));
              return;
            }
            const wt = s.worktrees[s.listCursor];
            if (!wt) {
              return;
            }
            startScriptRun("Setup", wt.path, sc.setup ?? "");
          } catch (e) {
            setState((x) => ({
              ...x,
              banner:
                e instanceof Error ? e.message : String(e),
            }));
          }
        })();
        return;
      }
      if (input === "g") {
        if (s.worktrees.length === 0 || !s.activeProject) {
          setState((x) => ({
            ...x,
            banner: "Activa un proyecto con worktrees (p).",
          }));
          return;
        }
        void (async () => {
          try {
            const sc = await readProjectConfig(s.activeProject);
            if (!(sc.run ?? "").trim()) {
              setState((x) => ({
                ...x,
                banner:
                  "No hay scripts.run (.topoductor/project.json o editor e).",
              }));
              return;
            }
            const wt = s.worktrees[s.listCursor];
            if (!wt) {
              return;
            }
            startScriptRun("Run", wt.path, sc.run ?? "");
          } catch (e) {
            setState((x) => ({
              ...x,
              banner:
                e instanceof Error ? e.message : String(e),
            }));
          }
        })();
        return;
      }
      if (input === "z") {
        if (s.worktrees.length === 0 || !s.activeProject) {
          setState((x) => ({
            ...x,
            banner: "Activa un proyecto con worktrees (p).",
          }));
          return;
        }
        void (async () => {
          try {
            const sc = await readProjectConfig(s.activeProject);
            if (!(sc.archive ?? "").trim()) {
              setState((x) => ({
                ...x,
                banner:
                  "No hay scripts.archive (.topoductor/project.json o editor e).",
              }));
              return;
            }
            const wt = s.worktrees[s.listCursor];
            if (!wt) {
              return;
            }
            setState((x) => ({
              ...x,
              dialog: {
                kind: "archiveRunConfirm",
                wtPath: wt.path,
                line: sc.archive ?? "",
              },
              banner: "",
            }));
          } catch (e) {
            setState((x) => ({
              ...x,
              banner:
                e instanceof Error ? e.message : String(e),
            }));
          }
        })();
        return;
      }

      if (s.listLoading || s.listError || s.busy) {
        return;
      }

      const rows = s.worktrees;
      const cols = gridCols(termW);

      if (input === "k" || key.upArrow) {
        const c = withGridCursor(s.listCursor, rows.length, cols, 0, -1);
        setState((x) => ({ ...x, listCursor: c }));
        return;
      }
      if (input === "j" || key.downArrow) {
        const c = withGridCursor(s.listCursor, rows.length, cols, 0, 1);
        setState((x) => ({ ...x, listCursor: c }));
        return;
      }
      if (input === "h" || key.leftArrow) {
        const c = withGridCursor(s.listCursor, rows.length, cols, -1, 0);
        setState((x) => ({ ...x, listCursor: c }));
        return;
      }
      if (input === "l" || key.rightArrow) {
        const c = withGridCursor(s.listCursor, rows.length, cols, 1, 0);
        setState((x) => ({ ...x, listCursor: c }));
        return;
      }
      if (key.return && rows.length > 0) {
        const wt = rows[s.listCursor];
        if (wt) {
          setState((x) => ({
            ...x,
            dialog: {
              kind: "exit",
              path: wt.path,
              exitCursor: 0,
              customBuf: "",
            },
            banner: "",
          }));
        }
      }
    }
  });

  const backdropDimmed = state.dialog.kind !== "none";

  const topBar = (
    <Box flexDirection="column">
      <Text dimColor>
        TopoDuctor (Ink) v{version} ·{" "}
        {state.configPath ? state.configPath : "…"}
      </Text>
      <Text dimColor wrap="truncate">
        Ruta actual: {cwd}
      </Text>
    </Box>
  );

  if (state.bootstrapError) {
    return (
      <Box flexDirection="column">
        {topBar}
        <Text color="red">{state.bootstrapError}</Text>
        <Text dimColor>q salir</Text>
      </Box>
    );
  }

  if (state.view.kind === "bootstrap") {
    return (
      <Box flexDirection="column">
        {topBar}
        <Text color="cyan">Cargando proyectos…</Text>
      </Box>
    );
  }

  if (state.view.kind === "lobby") {
    const lobby = (
      <Box flexDirection="column">
        {topBar}
        <Text bold color="magenta" dimColor={backdropDimmed || undefined}>
          TopoDuctor
        </Text>
        <Text dimColor={backdropDimmed || undefined}>
          Añade un repositorio con <Text bold>p</Text> → <Text bold>a</Text> o
          elige uno existente.
        </Text>
        <Text dimColor={backdropDimmed || undefined}>
          p / enter proyectos · ctrl+c config · q salir
        </Text>
        {state.banner ? (
          <Box marginTop={1}>
            <Text
              color="yellow"
              dimColor={backdropDimmed || undefined}
            >
              {state.banner}
            </Text>
          </Box>
        ) : null}
      </Box>
    );
    if (state.dialog.kind === "none") {
      return lobby;
    }
    return (
      <ModalOverlay
        width={termW}
        height={termH}
        backdrop={lobby}
        modal={
          <Box
            borderStyle="round"
            borderColor="magenta"
            paddingX={2}
            paddingY={1}
            width={modalMaxWidth(termW)}
          >
            {renderModalInner(state)}
          </Box>
        }
      />
    );
  }

  if (state.view.kind === "projectPicker") {
    const picker = (
      <Box flexDirection="column">
        {topBar}
        <Text bold dimColor={backdropDimmed || undefined}>
          Proyectos
        </Text>
        <Text dimColor={backdropDimmed || undefined}>
          ↑↓ j/k · enter · a añadir · d quitar · b ramas pref. · esc/q · ctrl+c
          config
        </Text>
        <Box marginTop={1} flexDirection="column">
          {state.projectPaths.length === 0 ? (
            <Text dimColor={backdropDimmed || undefined}>
              (vacío — pulsa a)
            </Text>
          ) : (
            state.projectPaths.map((p, i) => (
              <Text
                key={p}
                inverse={i === state.projectCursor}
                dimColor={backdropDimmed || undefined}
              >
                {i === state.projectCursor ? "▸ " : "  "}
                {p}
              </Text>
            ))
          )}
        </Box>
        {state.banner ? (
          <Box marginTop={1}>
            <Text
              color="yellow"
              dimColor={backdropDimmed || undefined}
            >
              {state.banner}
            </Text>
          </Box>
        ) : null}
      </Box>
    );
    if (state.dialog.kind === "none") {
      return picker;
    }
    return (
      <ModalOverlay
        width={termW}
        height={termH}
        backdrop={picker}
        modal={
          <Box
            borderStyle="round"
            borderColor="magenta"
            paddingX={2}
            paddingY={1}
            width={modalMaxWidth(termW)}
          >
            {renderModalInner(state)}
          </Box>
        }
      />
    );
  }

  const projectLabel = state.activeProject || cwd;
  const showGridSelection =
    state.view.kind === "list" &&
    !state.listLoading &&
    !state.listError &&
    !state.busy;
  const listView = (
    <Box flexDirection="column">
      {topBar}
      <Text bold color="cyan" dimColor={backdropDimmed || undefined}>
        {projectLabel}
      </Text>
      <Text dimColor={backdropDimmed || undefined}>
        hjkl/↑↓←→ · enter salir/cd · n r d · p b e · i g z · ctrl+l lobby ·
        ctrl+c config · q
      </Text>
      {state.banner ? (
        <Text
          color="yellow"
          dimColor={backdropDimmed || undefined}
        >
          {state.banner}
        </Text>
      ) : null}
      <Box marginTop={1} flexDirection="column">
        {state.listLoading ? (
          <Text dimColor={backdropDimmed || undefined}>
            Cargando worktrees…
          </Text>
        ) : state.listError ? (
          <Text color="red" dimColor={backdropDimmed || undefined}>
            {state.listError}
          </Text>
        ) : (
          <WorktreeGrid
            worktrees={state.worktrees}
            cursor={state.listCursor}
            termW={termW}
            showSelection={showGridSelection}
            marqueePhase={marqueeTick}
            activeProject={state.activeProject}
            dimmed={backdropDimmed}
          />
        )}
      </Box>
    </Box>
  );
  if (state.dialog.kind === "none") {
    return listView;
  }
  return (
    <ModalOverlay
      width={termW}
      height={termH}
      backdrop={listView}
      modal={
        <Box
          borderStyle="round"
          borderColor="magenta"
          paddingX={2}
          paddingY={1}
          width={modalMaxWidth(termW)}
        >
          {renderModalInner(state)}
        </Box>
      }
    />
  );
}

function renderModalInner(state: AppState): React.ReactNode {
  const d = state.dialog;
  switch (d.kind) {
    case "none":
      return null;
    case "claudePrompt": {
      const scrollPieces: string[] = [];
      if (d.err) {
        scrollPieces.push(d.err);
      }
      if (d.out) {
        scrollPieces.push(d.out);
      }
      const scrollSrc =
        scrollPieces.length > 0 ? scrollPieces.join("\n---\n") : "";
      const scrollLines = scriptOutLines(scrollSrc || "(sin salida)");
      return (
        <Box flexDirection="column">
          <Text bold>Claude Code</Text>
          <Text dimColor wrap="truncate">
            {d.workDir}
          </Text>
          <Text dimColor>
            Modo -p (respuesta de texto; no es la TUI interactiva completa de
            Claude). Tareas con herramientas pueden requerir --allowedTools.
          </Text>
          {d.loading ? (
            <Text color="cyan">Ejecutando… · esc cancelar</Text>
          ) : null}
          {!d.loading && !d.ran ? (
            <>
              <Text dimColor>enter ejecutar · esc cerrar</Text>
              <Text>
                › {d.promptBuf}
                <Text backgroundColor="white" color="black">
                  {" "}
                </Text>
              </Text>
            </>
          ) : null}
          {d.ran && !d.loading ? (
            <>
              <Box flexDirection="column">
                {scrollLines
                  .slice(d.scroll, d.scroll + scriptRunVisible)
                  .map((line, i) => (
                    <Text key={`${d.scroll}-${i}`}>{line}</Text>
                  ))}
              </Box>
              <Text dimColor>
                j/k scroll · enter/esc cerrar
                {scrollLines.length > scriptRunVisible
                  ? ` · ${d.scroll + 1}-${Math.min(
                      d.scroll + scriptRunVisible,
                      scrollLines.length
                    )} / ${scrollLines.length}`
                  : ""}
              </Text>
            </>
          ) : null}
        </Box>
      );
    }
    case "addProjectPath":
      return (
        <Box flexDirection="column">
          <Text bold>Ruta del repo</Text>
          <Text dimColor>enter confirmar · esc cancelar</Text>
          <Text>
            › {state.addPathBuffer}
            <Text backgroundColor="white" color="black">
              {" "}
            </Text>
          </Text>
          {state.banner ? (
            <Box marginTop={1}>
              <Text color="yellow">{state.banner}</Text>
            </Box>
          ) : null}
        </Box>
      );
    case "settings":
      return (
        <Box flexDirection="column">
          <Text bold>Configuración</Text>
          {d.checking || d.applying ? (
            <Text color="cyan">
              {d.checking
                ? "Comprobando GitHub…"
                : "Ejecutando Homebrew (puede tardar)…"}
            </Text>
          ) : (
            <>
              {d.err ? <Text color="red">{d.err}</Text> : null}
              {d.notice ? <Text>{d.notice}</Text> : null}
              {d.releaseURL && d.hasNewer ? (
                <Text dimColor>{d.releaseURL}</Text>
              ) : null}
              <Text dimColor>
                u comprobar de nuevo · i upgrade brew (macOS, si hay nueva) ·
                esc cerrar
              </Text>
            </>
          )}
        </Box>
      );
    case "scriptRun":
      return (
        <Box flexDirection="column">
          <Text bold>
            {d.title} — {d.workDir}
          </Text>
          <Text dimColor>{d.cmd}</Text>
          {d.loading ? (
            <Text color="cyan">Ejecutando…</Text>
          ) : d.err ? (
            <Text color="red">{d.err}</Text>
          ) : (
            <Box flexDirection="column">
              {scriptOutLines(d.out)
                .slice(d.scroll, d.scroll + scriptRunVisible)
                .map((line, i) => (
                  <Text key={`${d.scroll}-${i}`}>{line}</Text>
                ))}
            </Box>
          )}
          <Text dimColor>enter/esc cerrar · j/k scroll</Text>
        </Box>
      );
    case "scriptEdit":
      return (
        <Box flexDirection="column">
          <Text bold>Scripts (.topoductor/project.json)</Text>
          {d.loadErr ? <Text color="red">{d.loadErr}</Text> : null}
          <Text dimColor>tab cambiar campo · enter guardar · esc</Text>
          <Text inverse={d.focus === 0}>
            {d.focus === 0 ? "▸ " : "  "}
            setup — {d.setup || "·"}
          </Text>
          <Text inverse={d.focus === 1}>
            {d.focus === 1 ? "▸ " : "  "}
            run — {d.run || "·"}
          </Text>
          <Text inverse={d.focus === 2}>
            {d.focus === 2 ? "▸ " : "  "}
            archive — {d.archive || "·"}
          </Text>
        </Box>
      );
    case "branchPrefs":
      return (
        <Box flexDirection="column">
          <Text bold>Ramas preferidas — {d.repoPath}</Text>
          <Text dimColor>3 campos · tab · enter guardar · esc</Text>
          {[0, 1, 2].map((i) => {
            const f = `b${i}` as "b0" | "b1" | "b2";
            const inv = d.focus === i;
            return (
              <Text key={f} inverse={inv}>
                {i + 1}. {d[f]}
              </Text>
            );
          })}
        </Box>
      );
    case "archiveRunConfirm":
      return (
        <Box flexDirection="column">
          <Text bold>¿Ejecutar script archive?</Text>
          <Text dimColor>{d.line}</Text>
          <Text dimColor>y enter / n esc</Text>
        </Box>
      );
    case "deleteConfirm":
      return (
        <Box flexDirection="column">
          <Text bold>¿Eliminar worktree?</Text>
          <Text>{d.wtPath}</Text>
          <Text dimColor>y enter / n esc</Text>
        </Box>
      );
    case "rename":
      return (
        <Box flexDirection="column">
          <Text bold>Renombrar carpeta</Text>
          <Text dimColor>{d.wtPath}</Text>
          <Text>› {d.buf}</Text>
        </Box>
      );
    case "createName":
      return (
        <Box flexDirection="column">
          <Text bold>Nueva rama / carpeta desde {d.baseRef}</Text>
          <Text dimColor>enter crear · esc volver a ramas</Text>
          <Text>› {d.nameBuf}</Text>
        </Box>
      );
    case "createPick": {
      const prefs =
        state.preferredBranches[prefKey(state.activeProject)] ?? [];
      const all = d.branches ?? [];
      const f = filteredCreateBranches(all, d.filter, prefs);
      const c = d.loading ? 0 : clamp(d.brCursor, 0, Math.max(0, f.length - 1));
      const sc = d.loading
        ? 0
        : adjustBranchScroll(c, d.brScroll, createBranchVisible, f.length);
      const start = sc;
      const end = Math.min(start + createBranchVisible, f.length);
      return (
        <Box flexDirection="column">
          <Text bold>Rama base</Text>
          {d.loading ? (
            <Text color="cyan">Cargando ramas…</Text>
          ) : d.loadErr ? (
            <Text color="red">{d.loadErr}</Text>
          ) : (
            <>
              <Text dimColor>filtro: {d.filter || "·"}</Text>
              {f.length === 0 ? (
                <Text dimColor>— sin coincidencias —</Text>
              ) : (
                f.slice(start, end).map((br, idx) => {
                  const i = start + idx;
                  const sel = i === c;
                  return (
                    <Text key={`${br}-${i}`} inverse={sel}>
                      {sel ? "› " : "  "}
                      {br}
                    </Text>
                  );
                })
              )}
              <Text dimColor>
                ↑↓ · escribir filtra · enter · esc volver · q salir app
              </Text>
            </>
          )}
        </Box>
      );
    }
    case "exit":
      return (
        <Box flexDirection="column">
          <Text bold>Al salir, usar:</Text>
          <Text inverse={d.exitCursor === 0}>
            {d.exitCursor === 0 ? "› " : "  "}
            Terminal (cd + $SHELL)
          </Text>
          <Text inverse={d.exitCursor === 1}>
            {d.exitCursor === 1 ? "› " : "  "}
            Cursor (abrir carpeta)
          </Text>
          <Text inverse={d.exitCursor === 2}>
            {d.exitCursor === 2 ? "› " : "  "}
            Claude Code (prompt · claude -p aquí)
          </Text>
          <Text inverse={d.exitCursor === 3}>
            {d.exitCursor === 3 ? "› " : "  "}
            Comando personalizado — {'{path}'} = ruta
          </Text>
          {d.exitCursor === 3 ? (
            <Text>
              › {d.customBuf}
              <Text backgroundColor="white" color="black">
                {" "}
              </Text>
            </Text>
          ) : null}
          <Text dimColor>
            ↑↓ · enter ejecutar · esc cerrar panel · q atajo terminal (cd)
          </Text>
        </Box>
      );
    default: {
      const _x: never = d;
      void _x;
      return (
        <Box flexDirection="column">
          <Text>…</Text>
        </Box>
      );
    }
  }
}

