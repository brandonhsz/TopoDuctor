# TopoDuctor

TUI de terminal para **gestionar [git worktrees](https://git-scm.com/docs/git-worktree)**: listar, crear, renombrar, eliminar y **salir con contexto** (`cd`, Cursor o comando con `{path}`).

Implementación actual: **Node.js + React Ink** (sin dependencia de Go).

## Requisitos

- **Node.js** 20+ (recomendado 22)
- **Git** en `PATH`

## Homebrew (macOS)

La app es una **fórmula** de Homebrew que instala el paquete npm `topoductor` (CLI en Node). Ya no se usa cask ni binario Go.

1. Publica la versión en npm (`npm publish`) cuando corresponda; el tarball incluye `dist/` vía `prepublishOnly`.
2. Actualiza en `Formula/topoductor.rb` la URL del `.tgz` y el `sha256` del tarball **publicado** (el hash puede no coincidir con `npm pack` local; ver comentario en el archivo).
3. Instalación desde el repo:

```bash
brew install --formula ./Formula/topoductor.rb
```

O desde una etiqueta en GitHub (sustituye el tag):

```bash
brew install --formula https://raw.githubusercontent.com/brandonhsz/TopoDuctor/v0.0.1/Formula/topoductor.rb
```

Actualizar desde la TUI o la terminal: `brew update && brew upgrade topoductor` (sin `--cask`).

## Uso

Desde la raíz del repo:

```bash
npm install
npm start
```

Flags:

```bash
npx tsx src/cli.tsx --print-only
npx tsx src/cli.tsx --version
```

| Flag            | Descripción                                                                 |
|-----------------|-----------------------------------------------------------------------------|
| `--print-only`  | No ejecuta `cd` ni shell: imprime el comando en stdout (p. ej. para `eval`). |
| `--version`     | Muestra la versión del paquete y termina.                                    |

## Comportamiento (resumen)

- **Proyectos** en `projects.json` (directorio de config del usuario, misma convención que la app histórica en Go).
- Worktrees nuevos bajo `~/.topoDuctor/projects/…`.
- Teclas principales: **hjkl** / flechas en la rejilla, **Enter** para salir con opción, **n** crear, **r** renombrar, **d** borrar, **p** proyectos, **ctrl+l** lobby, **ctrl+c** ajustes / comprobar actualización.

## Desarrollo

```bash
npm run typecheck
```

## Licencia

La que indique el repositorio (revisa el archivo `LICENSE` si existe).
