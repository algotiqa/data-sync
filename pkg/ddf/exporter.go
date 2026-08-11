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

package ddf

import (
	"compress/gzip"
	"context"
	"database/sql"
	"io"
	"os"
	"strings"
)

//=============================================================================
// ExportConfig carries the options of an export operation.
//============================================================================

type ExportConfig struct {
	DB             *sql.DB
	IsPostgres     bool
	DBName         string // used as the default schema on MySQL
	Query          string
	FileName       string // "-" or empty means standard output
	UseCompression bool
	Listener       ExportListener
}

//============================================================================
// Export runs the given query and writes the resulting rows to a DDF file.
// It returns the number of exported rows.
//============================================================================

func Export(ctx context.Context, cfg ExportConfig) (int64, error) {
	rows, err := cfg.DB.QueryContext(ctx, cfg.Query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	cols, err := rows.ColumnTypes()
	if err != nil {
		return 0, err
	}

	fields := GetQueryFields(cols)

	table, schema := splitTable(GetTable(cfg.Query))

	if table != "" {
		schemaForPK := schema
		if schemaForPK == "" {
			if cfg.IsPostgres {
				schemaForPK = "public"
			} else {
				schemaForPK = cfg.DBName
			}
		}

		keys, err := GetPrimaryKeys(cfg.DB, cfg.IsPostgres, schemaForPK, table)
		if err != nil {
			return 0, err
		}

		for i := range fields {
			if keys[fields[i].Name] {
				fields[i].IsKey = true
			}
		}
	}

	out, closeOutput, err := openOutput(cfg.FileName, cfg.UseCompression)
	if err != nil {
		return 0, err
	}
	defer closeOutput()

	writer := NewWriter(out, false)
	writer.SetQuery(cfg.Query)
	writer.SetTable(table)
	writer.SetSchema(schema)
	writer.SetFields(fields)

	if err := writer.BeginInsertSection(); err != nil {
		return 0, err
	}

	var recordNum int64 = 1

	for rows.Next() {
		row, err := GetRow(rows, fields)
		if err != nil {
			return 0, err
		}

		if err := writer.WriteRow(row); err != nil {
			return 0, err
		}

		if cfg.Listener != nil {
			cfg.Listener.ExportedRow(row, recordNum)
		}

		recordNum++
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	return recordNum - 1, writer.End()
}

//=============================================================================
// splitTable splits a schema-qualified table name into its parts.
//============================================================================

func splitTable(table string) (name string, schema string) {
	if idx := strings.Index(table, "."); idx != -1 {
		return table[idx+1:], table[:idx]
	}

	return table, ""
}

//============================================================================
// openOutput opens the export destination, wrapping it with a gzip writer when
// requested. The returned function closes all opened resources.
//============================================================================

func openOutput(fileName string, useCompression bool) (io.Writer, func(), error) {
	var out io.Writer = os.Stdout
	var f *os.File
	var gz *gzip.Writer

	if fileName != "" && fileName != "-" {
		var err error

		f, err = os.Create(fileName)
		if err != nil {
			return nil, nil, err
		}
		out = f
	}

	if useCompression {
		gz = gzip.NewWriter(out)
		out = gz
	}

	closeOutput := func() {
		if gz != nil {
			_=gz.Close()
		}
		if f != nil {
			_=f.Close()
		}
	}

	return out, closeOutput, nil
}

//=============================================================================
