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
	"bytes"
	"testing"
	"time"
)

//=============================================================================

func TestWriterReaderRoundTrip(t *testing.T) {
	testWriterReaderRoundTrip(t, false)
}

//============================================================================

func TestWriterReaderRoundTripCompressed(t *testing.T) {
	testWriterReaderRoundTrip(t, true)
}

//============================================================================

func testWriterReaderRoundTrip(t *testing.T, compress bool) {
	fields := []QueryField{
		{Name: "id", Type: MapName("BIGINT"), IsKey: true},
		{Name: "name", Type: MapName("VARCHAR")},
		{Name: "created", Type: MapName("TIMESTAMP")},
		{Name: "amount", Type: MapName("DECIMAL")},
		{Name: "active", Type: MapName("BOOLEAN")},
		{Name: "data", Type: MapName("BLOB")},
		{Name: "notes", Type: MapName("LONGVARCHAR")},
	}

	created := time.Date(2026, 1, 2, 3, 4, 5, 123000000, time.UTC)

	// Rows written in the representation produced by the exporter
	writeRows := [][]any{
		{int64(1), "first row", created, "12.34", "T", []byte{0x01, 0x02}, "line1\nline2"},
		{int64(2), "accents àè and japan 日", created, "0.50", "F", []byte{0xFF}, "emoji 😀"},
		{nil, nil, nil, nil, nil, nil, nil},
	}

	// Rows as they should come back from the reader
	readRows := [][]any{
		{int64(1), "first row", created, 12.34, true, []byte{0x01, 0x02}, "line1\nline2"},
		{int64(2), "accents àè and japan 日", created, 0.5, false, []byte{0xFF}, "emoji 😀"},
		{nil, nil, nil, nil, nil, nil, nil},
	}

	var buf bytes.Buffer

	writer := NewWriter(&buf, compress)
	writer.SetQuery("SELECT * FROM test_table")
	writer.SetTable("test_table")
	writer.SetSchema("test_schema")
	writer.SetFields(fields)

	if err := writer.BeginInsertSection(); err != nil {
		t.Fatal(err)
	}

	for _, row := range writeRows {
		if err := writer.WriteRow(row); err != nil {
			t.Fatal(err)
		}
	}

	if err := writer.End(); err != nil {
		t.Fatal(err)
	}

	reader, err := NewReader(bytes.NewReader(buf.Bytes()), compress)
	if err != nil {
		t.Fatal(err)
	}

	listener := &recordingListener{}

	if err := reader.Read(listener); err != nil {
		t.Fatal(err)
	}

	if listener.info.Table != "test_table" || listener.info.Schema != "test_schema" || listener.info.Version != Version {
		t.Errorf("unexpected info: %+v", listener.info)
	}

	if len(listener.fields) != len(fields) {
		t.Fatalf("fields count = %d, want %d", len(listener.fields), len(fields))
	}

	for i, f := range fields {
		if listener.fields[i].Name != f.Name || listener.fields[i].IsKey != f.IsKey {
			t.Errorf("field %d mismatch: got %+v want %+v", i, listener.fields[i], f)
		}
	}

	if len(listener.rows) != len(readRows) {
		t.Fatalf("rows count = %d, want %d", len(listener.rows), len(readRows))
	}

	for i, want := range readRows {
		got := listener.rows[i].row

		if len(got) != len(want) {
			t.Fatalf("row %d: len = %d, want %d", i, len(got), len(want))
		}

		for j := range want {
			if !compareValues(got[j], want[j]) {
				t.Errorf("row %d col %d: got %#v (%T), want %#v (%T)", i, j, got[j], got[j], want[j], want[j])
			}
		}
	}
}

//============================================================================

func TestReaderRejectsDataOutsideSections(t *testing.T) {
	_, err := NewReader(bytes.NewBufferString("garbage\n"), false)
	if err != nil {
		t.Fatal(err)
	}

	reader, err := NewReader(bytes.NewBufferString("garbage\n"), false)
	if err != nil {
		t.Fatal(err)
	}

	err = reader.Read(&recordingListener{})
	if err == nil {
		t.Fatal("expected an error for data outside any section")
	}
}

//=============================================================================

func compareValues(got, want any) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}

	gb, ok1 := got.([]byte)
	wb, ok2 := want.([]byte)

	if ok1 && ok2 {
		return bytes.Equal(gb, wb)
	}

	return got == want
}

//=============================================================================

type recordedRow struct {
	op        Operation
	row       []any
	line      string
	recordNum int64
}

//============================================================================

type recordingListener struct {
	info   Info
	fields []QueryField
	rows   []recordedRow
}

//============================================================================

func (l *recordingListener) HandleInfo(info Info) {
	l.info = info
}

//============================================================================

func (l *recordingListener) HandleQueryFields(fields []QueryField) {
	l.fields = fields
}

//============================================================================

func (l *recordingListener) HandleRow(op Operation, row []any, line string, recordNum int64) (bool, error) {
	l.rows = append(l.rows, recordedRow{op: op, row: row, line: line, recordNum: recordNum})
	return true, nil
}

//============================================================================

func (l *recordingListener) HandlePostRow(op Operation, line string, recordNum int64) {
}

//=============================================================================
