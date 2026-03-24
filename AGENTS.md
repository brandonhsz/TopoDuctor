# Git Worktree Orchestrator — guía para agentes

## Propósito

Herramienta en terminal (TUI) para **gestionar git worktrees con poca fricción**: listar, crear (**n**), renombrar carpeta (**r**), eliminar (**d**), y **cambiar de contexto** (cd al worktree elegido al salir).

Comportamiento implementado:

- **Al abrir**: solo lista worktrees existentes (`git worktree list --porcelain`); no se crea ninguno automáticamente.
- **Crear**: atajo **n** y nombre deseado (carpeta `<repo>-<nombre>` y rama con el mismo slug).
- **Salida con selección**: por defecto `Chdir` + `exec` del `$SHELL` en esa ruta; con `-print-only` solo se imprime `cd "…"` en stdout.
- **Estado opcional**: `worktree-orchestrator.json` en el git dir común solo se usa para sincronizar la ruta “gestionada” si existía en versiones anteriores (mover/borrar); ya no se rellena al arrancar.

## Stack

- **Go** (ver `go.mod` para la versión).
- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) — modelo `Init` / `Update` / `View`.
- **Lip Gloss** — estilos de la UI.
- **Bubbles** — `key` para mapas de teclas (`tui/keys.go`).

Módulo: `github.com/macpro/git-worktree-orchestrator`.

## Arquitectura (vertical slice + puertos)

- **Slice vertical** `internal/worktree/`: dominio del orquestador (`Worktree`) y **puerto** `Service` (listar / crear / mover / borrar). La TUI y `main` solo dependen de este contrato.
- **Adaptador** `internal/worktree/git/`: implementa `worktree.Service` con el CLI de git (`Runner`, porcelain, estado JSON). Sustituible por tests o otro backend (mock que implemente `worktree.Service`).
- **`tui/`**: driver Bubble Tea; recibe `worktree.Service` inyectado en `tui.New(svc, workDir, printOnly)` — sin imports de `git` ni `exec`.
- **`main.go`**: cablea `wtgit.NewService(wd)` → `tui.New`.

## Estructura del repo

| Ruta | Rol |
|------|-----|
| `main.go` | `Getwd`, `wire` git adapter + TUI; `exec` shell o `-print-only`. |
| `internal/worktree/` | Dominio `Worktree`, interfaz `Service` (puerto). |
| `internal/worktree/git/` | Adaptador git: `Runner`, `Adapter`, parse porcelain, ops, tests. |
| `tui/model.go` | Modelo Bubble Tea: carga, grid, marquee, modos. |
| `tui/load.go` | `tea.Cmd` que llama `svc.List()`. |
| `tui/op.go` | Comandos async post mutación (add/move/remove). |
| `tui/keys.go` | Atajos de teclado. |

## Convenciones al implementar

- **Nuevas operaciones sobre worktrees**: añadir método al puerto `worktree.Service`, implementar en `internal/worktree/git`, usar desde `tui/op.go` o `load.go`.
- **Errores de Git**: propagar al mensaje de la TUI o stderr; no silenciar.
- **Dominio**: `worktree.Worktree` usa `Path`, `Branch`, `Head`; en la UI el “nombre” de carpeta es `filepath.Base(Path)`.
- Respetar estilo existente; comentarios en inglés en código donde ya aplique.
## Cómo ejecutar

```bash
go run .
```

Para compilar un binario:

```bash
go build -o git-worktree-orchestrator .
```

## Pruebas

Cuando existan tests, ejecutar `go test ./...`. Si aún no hay tests, priorizar tests unitarios en la capa que ejecute comandos git (con interfaces o `exec.Command` inyectable) para no depender del estado real del disco en CI.

## Resumen para el modelo

1. El objetivo del producto es **orquestar worktrees y facilitar el cambio de contexto entre ellos**.
2. **No hay creación automática al abrir**; el usuario crea worktrees con **n** cuando lo necesite.
3. El código vivo está en Go + Bubble Tea bajo `tui/` y `main.go`.
