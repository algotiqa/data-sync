//=============================================================================
// MIT License
//
// Copyright (c) 2026 Algotiqa
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.
//=============================================================================

package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/algotiqa/data-sync/pkg/command"
	import_ "github.com/algotiqa/data-sync/pkg/command/import"
	"github.com/algotiqa/data-sync/pkg/command/run"
	"github.com/algotiqa/data-sync/pkg/ddf"
	"github.com/spf13/cobra"
)

//=============================================================================

type migrateOptions struct {
	command.ConnOptions
	dir string
}

//=============================================================================

func NewMigrateCmd() *cobra.Command {
	o := &migrateOptions{}

	cmd := &cobra.Command{
		Use  : "migrate",
		Short: "Apply the pending migration files of a database",
		RunE : func(cmd *cobra.Command, args []string) error {
			return runMigrate(o)
		},
	}

	command.AddConnFlags(cmd, &o.ConnOptions)
	cmd.Flags().StringVar(&o.dir, "dir", "db", "base folder of the migration files")

	return cmd
}

//=============================================================================

func runMigrate(o *migrateOptions) error {
	if o.Database == "" {
		return errors.New("--db is required")
	}

	o.ResolvePort()

	db, err := command.Connect(&o.ConnOptions)
	if err != nil {
		return err
	}
	defer db.Close()

	isPostgres := o.DbType == "postgres"

	lastVersion, err := ensureMigrationTable(db)
	if err != nil {
		return err
	}

	folder := filepath.Join(o.dir, o.Database)

	entries, err := os.ReadDir(folder)
	if err != nil {
		return err
	}

	applied := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version, kind, err := parseMigrationFile(entry.Name())
		if err != nil {
			return err
		}

		if version <= lastVersion {
			continue
		}

		path := filepath.Join(folder, entry.Name())

		if err = applyMigration(path, kind, db, isPostgres); err != nil {
			return fmt.Errorf("migration %s failed: %w", entry.Name(), err)
		}

		if err = setLastVersion(db, isPostgres, version); err != nil {
			return fmt.Errorf("migration %s failed to record version %d: %w", entry.Name(), version, err)
		}

		lastVersion = version
		applied++

		slog.Info("Migration applied", "file", entry.Name(), "version", version)
	}

	slog.Info("Migrate completed", "folder", folder, "applied", applied, "last_version", lastVersion)
	return nil
}

//=============================================================================
// ensureMigrationTable creates the migration table when missing and guarantees
// it holds exactly one record, returning the last executed version.
//=============================================================================

func ensureMigrationTable(db *sql.DB) (int, error) {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS migration (last_version INT NOT NULL)"); err != nil {
		return 0, err
	}

	var count int

	if err := db.QueryRow("SELECT COUNT(*) FROM migration").Scan(&count); err != nil {
		return 0, err
	}

	if count == 0 {
		if _, err := db.Exec("INSERT INTO migration (last_version) VALUES (-1)"); err != nil {
			return 0, err
		}

		return -1, nil
	}

	var lastVersion int

	if err := db.QueryRow("SELECT last_version FROM migration").Scan(&lastVersion); err != nil {
		return 0, err
	}

	return lastVersion, nil
}

//=============================================================================
// parseMigrationFile extracts the version and the type from a migration file
// name of the form <version>_<name>.sql or <version>_<name>.ddf.
//=============================================================================

func parseMigrationFile(name string) (version int, kind string, err error) {
	ext := filepath.Ext(name)

	if ext != ".sql" && ext != ".ddf" {
		return 0, "", fmt.Errorf("unsupported migration file %q (must be .sql or .ddf)", name)
	}

	base := strings.TrimSuffix(name, ext)

	idx := strings.Index(base, "_")
	if idx <= 0 {
		return 0, "", fmt.Errorf("invalid migration file %q (expected <version>_<name>%s)", name, ext)
	}

	version, err = strconv.Atoi(base[:idx])
	if err != nil {
		return 0, "", fmt.Errorf("invalid migration version in file %q: %v", name, err)
	}

	return version, ext, nil
}

//=============================================================================
// setLastVersion records the last executed version inside a transaction.
//=============================================================================

func setLastVersion(db *sql.DB, isPostgres bool, version int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := "UPDATE migration SET last_version = " + ddf.Placeholder(1, isPostgres)

	if _, err = tx.Exec(query, version); err != nil {
		return err
	}

	return tx.Commit()
}

//=============================================================================

func applyMigration(path, kind string, db *sql.DB, isPostgres bool) error {
	switch kind {
		case ".sql":
		return run.ExecuteSQLFile(db, path)
		case ".ddf":
		return import_.ExecuteDDFFile(db, isPostgres, path)
	}

	return nil
}

//=============================================================================