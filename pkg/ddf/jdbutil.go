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
	"database/sql"
	"fmt"
	"strings"
)

//=============================================================================
// GetQueryFields builds the list of QueryField from the column metadata of a
// result set, mapping each driver-specific type name into a canonical type.
//============================================================================

func GetQueryFields(cols []*sql.ColumnType) []QueryField {
	fields := make([]QueryField, len(cols))

	for i, col := range cols {
		hasDecimals := false
		if _, scale, ok := col.DecimalSize(); ok && scale != 0 {
			hasDecimals = true
		}

		fields[i] = QueryField{
			Name: col.Name(),
			Type: MapDBType(col.DatabaseTypeName(), hasDecimals),
		}
	}

	return fields
}

//============================================================================
// GetRow scans the current row of the given result set and converts every
// column into the value representation used while writing a DDF file.
//============================================================================

func GetRow(rows *sql.Rows, fields []QueryField) ([]any, error) {
	dest := make([]any, len(fields))

	for i, f := range fields {
		t := f.Type

		switch {
		case t.IsTimestamp() || t.IsDate():
			dest[i] = new(sql.NullTime)
		case t.IsTime():
			dest[i] = new(sql.NullString)
		case t.IsBinaryType() || t.IsBlob():
			dest[i] = new([]byte)
		case t.IsBoolean():
			dest[i] = new(sql.NullBool)
		default:
			dest[i] = new(sql.NullString)
		}
	}

	if err := rows.Scan(dest...); err != nil {
		return nil, err
	}

	row := make([]any, len(fields))

	for i, f := range fields {
		t := f.Type

		switch {
		case t.IsTimestamp():
			if v := dest[i].(*sql.NullTime); v.Valid {
				row[i] = v.Time
			}
		case t.IsDate():
			if v := dest[i].(*sql.NullTime); v.Valid {
				row[i] = v.Time.Format(DateLayout)
			}
		case t.IsTime():
			if v := dest[i].(*sql.NullString); v.Valid {
				row[i] = normalizeTime(v.String)
			}
		case t.IsBinaryType() || t.IsBlob():
			row[i] = *dest[i].(*[]byte)
		case t.IsBoolean():
			if v := dest[i].(*sql.NullBool); v.Valid {
				if v.Bool {
					row[i] = "T"
				} else {
					row[i] = "F"
				}
			}
		default:
			if v := dest[i].(*sql.NullString); v.Valid {
				if t.ID == unknownID {
					row[i] = "< ???? >"
				} else {
					row[i] = v.String
				}
			}
		}
	}

	return row, nil
}

//============================================================================
// GetTable extracts the (possibly schema-qualified) table name from a SELECT
// query. It returns an empty string when the query targets multiple tables or
// is not a simple SELECT.
//============================================================================

func GetTable(query string) string {
	q := strings.ToLower(query)
	q = strings.ReplaceAll(q, "\n", " ")
	q = strings.ReplaceAll(q, "\r", " ")
	q = strings.ReplaceAll(q, "\t", " ")

	idx1 := strings.Index(q, " from ")
	if idx1 == -1 {
		return ""
	}

	idx2 := strings.Index(q, " where ")
	if idx2 == -1 {
		idx2 = len(query)
	}

	table := strings.TrimSpace(query[idx1+6 : idx2])

	if idx1 = strings.Index(table, " "); idx1 != -1 {
		table = table[:idx1]
	}

	if strings.Contains(table, ",") {
		return ""
	}

	return table
}

//============================================================================
// GetPrimaryKeys returns the set of primary key column names of a table. When
// schema is empty the current database (MySQL) or the public schema (PostgreSQL) is used.
//============================================================================

func GetPrimaryKeys(db *sql.DB, isPostgres bool, schema, table string) (map[string]bool, error) {
	var query string
	var args []any

	if isPostgres {
		if schema == "" {
			schema = "public"
		}

		query = `
			SELECT kcu.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
			  ON tc.constraint_name = kcu.constraint_name
			 AND tc.table_schema = kcu.table_schema
			WHERE tc.constraint_type = 'PRIMARY KEY'
			  AND tc.table_schema = $1
			  AND tc.table_name = $2
			ORDER BY kcu.ordinal_position`
		args = []any{schema, table}
	} else {
		// An empty schema means the current database.
		schemaExpr := "DATABASE()"
		if schema != "" {
			schemaExpr = "?"
			args = []any{schema, table}
		} else {
			args = []any{table}
		}

		query = `
			SELECT COLUMN_NAME
			FROM information_schema.KEY_COLUMN_USAGE
			WHERE TABLE_SCHEMA = ` + schemaExpr + `
			  AND TABLE_NAME = ?
			  AND CONSTRAINT_NAME = 'PRIMARY'
			ORDER BY ORDINAL_POSITION`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make(map[string]bool)

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		keys[name] = true
	}

	return keys, rows.Err()
}

//=============================================================================
// Placeholder returns the parameter placeholder for the n-th (1 based)
// bind argument, honoring the SQL dialect of the connected database.
//=============================================================================

func Placeholder(n int, isPostgres bool) string {
	if isPostgres {
		return fmt.Sprintf("$%d", n)
	}

	return "?"
}

//=============================================================================
// Placeholders returns the comma separated list of n parameter placeholders.
//=============================================================================

func Placeholders(count int, isPostgres bool) string {
	parts := make([]string, count)

	for i := range parts {
		parts[i] = Placeholder(i+1, isPostgres)
	}

	return strings.Join(parts, ",")
}

//=============================================================================

func normalizeTime(v string) string {
	if len(v) > len(TimeLayout) {
		return v[:len(TimeLayout)]
	}

	return v
}

//=============================================================================
