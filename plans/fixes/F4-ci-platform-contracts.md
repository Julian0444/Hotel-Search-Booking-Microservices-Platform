# Fix F4 — CI: el leg de platform-contracts apunta a un go.sum inexistente

> **Alcance:** RV13
> **Prerequisitos:** ninguno. **Hacerlo antes de que el CI corra por primera vez** con esta matriz (si no, ese primer run puede salir rojo por esto y ensuciar el diagnóstico).
> **Esfuerzo:** minutos (XS)

## Contexto

`.github/workflows/ci.yml:20` configura el cache de `actions/setup-go@v5` con:

```yaml
          cache-dependency-path: ${{ matrix.module }}/go.sum
```

La matriz incluye `platform-contracts` (`:11`), pero ese módulo **no tiene `go.sum`** (cero dependencias: solo `contracts.go`, `contracts_test.go`, `go.mod`, `testdata/`). `setup-go` con cache habilitado falla el job cuando el path de dependencias no resuelve a ningún archivo. Como el CI todavía no corrió nunca con esta matriz, está latente.

## Pasos

### 1. Usar un glob que siempre matchee

En `.github/workflows/ci.yml:20`:

```yaml
          cache-dependency-path: ${{ matrix.module }}/go.*
```

`cache-dependency-path` acepta globs y solo se usa como clave de hash del cache: para los 3 servicios matchea `go.mod` + `go.sum` (clave igual de buena) y para `platform-contracts` matchea `go.mod`. Alternativa equivalente si se prefiere explícita: `cache: false` solo para ese leg (con un `if` en la matriz) — el glob es más simple.

### 2. Los otros jobs no necesitan cambios (verificado 2026-07-11)

El job `integration` (`ci.yml:47`) apunta fijo a `hotels-api/go.sum`, que existe; el job `frontend` usa `package-lock.json`, que existe. Solo la matriz del job `go` tiene el problema.

## Verificar

```bash
# Validación local de sintaxis (si está instalado):
actionlint .github/workflows/ci.yml || true
```

La verificación real es el **primer run del CI en GitHub Actions**: los 4 legs de la matriz (`users-api`, `hotels-api`, `search-api`, `platform-contracts`) deben ponerse verdes, en particular el paso "Setup Go" del leg `platform-contracts`. Aprovechar ese run para la advertencia pendiente del HANDOFF (2026-07-04): confirmar que los runners resuelven bien los toolchains 1.24/1.25 de los `go.mod`.

Al terminar: tick en `plans/fixes/README.md` y sección nueva en `plans/HANDOFF.md`.
