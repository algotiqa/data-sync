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

var queryFields = []QueryField{
	{Name: "id",      Type: MapName("BIGINT"), IsKey: true},
	{Name: "name",    Type: MapName("VARCHAR")},
	{Name: "created", Type: MapName("TIMESTAMP")},
}

//============================================================================

func TestInsertQuery(t *testing.T) {
	cases := []struct {
		isPostgres bool
		want       string
	}{
		{false, "INSERT INTO t(id,name,created) VALUES (?,?,?)"},
		{true, "INSERT INTO t(id,name,created) VALUES ($1,$2,$3)"},
	}

	for _, c := range cases {
		if got := insertQuery("t", queryFields, c.isPostgres); got != c.want {
			t.Errorf("insertQuery(postgres=%v) = %q, want %q", c.isPostgres, got, c.want)
		}
	}
}

//============================================================================

func TestUpdateQuery(t *testing.T) {
	cases := map[string]string{
		"mysql":    "UPDATE t SET name = ?,created = ? WHERE id = ?",
		"postgres": "UPDATE t SET name = $1,created = $2 WHERE id = $3",
	}

	for dbType, want := range cases {
		got, err := updateQuery("t", queryFields, dbType == "postgres")
		if err != nil {
			t.Fatal(err)
		}

		if got != want {
			t.Errorf("updateQuery(%s) = %q, want %q", dbType, got, want)
		}
	}
}

//============================================================================

func TestUpdateQueryMissingPKey(t *testing.T) {
	_, err := updateQuery("t", queryFields[1:], false)
	if err == nil {
		t.Fatal("expected an error when no primary key is present")
	}
}

//============================================================================

func TestDeleteQuery(t *testing.T) {
	cases := map[string]struct {
		want      string
		missingPk bool
	}{
		"mysql": {
			want: "DELETE FROM t WHERE id = ?",
		},
		"postgres": {
			want: "DELETE FROM t WHERE id = $1",
		},
	}

	for dbType, c := range cases {
		got, missingPk := deleteQuery("t", queryFields, dbType == "postgres")

		if got != c.want || missingPk != c.missingPk {
			t.Errorf("deleteQuery(%s) = %q, %v; want %q, %v", dbType, got, missingPk, c.want, c.missingPk)
		}
	}
}

//============================================================================

func TestDeleteQueryNoPrimaryKey(t *testing.T) {
	fields := []QueryField{
		{Name: "a", Type: MapName("VARCHAR")},
		{Name: "b", Type: MapName("INTEGER")},
	}

	got, missingPk := deleteQuery("t", fields, true)
	want := "DELETE FROM t WHERE a = $1 AND b = $2"

	if got != want || !missingPk {
		t.Errorf("deleteQuery(no pk) = %q, %v; want %q, true", got, missingPk, want)
	}
}

//=============================================================================
