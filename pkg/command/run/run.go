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

package run

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/algotiqa/data-sync/pkg/command"
	"github.com/spf13/cobra"
)

//=============================================================================

type runOptions struct {
	command.ConnOptions
	file string
}

//=============================================================================

func NewRunCmd() *cobra.Command {
	o := &runOptions{}

	cmd := &cobra.Command{
		Use  : "run",
		Short: "Run the SQL statements contained in a file",
		RunE : func(cmd *cobra.Command, args []string) error {
			return runRun(o)
		},
	}

	command.AddConnFlags(cmd, &o.ConnOptions)
	cmd.Flags().StringVar(&o.file, "file", "", "input SQL file ('-' for stdin)")

	return cmd
}

//=============================================================================

func runRun(o *runOptions) error {
	if o.file == "" {
		return errors.New("--file is required")
	}

	o.ResolvePort()

	db, err := command.Connect(&o.ConnOptions)
	if err != nil {
		return err
	}
	defer db.Close()

	r, closeInput, err := openScript(o.file)
	if err != nil {
		return err
	}
	defer closeInput()

	statements, err := parseStatements(r)
	if err != nil {
		return err
	}

	if len(statements) == 0 {
		slog.Warn("No SQL statements found", "file", o.file)
		return nil
	}

	if err := ExecuteStatements(db, statements); err != nil {
		return err
	}

	slog.Info("Run completed", "file", o.file, "statements", len(statements))
	return nil
}

//=============================================================================
// ExecuteSQLFile parses the given SQL file and executes its statements,
// aborting on the first failing one.
//=============================================================================

func ExecuteSQLFile(db *sql.DB, file string) error {
	r, closeInput, err := openScript(file)
	if err != nil {
		return err
	}
	defer closeInput()

	statements, err := parseStatements(r)
	if err != nil {
		return err
	}

	return ExecuteStatements(db, statements)
}

//=============================================================================
// ExecuteStatements executes the given SQL statements one by one, aborting on
// the first failing one.
//=============================================================================

func ExecuteStatements(db *sql.DB, statements []string) error {
	for i, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("statement %d of %d failed: %w", i+1, len(statements), err)
		}
	}

	return nil
}

//=============================================================================
// openScript opens the input SQL file, or returns standard input for '-'.
//=============================================================================

func openScript(file string) (io.Reader, func(), error) {
	if file == "-" {
		return os.Stdin, func() {}, nil
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, nil, err
	}

	return f, func() { _ = f.Close() }, nil
}

//=============================================================================
// parseStatements reads an SQL script line by line, grouping the lines into
// statements that end with a semicolon. The trailing semicolon is stripped
// from the returned statements and empty statements are skipped.
//=============================================================================

func parseStatements(r io.Reader) ([]string, error) {
	br := bufio.NewReader(r)

	var statements []string
	var pending []string

	for {
		line, err := br.ReadString('\n')

		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			trimmed := strings.TrimRight(line, " \t")

			if strings.HasPrefix(trimmed, "--") {
				continue
			}

			if strings.HasSuffix(trimmed, ";") {
				pending = append(pending, strings.TrimSuffix(trimmed, ";"))

				statement := strings.TrimSpace(strings.TrimRight(strings.Join(pending, "\n"), ";"))
				pending = nil

				if statement != "" {
					statements = append(statements, statement)
				}
			} else if strings.TrimSpace(line) != "" {
				pending = append(pending, line)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	if len(pending) > 0 {
		return nil, errors.New("unterminated SQL statement (missing ';')")
	}

	return statements, nil
}

//=============================================================================