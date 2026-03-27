# TopoDuctor

A terminal UI (TUI) for **managing [git worktrees](https://git-scm.com/docs/git-worktree)** with minimal friction: see what you have, create more, rename a checkout folder, remove them, and **switch context** on exit (for example `cd` into the chosen directory or open it in Cursor).

## What it does

- **Lists** worktrees for the active repository using `git worktree list --porcelain`. It does not create any when you launch the app.
- **Projects**: register multiple clone paths and switch between them without leaving the TUI. The list and active project are stored in a JSON file under the user config directory.
- **Create** a new worktree (**n**): pick a base branch (or ref), then a name; a new branch is created and the checkout goes under `~/.topoDuctor/projects/...`.
- **Rename** the selected worktree folder (**r**) and **delete** one (**d**), with confirmation on delete.
- **Exit with a chosen worktree** (**Enter** on a card): run an interactive `cd`, open the folder in **Cursor**, or a **custom command** using the `{path}` placeholder.

## Requirements

- **Go** 1.26 or compatible (see `go.mod`).
- **Git** installed and on `PATH`.
- A terminal that supports the alternate screen (standard TUI mode).

## Installation

From the repository root:

```bash
go build -o topoductor .
```

Or run without a separate binary:

```bash
go run .
```

You can also install with `go install` if the module is available from a remote:

```bash
go install github.com/macpro/topoductor@latest
```

(Adjust the module path if your fork or mirror uses a different import path.)

### Homebrew (tap)

After a [GoReleaser](https://goreleaser.com/) release (see `.goreleaser.yaml` and `.github/workflows/release.yml`):

```bash
brew tap brandonhsz/tap
brew install --cask topoductor
```

**Maintainers — one-time setup**

1. Repository [brandonhsz/homebrew-tap](https://github.com/brandonhsz/homebrew-tap) must exist (empty is fine).
2. In **TopoDuctor** → *Settings* → *Secrets and variables* → *Actions*, add **`GORELEASER_GITHUB_TOKEN`**: a classic PAT with the **`repo`** scope, or a fine-grained token with **Contents: Read and write** on **`brandonhsz/TopoDuctor`** and **`brandonhsz/homebrew-tap`**. The default `GITHUB_TOKEN` in Actions cannot push to the tap repo; see [GoReleaser: resource not accessible](https://goreleaser.com/errors/resource-not-accessible-by-integration/).
3. Push a version tag to run the release workflow, for example: `git tag v0.1.0 && git push origin v0.1.0`.

If you fork the project, update `release.github` and `homebrew_casks.repository` in `.goreleaser.yaml` to match your GitHub owner and repo names.

## Usage

```text
topoductor [-print-only] [-version]
```

| Flag | Description |
|------|-------------|
| `-print-only` | Does not `cd` or exec the shell: only prints the action to stdout (e.g. `cd "…"` or the Cursor command). Handy for `eval "$(topoductor -print-only)"` or scripts. |
| `-version` | Prints the build version and exits (set at release time via GoReleaser `ldflags`). |

You can run the binary **from any directory**; the current working directory only affects whether you see the **lobby** at startup or go straight to a project’s worktree list (see below).

## Lobby and projects

On startup the app reads `projects.json` (path under [Configuration files and paths](#configuration-files-and-paths)).

- If **there are no registered projects**, or the **cwd is not a git repository**, or the **repo toplevel for cwd is not** in the project list, you see the **lobby**: a minimal screen where you open the project picker with **p** or **Enter**.
- With one or more projects and cwd **inside** a repo that **is** on the list, it opens the **worktree view** for that project (or the saved active one, as applicable).

In the **project picker** (**p** from the main list):

- **↑/↓** or **k/j**: move the cursor.
- **Enter**: activate the project and load its worktrees.
- **a**: add a clone path (must be a valid directory).
- **b**: preferred branches (up to 3 per project) shown first when creating a worktree.
- **d**: remove the project from the list (does not delete files on disk).

From the worktree list, **Ctrl+L** returns to the lobby (unless you are already in the lobby).

## Main shortcuts (worktree view)

| Key | Action |
|-----|--------|
| **↑/↓** or **k/j** | Move selection on the grid (including across rows). |
| **←/→** or **h/l** | Move across columns in the grid. |
| **Enter** | Confirm worktree and choose [how to exit](#exit-cd-cursor-or-custom-command). |
| **n** | Create a new worktree (requires an active project). |
| **r** | Rename the selected worktree folder. |
| **d** | Remove the selected worktree (cannot be the only one). |
| **p** | Open the project picker. |
| **b** | Branch preferences for the active project. |
| **q** or **Ctrl+C** | Quit without changing directory (unless you already confirmed exit with Enter). |

In the **create** flow (**n**): first pick the base branch (with text filter and scroll), then enter the new branch/folder name and confirm with **Enter**. **Esc** goes back one step.

For **rename** and **delete confirm**, **Esc** cancels; on delete, **y** or **Enter** confirms, **n** cancels.

## Exit: `cd`, Cursor, or custom command

After **Enter** on a card, the exit dialog appears:

1. **Open shell here** — `chdir` to the worktree path and replace the process with your `$SHELL` (interactive mode on bash/zsh).
2. **Open in Cursor** — `cursor <path>` or on macOS `open -a Cursor` if the CLI is not on `PATH`.
3. **Custom command** — one line that may include `{path}`; it is replaced with a properly quoted path for the shell.

With **`-print-only`**, only the equivalent line is printed (e.g. `cd "/path/to/worktree"`).

### Windows limitations

On **Windows**, the program **cannot** replace the process with the shell after `cd` like on Unix. You will see a message telling you to use `-print-only` or copy the command. Custom command mode with `-print-only` is meant for running the line manually when needed.

## Configuration files and paths

| Location | Purpose |
|----------|---------|
| `~/.config/topoductor/projects.json` (OS-specific via `UserConfigDir`) | Repo paths, active project, and per-repo preferred branches. |
| `~/.topoDuctor/projects/<segment>/worktree/<name>` | New checkouts created from the TUI (segment derived from the repo). |
| `<common-git-dir>/topoductor.json` | Optional legacy state: syncs the “managed” path if it existed in older versions when moving or removing worktrees; **not** written on startup. |

## Technical overview

- Domain in `internal/worktree/` (`Worktree` type and `Service` port).
- Git CLI implementation in `internal/worktree/git/`.
- TUI with [Bubble Tea](https://github.com/charmbracelet/bubbletea), styling with Lip Gloss.

## Tests

```bash
go test ./...
```

## Contributing / agent docs

Architecture details and conventions: [AGENTS.md](./AGENTS.md).
