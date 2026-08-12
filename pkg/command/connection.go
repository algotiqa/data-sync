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

package command

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/cobra"
)

//=============================================================================
// connOptions holds the flags needed to connect to a database.
//=============================================================================

type ConnOptions struct {
	DbType   string
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Dsn      string
}

//=============================================================================
// resolvePort fills the default port when it was not specified.
//=============================================================================

func (o *ConnOptions) ResolvePort() {
	if o.Port != 0 {
		return
	}

	if o.DbType == "postgres" {
		o.Port = 5432
	} else {
		o.Port = 3306
	}
}

//=============================================================================
// addConnFlags registers the connection flags on a command. Every flag can be
// overridden by the corresponding DB_* environment variable; the value of a
// flag takes precedence over the environment one.
//=============================================================================

func AddConnFlags(cmd *cobra.Command, o *ConnOptions) {
	port := 0
	if v := os.Getenv("DB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}

	cmd.Flags().StringVar(&o.DbType,   "dbtype",   envOrDefault("DB_TYPE", "mysql"), "database type: mysql, postgres")
	cmd.Flags().StringVar(&o.Host,     "host",     envOrDefault("DB_HOST", "127.0.0.1"), "database host")
	cmd.Flags().IntVar   (&o.Port,     "port",     port, "database port (default depends on --dbtype)")
	cmd.Flags().StringVar(&o.User,     "user",     envOrDefault("DB_USER", "root"), "database user")
	cmd.Flags().StringVar(&o.Password, "password", envOrDefault("DB_PASSWORD", ""), "database password")
	cmd.Flags().StringVar(&o.Database, "db",       envOrDefault("DB_NAME", ""), "database name")
	cmd.Flags().StringVar(&o.Dsn,      "dsn",      os.Getenv("DB_DSN"), "full connection string (overrides the single flags)")
}

//=============================================================================
// envOrDefault returns the value of the given environment variable or fallback
// when it is empty.
//=============================================================================

func envOrDefault(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}

	return fallback
}

//=============================================================================
// connect opens and pings a database connection matching the given options.
//=============================================================================

func Connect(o *ConnOptions) (*sql.DB, error) {
	var driver, dsn string

	switch o.DbType {
		case "mysql":
			driver = "mysql"

			if o.Dsn == "" {
				dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=UTC", o.User, o.Password, o.Host, o.Port, o.Database)
			} else {
				dsn = o.Dsn
			}

		case "postgres":
			driver = "pgx"

			if o.Dsn == "" {
				dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", o.User, o.Password, o.Host, o.Port, o.Database)
			} else {
				dsn = o.Dsn
			}

		default:
			return nil, fmt.Errorf("unsupported database type %q (must be 'mysql' or 'postgres')", o.DbType)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		_=db.Close()
		return nil, err
	}

	db.SetMaxOpenConns(5)

	return db, nil
}

//=============================================================================
