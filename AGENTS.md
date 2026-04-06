# TopoDuctor — guía para agentes

## Propósito

TUI para **orquestar git worktrees**: listar, crear (**n**), renombrar (**r**), eliminar (**d**), proyectos múltiples, scripts de repo, y **salida** con `cd` / Cursor / comando custom (`--print-only` para imprimir solo el comando).

## Stack

- **Node.js** (ESM) + **TypeScript**
- **React 18** + **Ink 5** (`render`, `useInput`, `useStdout`, `Box`, `Text`)
- **git** vía `child_process` (`src/git/spawn.ts`)

## Estructura del repo

| Ruta | Rol |
|------|-----|
| `src/cli.tsx` | Args (`--print-only`, `--version`), `render`, `waitUntilExit`, `runExitAction` |
| `src/App.tsx` | Shell que monta `TopoductorUi` |
| `src/TopoductorUi.tsx` | Estado global, modales, teclado, vistas |
| `src/git/` | Spawn git, porcelain, operaciones worktree, sanitize, `topoductor.json` |
| `src/projects/` | `projects.json`, lobby, `project.json`, scripts shell |
| `src/update/` | GitHub releases, `brew upgrade topoductor` (fórmula npm, no cask) |
| `src/ui/` | Rejilla de tarjetas, marquee, navegación grid |
| `src/exit/` | Post-salida: shell, Cursor, template `{path}` |

## Convenciones

- Paridad de comportamiento con la lógica documentada en el README (rutas de config, worktrees bajo `~/.topoDuctor`).
- Errores de git: mostrar en banner o stderr; no silenciar fallos reales.
- Respuestas al usuario en **español** si el producto ya usa ese idioma en la UI.

## Ejecutar / probar

```bash
npm install
npm start
npm run typecheck
```
