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

import "testing"

//=============================================================================

func TestMapName(t *testing.T) {
	if got := MapName("BIGINT"); got.ID != JdbcBigInt {
		t.Errorf("MapName(BIGINT).ID = %d, want %d", got.ID, JdbcBigInt)
	}

	if got := MapName("NOPE"); got.Name != "(unknown)" {
		t.Errorf("MapName(NOPE).Name = %q, want (unknown)", got.Name)
	}
}

//============================================================================

func TestMapDBTypeMySQL(t *testing.T) {
	cases := []struct {
		dbType      string
		hasDecimals bool
		wantName    string
	}{
		{"bigint", false, "BIGINT"},
		{"int", false, "INTEGER"},
		{"varchar", false, "VARCHAR"},
		{"text", false, "LONGVARCHAR"},
		{"longtext", false, "LONGVARCHAR"},
		{"blob", false, "BLOB"},
		{"datetime", false, "TIMESTAMP"},
		{"date", false, "DATE"},
		{"time", false, "TIME"},
		{"boolean", false, "BOOLEAN"},
		{"decimal", true, "DECIMAL"},
		{"decimal", false, "INTEGER"}, // scale 0 -> INTEGER
		{"json", false, "LONGVARCHAR"},
	}

	for _, c := range cases {
		got := MapDBType(c.dbType, c.hasDecimals)

		if got.Name != c.wantName {
			t.Errorf("MapDBType(%q, %v).Name = %q, want %q", c.dbType, c.hasDecimals, got.Name, c.wantName)
		}
	}
}

//============================================================================

func TestMapDBTypePostgres(t *testing.T) {
	cases := []struct {
		dbType      string
		hasDecimals bool
		wantName    string
	}{
		{"int8", false, "BIGINT"},
		{"int4", false, "INTEGER"},
		{"int2", false, "SMALLINT"},
		{"float8", false, "DOUBLE"},
		{"numeric", true, "NUMERIC"},
		{"numeric", false, "INTEGER"},
		{"text", false, "LONGVARCHAR"},
		{"bytea", false, "BLOB"},
		{"timestamptz", false, "TIMESTAMP"},
		{"timetz", false, "TIME"},
		{"bool", false, "BOOLEAN"},
	}

	for _, c := range cases {
		got := MapDBType(c.dbType, c.hasDecimals)

		if got.Name != c.wantName {
			t.Errorf("MapDBType(%s, %v).Name = %q, want %q", c.dbType, c.hasDecimals, got.Name, c.wantName)
		}
	}
}

//============================================================================

func TestGetTable(t *testing.T) {
	cases := []struct {
		query string
		want  string
	}{
		{"SELECT * FROM my_table", "my_table"},
		{"SELECT id FROM schema.my_table WHERE id > 5", "schema.my_table"},
		{"SELECT * FROM a, b", ""},
		{"SELECT * FROM my_table JOIN other ON ...", "my_table"},
		{"UPDATE my_table SET x = 1", ""},
	}

	for _, c := range cases {
		if got := GetTable(c.query); got != c.want {
			t.Errorf("GetTable(%q) = %q, want %q", c.query, got, c.want)
		}
	}
}

//=============================================================================
