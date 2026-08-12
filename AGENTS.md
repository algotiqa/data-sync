# Data-sync — AGENTS.md

Standalone Go module `github.com/algotiqa/data-sync` (Go 1.25). CLI tool + library
to migrate table data between SQL databases via the DDF format. Ported from the
Java `ddf` project (`../ddf`). Single module; do not run Go commands from the
parent `algotiqa/git` workspace directory.

## Layout

- Root `package main` — Cobra wiring only: `main.go` registers the subcommands.
- `pkg/command/` — one package per CLI command, each exposing `New<Cmd>Cmd()`:
  `connection.go` (shared flags, DSN build, env fallback), `export/`, `import/`,
  `run/`, `migrate/`. The `import` directory declares `package import_`
  (Go keyword) — alias it as `import_` when importing. `run` and `import`
  also export execution helpers (`ExecuteSQLFile`/`ExecuteStatements`,
  `ExecuteDDFFile`) reused by `migrate`; do not duplicate that logic.
- `pkg/ddf/` — the importable library (`github.com/algotiqa/data-sync/pkg/ddf`):
  `codec.go` (`~XXXX` string codec + hex bytes), `sqlmapper.go` (DB-type to
  canonical JDBC-type maps), `writer.go`/`reader.go` (DDF file I/O),
  `exporter.go`/`importer.go`, `jdbutil.go` (row scan/format, PK lookup,
  `?`/`$n` placeholders). Keep it importable; the CLI depends on it.

## Commands

- `go build ./...` — acceptance baseline; must stay clean.
- `go vet ./...` — passes.
- `go test ./...` — pure unit tests, no DB needed, in three packages:
  `pkg/ddf/`, `pkg/command/run/`, `pkg/command/migrate/`.
- No Makefile, lint, typecheck, or CI scripts exist.

## Gotchas

- **MIT license, not ELv2.** The workspace-root `AGENTS.md` mandates ELv2 headers,
  but this repo deliberately uses MIT (`Copyright (c) 2026 Algotiqa`) plus a root
  `LICENSE` file. Replicate MIT on new files; do not "fix" the headers.
- **Not gofmt-clean on purpose.** `gofmt -l .` reports most files (hand-formatted
  alignment). Do not run `gofmt -w`; match the surrounding file style.
- The built `data-sync` binary at the repo root is gitignored — never commit it.
- DDF quirks: empty token = NULL (empty BLOB/CLOB cannot round-trip); `~` (0x7E)
  is itself encoded `~007E`; timestamps are UTC `2006-01-02 15:04:05.000`;
  `[DATA]` is a legacy alias for `[INSERT]`; CLOB is decoded as an encoded string
  (deliberately fixes a Java round-trip bug).
- `--dbtype mysql|postgres` selects the driver and SQL dialect (`?` vs `$n`
  placeholders, dialect-specific `information_schema` PK query). Postgres defaults
  the PK schema to `public`; MySQL uses `DATABASE()`.
- Connection flags are overridable by `DB_*` env vars (`DB_TYPE`, `DB_HOST`,
  `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_DSN`); precedence is
  flag > env > default.
- Import aborts on the first failing row by default; `--continue-on-error` skips
  rows. Import auto-detects gzip via magic bytes; export compresses with
  `--compress` or a `.gz` filename.
- `run` executes the statements of a plain SQL file (split on lines ending with
  `;`, the trailing `;` is stripped, `--` comment and blank/empty statements are
  skipped); it aborts on the first failing statement. Reuses the connection
  flags from the other commands.
- `migrate` scans `<dir>/<DB_NAME>` (default `dir=db`) for files named
  `<version>_<name>.sql|.ddf` (any other extension errors) and applies, in
  alphabetical order, only those whose version is greater than the value stored
  in the auto-created `migration (last_version INT)` table. The version is
  updated transactionally after each successful file; `.sql` files reuse the run
  logic, `.ddf` files the import logic. Requires `--db`.
- Live runs need a running MySQL or PostgreSQL. The platform stack
  (`environment/`) exposes MySQL :3400-3403 and PostgreSQL/TimescaleDB :3410,
  but a plain local MySQL on :3306 (user `admin`/`admin`, dev DBs `collector`,
  `portfolio`, `inventory`, `event`) also works.