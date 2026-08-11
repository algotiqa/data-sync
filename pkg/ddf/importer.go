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
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
)

//=============================================================================
// ImportConfig carries the options of an import operation.
//=============================================================================

type ImportConfig struct {
	DB         *sql.DB
	IsPostgres bool
	Table      string // optional, overrides the table declared in [INFO]
	Listener   ImportListener
}

//=============================================================================
// Import reads a DDF stream and applies its records to the configured
// database. It returns the number of records applied.
//=============================================================================

func Import(cfg ImportConfig, r io.Reader, useCompression bool) (int64, error) {
	im := &Importer{
		ctx       : context.Background(),
		db        : cfg.DB,
		isPostgres: cfg.IsPostgres,
		table     : cfg.Table,
		listener  : cfg.Listener,
	}
	defer im.closeStatements()

	rdr, err := NewReader(r, useCompression)
	if err != nil {
		return 0, err
	}

	if err = rdr.Read(im); err != nil {
		return 0, err
	}

	return im.insertedCount + im.updatedCount + im.deletedCount, nil
}

//=============================================================================
// Importer applies the rows of a DDF file to a database table. INSERT, UPDATE
// and DELETE statements are created lazily and reused for every row.
//=============================================================================

type Importer struct {
	ctx        context.Context
	db         *sql.DB
	isPostgres bool
	table      string
	listener   ImportListener

	stmtInsert *sql.Stmt
	stmtUpdate *sql.Stmt
	stmtDelete *sql.Stmt

	fields       []QueryField
	skipAll      bool
	missingPKeys bool

	insertedCount int64
	updatedCount  int64
	deletedCount  int64
}

//=============================================================================

func (im *Importer) HandleInfo(info Info) {
	if im.table == "" {
		im.table = info.Table
	}
}

//=============================================================================

func (im *Importer) HandleQueryFields(fields []QueryField) {
	im.fields = fields
}

//=============================================================================

func (im *Importer) HandleRow(op Operation, row []any, line string, recordNum int64) (bool, error) {
	switch op {
		case OpInsert:
			return im.insertRow(row, line, recordNum)
		case OpUpdate:
			return im.updateRow(row, line, recordNum)
		case OpDelete:
			return im.deleteRow(row, line, recordNum)
	}

	return false, errors.New("unknown operation: " + string(op))
}

//=============================================================================

func (im *Importer) HandlePostRow(op Operation, line string, recordNum int64) {
	if im.listener != nil {
		im.listener.PostRow(op, recordNum, line)
	}
}

//=============================================================================
// Insert
//=============================================================================

func (im *Importer) insertRow(row []any, line string, recordNum int64) (bool, error) {
	if im.stmtInsert == nil {
		stmt, err := im.createInsertStatement()
		if err != nil {
			return false, err
		}
		im.stmtInsert = stmt
	}

	if im.listener != nil {
		im.listener.InsertRow(im.fields, row, recordNum)
	}

	ok, err := im.applyRow(im.stmtInsert, row, OpInsert, line, recordNum)
	if ok {
		im.insertedCount++
	}

	return ok, err
}

//=============================================================================

func (im *Importer) createInsertStatement() (*sql.Stmt, error) {
	if im.table == "" {
		return nil, errors.New("missing table name")
	}

	return im.db.PrepareContext(im.ctx, insertQuery(im.table, im.fields, im.isPostgres))
}

//=============================================================================

func insertQuery(table string, fields []QueryField, isPostgres bool) string {
	names := make([]string, len(fields))

	for i, f := range fields {
		names[i] = f.Name
	}

	return "INSERT INTO " + table + "(" + strings.Join(names, ",") +
		") VALUES (" + Placeholders(len(fields), isPostgres) + ")"
}

//=============================================================================
// Update
//=============================================================================

func (im *Importer) updateRow(row []any, line string, recordNum int64) (bool, error) {
	if im.stmtUpdate == nil {
		stmt, err := im.createUpdateStatement()
		if err != nil {
			return false, err
		}
		im.stmtUpdate = stmt
	}

	// Fields and values must be reorganized: first the non-key fields, then
	// the key fields, following the order of the placeholders in the statement

	var valueRow, whereRow []any

	for i, f := range im.fields {
		if f.IsKey {
			whereRow = append(whereRow, row[i])
		} else {
			valueRow = append(valueRow, row[i])
		}
	}

	args := append(valueRow, whereRow...)

	if im.listener != nil {
		im.listener.UpdateRow(im.fields, row, recordNum)
	}

	ok, err := im.applyRow(im.stmtUpdate, args, OpUpdate, line, recordNum)
	if ok {
		im.updatedCount++
	}

	return ok, err
}

//=============================================================================

func (im *Importer) createUpdateStatement() (*sql.Stmt, error) {
	if im.table == "" {
		return nil, errors.New("missing table name")
	}

	query, err := updateQuery(im.table, im.fields, im.isPostgres)
	if err != nil {
		return nil, err
	}

	return im.db.PrepareContext(im.ctx, query)
}

//=============================================================================
// updateQuery builds the UPDATE statement. Non-key fields are assigned first
// (they get the lowest placeholder numbers), then the key fields appear in the WHERE clause.
//=============================================================================

func updateQuery(table string, fields []QueryField, isPostgres bool) (string, error) {
	var values, where []string
	n := 0

	for _, f := range fields {
		if f.IsKey {
			continue
		}
		n++
		values = append(values, f.Name+" = "+Placeholder(n, isPostgres))
	}

	for _, f := range fields {
		if !f.IsKey {
			continue
		}
		n++
		where = append(where, f.Name+" = "+Placeholder(n, isPostgres))
	}

	if len(where) == 0 {
		return "", errors.New("missing primary keys for update operation")
	}
	if len(values) == 0 {
		return "", errors.New("missing values for update operation")
	}

	return "UPDATE " + table + " SET " + strings.Join(values, ",") +
		" WHERE " + strings.Join(where, " AND "), nil
}

//=============================================================================
// Delete
//=============================================================================

func (im *Importer) deleteRow(row []any, line string, recordNum int64) (bool, error) {
	if im.stmtDelete == nil {
		stmt, err := im.createDeleteStatement()
		if err != nil {
			return false, err
		}
		im.stmtDelete = stmt
	}

	args := row

	if !im.missingPKeys {
		var pkeyRow []any

		for i, f := range im.fields {
			if f.IsKey {
				pkeyRow = append(pkeyRow, row[i])
			}
		}
		args = pkeyRow
	}

	if im.listener != nil {
		im.listener.DeleteRow(im.fields, row, recordNum)
	}

	ok, err := im.applyRow(im.stmtDelete, args, OpDelete, line, recordNum)
	if ok {
		im.deletedCount++
	}

	return ok, err
}

//=============================================================================

func (im *Importer) createDeleteStatement() (*sql.Stmt, error) {
	if im.table == "" {
		return nil, errors.New("missing table name")
	}

	query, missingPKeys := deleteQuery(im.table, im.fields, im.isPostgres)
	im.missingPKeys = missingPKeys

	return im.db.PrepareContext(im.ctx, query)
}

//=============================================================================
// deleteQuery builds the DELETE statement. When the table has no primary key,
// the delete targets all the row fields and missingPKeys is returned true.
//=============================================================================

func deleteQuery(table string, fields []QueryField, isPostgres bool) (string, bool) {
	var where []string
	n := 0
	missingPKeys := false

	for _, f := range fields {
		if !f.IsKey {
			continue
		}
		n++
		where = append(where, f.Name+" = "+Placeholder(n, isPostgres))
	}

	if len(where) == 0 {
		// No primary keys: fall back to a delete on all the row fields

		missingPKeys = true

		for _, f := range fields {
			n++
			where = append(where, f.Name+" = "+Placeholder(n, isPostgres))
		}
	}

	return "DELETE FROM " + table + " WHERE " + strings.Join(where, " AND "), missingPKeys
}

//=============================================================================

func (im *Importer) applyRow(stmt *sql.Stmt, args []any, op Operation, line string, recordNum int64) (bool, error) {
	for {
		_, err := stmt.ExecContext(im.ctx, args...)

		if err == nil {
			return true, nil
		}
		if im.listener == nil {
			return false, err
		}
		if im.skipAll {
			return true, nil
		}

		switch im.listener.OnError(op, err, recordNum, line) {
			case ActionSkip:
				return true, nil
			case ActionSkipAll:
				im.skipAll = true
				return true, nil
			default: // ActionAbort
				return false, fmt.Errorf("import aborted: record %d failed: %w", recordNum, err)
		}
	}
}

//=============================================================================

func (im *Importer) closeStatements() {
	if im.stmtInsert != nil {
		_=im.stmtInsert.Close()
	}
	if im.stmtUpdate != nil {
		_=im.stmtUpdate.Close()
	}
	if im.stmtDelete != nil {
		_=im.stmtDelete.Close()
	}
}

//=============================================================================
