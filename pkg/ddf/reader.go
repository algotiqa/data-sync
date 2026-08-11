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
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

//=============================================================================

const readerBufferSize = 32768

//=============================================================================
//
// Reader parses a DDF file, dispatching every encountered section to a ReaderListener.
//
//=============================================================================

type Reader struct {
	br   *bufio.Reader
	info Info
}

//=============================================================================
// NewReader creates a Reader over the given stream, optionally decompressing it with gzip.
//=============================================================================

func NewReader(r io.Reader, useCompression bool) (*Reader, error) {
	if useCompression {
		gz, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}

		r = gz
	}

	return &Reader{br: bufio.NewReaderSize(r, readerBufferSize)}, nil
}

//=============================================================================
// Read parses the whole DDF input, dispatching events to the given listener.
//=============================================================================

func (r *Reader) Read(listener ReaderListener) error {
	var section Section
	var fields []QueryField
	fieldsFired := false
	var (
		addedCount   int64
		updatedCount int64
		removedCount int64
	)

	for {
		line, err := r.br.ReadString('\n')

		if len(line) > 0 {
			line = strings.TrimRight(line, "\n")
			line = strings.TrimRight(line, "\r")

			if line != "" && !strings.HasPrefix(line, "#") {
				switch Section(line) {
					case SectionInfo:
						section = SectionInfo
					case SectionFields:
						section = SectionFields
					case SectionInsert, SectionData: // SectionData kept for backward compatibility
						section = SectionInsert
					case SectionUpdate:
						section = SectionUpdate
					case SectionDelete:
						section = SectionDelete
					default:
						err = r.handleLine(listener, section, &fields, &fieldsFired, line, &addedCount, &updatedCount, &removedCount)
				}
			}
		}

		if errors.Is(err, errStop) {
			break
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	if section != SectionInsert && section != SectionUpdate && section != SectionDelete {
		return errors.New("unexpected EOF encountered")
	}

	return nil
}

//=============================================================================
// Info returns the metadata read from the [INFO] section.
//=============================================================================

func (r *Reader) Info() Info {
	return r.info
}

//=============================================================================

func (r *Reader) handleLine(listener ReaderListener, section Section, fields *[]QueryField, fieldsFired *bool,
							line string, addedCount, updatedCount, removedCount *int64) error {

	// Fires query fields before the first data row, if not fired yet

	if (section == SectionInsert || section == SectionUpdate || section == SectionDelete) && !*fieldsFired {
		listener.HandleQueryFields(*fields)
		*fieldsFired = true
	}

	switch section {
		case SectionInfo:
			r.handleInfo(listener, line)

		case SectionFields:
			field, err := decodeField(line)
			if err != nil {
				return err
			}
			*fields = append(*fields, field)

		case SectionInsert:
			*addedCount++
			return r.handleDataRow(listener, OpInsert, *fields, line, *addedCount)

		case SectionUpdate:
			*updatedCount++
			return r.handleDataRow(listener, OpUpdate, *fields, line, *updatedCount)

		case SectionDelete:
			*removedCount++
			return r.handleDataRow(listener, OpDelete, *fields, line, *removedCount)

		default:
			return errors.New("data not allowed in this section")
	}

	return nil
}

//=============================================================================

func (r *Reader) handleInfo(listener ReaderListener, line string) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) < 2 {
		return
	}

	switch strings.TrimSpace(parts[0]) {
		case "version":
			r.info.Version = strings.TrimSpace(parts[1])
		case "table":
			r.info.Table = strings.TrimSpace(parts[1])
		case "schema":
			r.info.Schema = strings.TrimSpace(parts[1])
	}

	listener.HandleInfo(r.info)
}

//=============================================================================
// decodeField parses a line of the [FIELDS] section.
//=============================================================================

func decodeField(line string) (QueryField, error) {
	parts := strings.Split(line, ",")

	if len(parts) < 3 {
		return QueryField{}, errors.New("bad field format")
	}

	name     := strings.TrimSpace(parts[0])
	typeName := strings.TrimSpace(parts[1])
	pkey     := strings.TrimSpace(parts[2])
	sqlType  := MapName(typeName)

	if typeName != sqlType.Name {
		return QueryField{}, fmt.Errorf("column types differ [%s/%s]", typeName, sqlType.Name)
	}

	return QueryField{Name: name, Type: sqlType, IsKey: pkey == "K"}, nil
}

//=============================================================================

func (r *Reader) handleDataRow(listener ReaderListener, op Operation, fields []QueryField, line string, recordNum int64) error {
	row, err := decodeRow(fields, line)
	if err != nil {
		return err
	}

	ok, err := listener.HandleRow(op, row, line, recordNum)
	if err != nil {
		return err
	}
	if !ok {
		return errStop
	}

	listener.HandlePostRow(op, line, recordNum)
	return nil
}

//=============================================================================
// decodeRow converts a tab separated data line into a slice of typed values.
//=============================================================================

func decodeRow(fields []QueryField, line string) ([]any, error) {
	tokens := strings.Split(line, "\t")

	if len(tokens) != len(fields) {
		return nil, fmt.Errorf("fields count differs from token count :\n%s", line)
	}

	row := make([]any, len(fields))

	for i, f := range fields {
		token := tokens[i]
		t     := f.Type

		switch {
			case token == "":
				row[i] = nil

			case t.IsDate():
				v, err := time.ParseInLocation(DateLayout, token, time.UTC)
				if err != nil {
					return nil, err
				}
				row[i] = v

			case t.IsTime():
				row[i] = normalizeTime(token)

			case t.IsTimestamp():
				v, err := parseTimestamp(token)
				if err != nil {
					return nil, err
				}
				row[i] = v

			case t.IsInteger():
				v, err := strconv.ParseInt(token, 10, 64)
				if err != nil {
					return nil, err
				}
				row[i] = v

			case t.IsReal():
				v, err := strconv.ParseFloat(token, 64)
				if err != nil {
					return nil, err
				}
				row[i] = v

			case t.IsBoolean():
				row[i] = token == "T"

			case t.IsBinaryType() || t.IsBlob():
				row[i] = DecodeBytes(token)

			case t.IsClob():
				row[i] = DecodeString(token)

			case t.IsString() || t.IsLongVarChar():
				row[i] = DecodeString(token)
		}
	}

	return row, nil
}

//=============================================================================
// parseTimestamp parses a timestamp honoring the DDF version. Version 1 and 2
// files use the JDBC representation, version 3 the canonical one.
//=============================================================================

func parseTimestamp(token string) (time.Time, error) {
	for _, layout := range []string{TimestampLayout, "2006-01-02 15:04:05"} {
		if v, err := time.ParseInLocation(layout, token, time.UTC); err == nil {
			return v, nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse timestamp %q", token)
}

//=============================================================================

var errStop = errors.New("parsing stopped by listener")

//=============================================================================
