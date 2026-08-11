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
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"log/slog"
	"os"

	"github.com/algotiqa/data-sync/pkg/ddf"
	"github.com/spf13/cobra"
)

//=============================================================================

type importOptions struct {
	connOptions
	file            string
	table           string
	compress        bool
	continueOnError bool
}

//=============================================================================

func newImportCmd() *cobra.Command {
	o := &importOptions{}

	cmd := &cobra.Command{
		Use  : "import",
		Short: "Import the data of a DDF file into a database table",
		RunE : func(cmd *cobra.Command, args []string) error {
			return runImport(o)
		},
	}

	addConnFlags(cmd, &o.connOptions)
	cmd.Flags().StringVar(&o.file,            "file",              "",    "input DDF file ('-' for stdin)")
	cmd.Flags().StringVar(&o.table,           "table",             "",    "target table (default: the table declared in the file)")
	cmd.Flags().BoolVar  (&o.compress,        "compress",          false, "decompress the input with gzip")
	cmd.Flags().BoolVar  (&o.continueOnError, "continue-on-error", false, "skip the rows that fail to apply")

	return cmd
}

//=============================================================================

func runImport(o *importOptions) error {
	if o.file == "" {
		return errors.New("--file is required")
	}

	o.resolvePort()

	db, err := connect(&o.connOptions)
	if err != nil {
		return err
	}
	defer db.Close()

	r, closeInput, err := openInput(o.file, o.compress)
	if err != nil {
		return err
	}
	defer closeInput()

	listener := &importListener{continueOnError: o.continueOnError}

	count, err := ddf.Import(ddf.ImportConfig{
		DB:         db,
		IsPostgres: o.dbType == "postgres",
		Table:      o.table,
		Listener:   listener,
	}, r, false)
	if err != nil {
		return err
	}

	slog.Info("Import completed", "file", o.file, "rows", count)
	return nil
}

//=============================================================================
// openInput opens the import source and transparently wraps it with a gzip
// reader when the stream is compressed. Compression is detected by the gzip
// magic bytes unless forceCompression is set.
//=============================================================================

func openInput(file string, forceCompression bool) (io.Reader, func(), error) {
	var r io.Reader
	var f *os.File

	if file == "-" {
		r = os.Stdin
	} else {
		var err error

		f, err = os.Open(file)
		if err != nil {
			return nil, nil, err
		}
		r = f
	}

	br := bufio.NewReader(r)

	useCompression := forceCompression
	if !useCompression {
		if magic, err := br.Peek(2); err == nil && magic[0] == 0x1f && magic[1] == 0x8b {
			useCompression = true
		}
	}

	var gz *gzip.Reader

	if useCompression {
		var err error

		gz, err = gzip.NewReader(br)
		if err != nil {
			if f != nil {
				_=f.Close()
			}
			return nil, nil, err
		}
		r = gz
	} else {
		r = br
	}

	closeInput := func() {
		if gz != nil {
			_=gz.Close()
		}
		if f != nil {
			_=f.Close()
		}
	}

	return r, closeInput, nil
}

//=============================================================================

type importListener struct {
	continueOnError bool
}

//=============================================================================

func (l *importListener) InsertRow(fields []ddf.QueryField, row []any, recordNum int64) {
	l.progress(ddf.OpInsert, recordNum)
}

//=============================================================================

func (l *importListener) UpdateRow(fields []ddf.QueryField, row []any, recordNum int64) {
	l.progress(ddf.OpUpdate, recordNum)
}

//=============================================================================

func (l *importListener) DeleteRow(fields []ddf.QueryField, row []any, recordNum int64) {
	l.progress(ddf.OpDelete, recordNum)
}

//=============================================================================

func (l *importListener) progress(op ddf.Operation, recordNum int64) {
	if recordNum%10000 == 0 {
		slog.Info("Rows processed", "op", string(op), "count", recordNum)
	}
}

//=============================================================================

func (l *importListener) OnError(op ddf.Operation, err error, recordNum int64, line string) ddf.ErrorAction {
	slog.Error("Row failed", "op", string(op), "record", recordNum, "error", err)
	slog.Debug("Row content", "line", line)

	if l.continueOnError {
		return ddf.ActionSkip
	}

	return ddf.ActionAbort
}

//=============================================================================

func (l *importListener) PostRow(op ddf.Operation, recordNum int64, line string) {
}

//=============================================================================
