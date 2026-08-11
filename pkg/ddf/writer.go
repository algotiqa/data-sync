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
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"time"
)

//=============================================================================

const writerBufferSize = 32768

//============================================================================
// Writer serializes rows into the DDF format. A DDF file starts with a header
// (comments, [INFO] and [FIELDS] sections) followed by one or more data
// sections ([INSERT], [UPDATE], [DELETE]).
//============================================================================

type Writer struct {
	bw              *bufio.Writer
	gz              *gzip.Writer
	query           string
	table           string
	schema          string
	fields          []QueryField
	isHeaderWritten bool
}

//============================================================================
// NewWriter creates a Writer over the given stream, optionally compressing its output with gzip.
//============================================================================

func NewWriter(w io.Writer, useCompression bool) *Writer {
	var gz *gzip.Writer

	if useCompression {
		gz = gzip.NewWriter(w)
		w = gz
	}

	return &Writer{bw: bufio.NewWriterSize(w, writerBufferSize), gz: gz}
}

//============================================================================
// SetQuery stores the query the exported data comes from. It is written in the header comment block.
//============================================================================

func (w *Writer) SetQuery(query string) {
	w.query = query
}

//============================================================================
// SetTable sets the table name written in the [INFO] section.
//============================================================================

func (w *Writer) SetTable(table string) {
	w.table = table
}

//============================================================================
// SetSchema sets the schema name written in the [INFO] section.
//============================================================================

func (w *Writer) SetSchema(schema string) {
	w.schema = schema
}

//============================================================================
// SetFields sets the fields of the exported data.
//============================================================================

func (w *Writer) SetFields(fields []QueryField) {
	w.fields = fields
}

//============================================================================
// BeginInsertSection starts the [INSERT] section, writing the header first if needed.
//============================================================================

func (w *Writer) BeginInsertSection() error {
	return w.beginSection(SectionInsert)
}

//============================================================================
// BeginUpdateSection starts the [UPDATE] section, writing the header first if needed.
//============================================================================

func (w *Writer) BeginUpdateSection() error {
	return w.beginSection(SectionUpdate)
}

//============================================================================
// BeginDeleteSection starts the [DELETE] section, writing the header first if needed.
//============================================================================

func (w *Writer) BeginDeleteSection() error {
	return w.beginSection(SectionDelete)
}

//============================================================================
// WriteRow writes a single data row. Values may be string, []byte or time.Time
// and are encoded according to the DDF conventions.
//============================================================================

func (w *Writer) WriteRow(row []any) error {
	for i, v := range row {
		if v != nil {
			switch x := v.(type) {
			case []byte:
				if _, err := w.bw.WriteString(EncodeBytes(x)); err != nil {
					return err
				}
			case time.Time:
				if _, err := w.bw.WriteString(x.UTC().Format(TimestampLayout)); err != nil {
					return err
				}
			default:
				if _, err := w.bw.WriteString(EncodeString(fmt.Sprint(x))); err != nil {
					return err
				}
			}
		}

		if i != len(row)-1 {
			if err := w.bw.WriteByte('\t'); err != nil {
				return err
			}
		}
	}

	return w.bw.WriteByte('\n')
}

//============================================================================
// End flushes the buffered data, closing the gzip stream when the writer
// created one. The underlying writer is not closed.
//============================================================================

func (w *Writer) End() error {
	if err := w.bw.Flush(); err != nil {
		return err
	}

	if w.gz != nil {
		return w.gz.Close()
	}

	return nil
}

//=============================================================================

func (w *Writer) beginSection(section Section) error {
	if !w.isHeaderWritten {
		if err := w.writeHeader(); err != nil {
			return err
		}
		w.isHeaderWritten = true
	}

	if _, err := w.bw.WriteString("\n" + string(section) + "\n"); err != nil {
		return err
	}

	return nil
}

//============================================================================

func (w *Writer) writeHeader() error {
	if _, err := w.bw.WriteString("#---------------------------------------------------------\n"); err != nil {
		return err
	}
	if _, err := w.bw.WriteString("#--- File exported in DDF format on " + currentDate() + "\n"); err != nil {
		return err
	}
	if _, err := w.bw.WriteString("#---\n"); err != nil {
		return err
	}

	if w.query != "" {
		query := strings.ReplaceAll(w.query, "\n", " ")
		query = strings.ReplaceAll(query, "\r", " ")

		if _, err := w.bw.WriteString("#--- " + query + "\n"); err != nil {
			return err
		}
	}

	if _, err := w.bw.WriteString("#---------------------------------------------------------\n\n"); err != nil {
		return err
	}
	if _, err := w.bw.WriteString(string(SectionInfo) + "\n"); err != nil {
		return err
	}
	if _, err := w.bw.WriteString("version=" + Version + "\n"); err != nil {
		return err
	}

	if w.table != "" {
		if _, err := w.bw.WriteString("table=" + w.table + "\n"); err != nil {
			return err
		}

		if w.schema != "" {
			if _, err := w.bw.WriteString("schema=" + w.schema + "\n"); err != nil {
				return err
			}
		}
	}

	if _, err := w.bw.WriteString("\n" + string(SectionFields) + "\n"); err != nil {
		return err
	}

	for _, field := range w.fields {
		if _, err := w.bw.WriteString(encodeField(field) + "\n"); err != nil {
			return err
		}
	}

	return nil
}

//============================================================================

func encodeField(field QueryField) string {
	key := "N"
	if field.IsKey {
		key = "K"
	}

	return pad(field.Name+",", 15) + pad(field.Type.Name+",", 12) + key
}

//============================================================================
// pad right pads the given text with spaces up to the given total length.
//============================================================================

func pad(text string, chars int) string {
	if len(text) >= chars {
		return text
	}

	return text + strings.Repeat(" ", chars-len(text))
}

//============================================================================
// currentDate returns the local date and time in the format used in the DDF header comment.
//============================================================================

func currentDate() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

//=============================================================================
