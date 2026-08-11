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

package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/algotiqa/data-sync/pkg/ddf"
	"github.com/spf13/cobra"
)

//=============================================================================

type exportOptions struct {
	connOptions
	query    string
	table    string
	file     string
	compress bool
}

func newExportCmd() *cobra.Command {
	o := &exportOptions{}

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export the data of a table or query into a DDF file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(o)
		},
	}

	addConnFlags(cmd, &o.connOptions)
	cmd.Flags().StringVar(&o.query, "query", "", "query to export (alternative to --table)")
	cmd.Flags().StringVar(&o.table, "table", "", "table to export (alternative to --query)")
	cmd.Flags().StringVar(&o.file, "file", "", "output DDF file ('-' for stdout, default '<table>.ddf')")
	cmd.Flags().BoolVar(&o.compress, "compress", false, "compress the output with gzip")

	return cmd
}

//=============================================================================

func runExport(o *exportOptions) error {
	query := o.query

	if query == "" {
		if o.table == "" {
			return errors.New("either --query or --table is required")
		}
		query = "SELECT * FROM " + o.table
	}

	file := o.file

	if file == "" {
		if o.table != "" {
			file = o.table + ".ddf"
		} else {
			file = "export.ddf"
		}
	}

	useCompression := o.compress || strings.HasSuffix(file, ".gz")
	o.resolvePort()

	db, err := connect(&o.connOptions)
	if err != nil {
		return err
	}
	defer db.Close()

	count, err := ddf.Export(context.Background(), ddf.ExportConfig{
		DB:             db,
		IsPostgres:     o.dbType == "postgres",
		DBName:         o.database,
		Query:          query,
		FileName:       file,
		UseCompression: useCompression,
		Listener:       &exportListener{},
	})
	if err != nil {
		return err
	}

	slog.Info("Export completed", "file", file, "rows", count, "compressed", useCompression)
	return nil
}

//=============================================================================

type exportListener struct{}

//=============================================================================

func (l *exportListener) ExportedRow(row []any, recordNum int64) {
	if recordNum%10000 == 0 {
		slog.Info("Rows exported", "count", recordNum)
	}
}

//=============================================================================
