# Git Worktree Orchestrator — guía para agentes

## Propósito

Herramienta en terminal (TUI) para **gestionar git worktrees con poca fricción**: listar worktrees, **crear uno en el primer arranque si hace falta**, y **cambiar de contexto** (moverse entre directorios de trabajo asociados a cada worktree).

Comportamiento implementado:

- **Primera ejecución (por repo)**: si no existe aún el worktree “gestionado”, se crea uno hermano del toplevel con nombre `<repo>-wt-<uuid>` y rama `wt-<uuid>`. El estado se guarda en `<git-common-dir>/worktree-orchestrator.json` para no duplicar worktrees.
- **Uso habitual**: lista todos los worktrees (`git worktree list --porcelain`), navegación con j/k, **Enter** fija la ruta elegida; al salir, si hubo selección, se imprime `cd "<path>"` en stdout para copiar o evaluar en la shell.

## Stack

- **Go** (ver `go.mod` para la versión).
- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) — modelo `Init` / `Update` / `View`.
- **Lip Gloss** — estilos de la UI.
- **Bubbles** — `key` para mapas de teclas (`tui/keys.go`).

Módulo: `github.com/macpro/git-worktree-orchestrator`.

## Estructura del repo

| Ruta | Rol |
|------|-----|
| `main.go` | Punto de entrada: `Getwd`, programa Bubble Tea, imprime `cd` en stdout si hubo selección. |
| `internal/gitworktree/` | Encapsula `git`: bootstrap del worktree UUID, listado porcelain, estado en el git dir común. |
| `tui/model.go` | Modelo Bubble Tea: carga async, lista, errores, vista. |
| `tui/load.go` | Comando inicial que llama al paquete gitworktree. |
| `tui/keys.go` | Definición de atajos (arriba/abajo, seleccionar, salir). |

## Convenciones al implementar

- Mantener la TUI **acoplada de forma delgada** a Git: idealmente encapsular `git worktree` (listar, agregar, etc.) en un paquete o funciones dedicadas y dejar `tui` como capa de presentación y mensajes.
- **Errores de Git** (no es repo, comando fallido): mostrar mensaje claro en la TUI o en stderr según el patrón que elija el proyecto; no silenciar fallos.
- El tipo `Worktree` en `tui/model.go` debe alinearse con lo que devuelva la integración real (`Name`, `Branch`, y campos extra si hacen falta para “moverse” al worktree).
- Respetar el estilo existente: comentarios en inglés donde ya lo estén, nombres exportados con sentido, sin refactors masivos fuera del alcance del cambio.
- Tras añadir lógica de “primer arranque”, considerar **persistencia mínima** (por ejemplo, marcar en un archivo de estado o detectar por `git worktree list`) para no crear worktrees duplicados en cada ejecución.

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
2. **Primera corrida**: crear worktree con nombre basado en **UUID** si corresponde a la lógica de onboarding.
3. El código vivo está en Go + Bubble Tea bajo `tui/` y `main.go`.
