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

## Estructura del repo

| Ruta | Rol |
|------|-----|
| `main.go` | Punto de entrada: `Getwd`, TUI; tras elegir ruta, `exec` del shell o `-print-only`. |
| `internal/gitworktree/` | Encapsula `git`: listado porcelain, add/move/remove, estado opcional en el git dir común. |
| `tui/model.go` | Modelo Bubble Tea: carga async, lista, errores, vista. |
| `tui/load.go` | Comando inicial que llama al paquete gitworktree. |
| `tui/keys.go` | Definición de atajos (arriba/abajo, seleccionar, salir). |

## Convenciones al implementar

- Mantener la TUI **acoplada de forma delgada** a Git: idealmente encapsular `git worktree` (listar, agregar, etc.) en un paquete o funciones dedicadas y dejar `tui` como capa de presentación y mensajes.
- **Errores de Git** (no es repo, comando fallido): mostrar mensaje claro en la TUI o en stderr según el patrón que elija el proyecto; no silenciar fallos.
- El tipo `Worktree` en `tui/model.go` debe alinearse con lo que devuelva la integración real (`Name`, `Branch`, y campos extra si hacen falta para “moverse” al worktree).
- Respetar el estilo existente: comentarios en inglés donde ya lo estén, nombres exportados con sentido, sin refactors masivos fuera del alcance del cambio.
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
