# Data-sync

CLI tool and Go library to migrate table data between SQL databases using the
**DDF** (Data Description Format) format. It is a Go port of the Java `ddf`
project.

DDF is a simple, database-independent text format that stores the schema of a
table (columns, types, primary keys) together with its rows, so the same file
can be exported from one database and imported into another. Data can optionally
be compressed with gzip.

The tool provides four commands:

| command   | purpose                                                        |
|-----------|----------------------------------------------------------------|
| `export`  | export the data of a table or query into a DDF file            |
| `import`  | import the data of a DDF file into a database table            |
| `run`     | execute the SQL statements contained in a plain text file      |
| `migrate` | apply a set of pending `.sql`/`.ddf` migration files in order  |

## Requirements

- Go 1.25 or later (only needed to build from source)
- A running **MySQL** database (it also works with MariaDB) **or** a running
  **PostgreSQL** database
- The database user must have the rights to create/read/write tables and, for
  `migrate`, to create the `migration` bookkeeping table

## Build

```sh
go build -o data-sync .
```

This produces the `data-sync` binary in the current folder. You can also run it
without building with `go run .`.

## Connecting to the database

Every command accepts the following flags to connect to the database:

| Flag            | Default       | Description                                     |
|-----------------|---------------|-------------------------------------------------|
| `--dbtype`      | `mysql`       | database type: `mysql` or `postgres`            |
| `--host`        | `127.0.0.1`   | database host                                   |
| `--port`        | (see below)   | database port                                   |
| `--user`        | `root`        | database user                                   |
| `--password`    | *(empty)*     | database password                               |
| `--db`          | *(empty)*     | database name                                   |
| `--dsn`         | *(empty)*     | full connection string, overrides the flags above |

The default port depends on the database type: `3306` for MySQL, `5432` for
PostgreSQL.

Each connection flag can also be provided through a `DB_*` environment variable
(`DB_TYPE`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`,
`DB_DSN`). Precedence is: **flag > environment variable > default**.

## Commands

### `export`

```
data-sync export --query <sql> | --table <table> [flags]
```

Exports the result of a query, or all the rows of a table, into a DDF file.

| Flag          | Description                                                                 |
|---------------|-----------------------------------------------------------------------------|
| `--query`     | SQL query to export (alternative to `--table`)                              |
| `--table`     | table to export (alternative to `--query`)                                  |
| `--file`      | output DDF file (`-` for stdout, default `<table>.ddf`)                    |
| `--compress`  | compress the output with gzip (also enabled by a `.gz` file name)           |

When a single table is exported, the file header records its primary keys, so
the same file can be used for updates and deletes on import.

### `import`

```
data-sync import --file <file> [flags]
```

Imports the records of a DDF file into a database table. By default it applies
`INSERT`, `UPDATE` and `DELETE` records to the table declared inside the file.

| Flag                 | Default   | Description                                                      |
| -------------------- | --------- | ---------------------------------------------------------------- |
| `--file`             | *(required)* | input DDF file (`-` for stdin)                                   |
| `--table`            | *(file's)*   | target table, overriding the one declared in the file            |
| `--compress`         | `false`   | force gzip decompression (otherwise it is auto-detected by the magic bytes) |
| `--continue-on-error` | `false`   | skip the rows that fail to apply instead of aborting             |

### `run`

```
data-sync run --file <file> [flags]
```

Executes the SQL statements of a plain text file. The file is read line by
line: all the lines up to a line ending with `;` form a single statement (the
trailing `;` is stripped). Statements are executed one by one and the command
succeeds only when every statement has been executed without errors.

| Flag      | Default        | Description                   |
| --------- | -------------- | ----------------------------- |
| `--file`  | *(required)*   | input SQL file (`-` for stdin) |

### `migrate`

```
data-sync migrate --db <database> [flags]
```

Applies the pending migration files of a database, so the schema and the seed
data can be deployed incrementally. It requires `--db`.

| Flag     | Default   | Description                                                  |
| -------- | --------- | ------------------------------------------------------------ |
| `--dir`  | `db`      | base folder of the migration files                           |

On the first run it creates a `migration` table (single `last_version INT`
column, single record) in the target database. It then scans the
`<dir>/<DB_NAME>` folder for files named `<version>_<name>.sql` or
`<version>_<name>.ddf`, sorted alphabetically, and applies every file whose
`<version>` is greater than the value stored in the `migration` table:

- `.sql` files are executed with the same logic as the `run` command
- `.ddf` files are imported with the same logic as the `import` command

After a file is applied successfully its version is recorded in the
`migration` table, inside a transaction. Any other file extension, or a file
whose name does not start with a numeric version, aborts the run.

## Examples

Export the whole `broker_product` table into a compressed DDF file:

```sh
data-sync export --table collector.broker_product --file backup.ddf.gz \
    --user admin --password admin --db collector
```

Export a custom query to standard output:

```sh
data-sync export --query "SELECT id, symbol FROM broker_product WHERE active = 1" \
    --user admin --password admin --db collector --file -
```

Import a DDF file, skipping the rows that fail:

```sh
data-sync import --file backup.ddf.gz --db collector \
    --user admin --password admin --continue-on-error
```

Import the same file into another table:

```sh
data-sync import --file backup.ddf.gz --table broker_product_archive \
    --db collector --user admin --password admin
```

Run an SQL script:

```sh
data-sync run --file schema.sql --db collector --user admin --password admin
```

Migrate a database folder:

```sh
# db/collector/ contains e.g. 001_create_table.sql, 002_seed.ddf, ...
data-sync migrate --db collector --dir db \
    --user admin --password admin
```

## Working with PostgreSQL

All the commands support PostgreSQL by passing `--dbtype postgres` (the port
defaults to `5432`):

```sh
data-sync export --table datastore --dbtype postgres --db mydb \
    --user admin --password admin --file datastore.ddf
```

## Library

The `pkg/ddf` package exposes the DDF reader, writer, importer and exporter as
an importable Go library (`github.com/algotiqa/data-sync/pkg/ddf`), so the
format can be used by other tools as well.