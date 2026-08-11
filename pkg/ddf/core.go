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

//=============================================================================

// Version is the DDF format version written in the [INFO] section.
const Version = "3"

// TimestampLayout is the canonical representation of a timestamp in a DDF file,
// expressed in UTC with millisecond precision.
const TimestampLayout = "2006-01-02 15:04:05.000"

// DateLayout is the representation of a DATE value in a DDF file.
const DateLayout = "2006-01-02"

// TimeLayout is the representation of a TIME value in a DDF file.
const TimeLayout = "15:04:05"

//=============================================================================
// Section identifies one of the logical sections of a DDF file.
//=============================================================================

type Section string

const (
	SectionInfo   Section = "[INFO]"
	SectionFields Section = "[FIELDS]"
	SectionInsert Section = "[INSERT]"
	SectionUpdate Section = "[UPDATE]"
	SectionDelete Section = "[DELETE]"
	SectionData   Section = "[DATA]" // legacy alias for [INSERT]
)

//=============================================================================
// Operation is the type of row operation carried by a DDF record.
//=============================================================================

type Operation string

const (
	OpInsert Operation = "[INSERT]"
	OpUpdate Operation = "[UPDATE]"
	OpDelete Operation = "[DELETE]"
)

//=============================================================================
// QueryField describes a single column of an exported table.
//=============================================================================

type QueryField struct {
	Name  string
	Type  SqlType
	IsKey bool
}

//=============================================================================
// Info carries the metadata stored in the [INFO] section of a DDF file.
//=============================================================================

type Info struct {
	Version string
	Table   string
	Schema  string
}

//=============================================================================
// ReaderListener receives the events fired while parsing a DDF file. The row
// handlers return false to stop the parsing.
//=============================================================================

type ReaderListener interface {
	HandleInfo(info Info)
	HandleQueryFields(fields []QueryField)
	HandleRow(op Operation, row []any, line string, recordNum int64) (bool, error)
	HandlePostRow(op Operation, line string, recordNum int64)
}

//=============================================================================
// ErrorAction is the action to take after a row could not be applied.
//=============================================================================

type ErrorAction int

const (
	ActionAbort ErrorAction = iota
	ActionSkip
	ActionSkipAll
)

//=============================================================================
// ImportListener receives progress notifications while importing a DDF file.
//=============================================================================

type ImportListener interface {
	InsertRow(fields []QueryField, row []any, recordNum int64)
	UpdateRow(fields []QueryField, row []any, recordNum int64)
	DeleteRow(fields []QueryField, row []any, recordNum int64)
	OnError(op Operation, err error, recordNum int64, line string) ErrorAction
	PostRow(op Operation, recordNum int64, line string)
}

//=============================================================================
// ExportListener receives a notification for every exported row.
//=============================================================================

type ExportListener interface {
	ExportedRow(row []any, recordNum int64)
}

//=============================================================================
