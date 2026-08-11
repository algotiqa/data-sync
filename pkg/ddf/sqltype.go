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

import "fmt"

//=============================================================================

// JDBC type codes (as defined in java.sql.Types) used to tag the canonical
// SQL type names stored in the [FIELDS] section of a DDF file.
const (
	JdbcBit           = -7
	JdbcTinyint       = -6
	JdbcBigInt        = -5
	JdbcLongVarBinary = -4
	JdbcVarBinary     = -3
	JdbcBinary        = -2
	JdbcLongVarChar   = -1
	JdbcNull          = 0
	JdbcChar          = 1
	JdbcNumeric       = 2
	JdbcDecimal       = 3
	JdbcInteger       = 4
	JdbcSmallInt      = 5
	JdbcFloat         = 6
	JdbcReal          = 7
	JdbcDouble        = 8
	JdbcBoolean       = 16
	JdbcDate          = 91
	JdbcTime          = 92
	JdbcTimestamp     = 93
	JdbcVarchar       = 12
	JdbcOther         = 1111
	JdbcJavaObject    = 2000
	JdbcDistinct      = 2001
	JdbcStruct        = 2002
	JdbcArray         = 2003
	JdbcBlob          = 2004
	JdbcClob          = 2005
	JdbcRef           = 2006
)

//============================================================================
// SizeKind describes how the length of a type is declared.
//============================================================================

type SizeKind int

const (
	SizeUnknown SizeKind = iota
	SizeVar              // variable length type (VARCHAR, VARBINARY)
	SizeConst            // fixed length type (BIGINT, BLOB, DATE)
	SizeBoth             // both fixed and variable length are legal (CHAR, BIT)
)

//============================================================================
// SqlType is the canonical description of a column type. Name is always the
// standard JDBC name that gets written into the [FIELDS] section of a DDF
// file, so that files remain database independent.
//============================================================================

type SqlType struct {
	ID   int
	Name string
	Size SizeKind
}

//============================================================================
// NewSqlType creates a new SqlType normalizing a negative size to SizeUnknown.
//============================================================================

func NewSqlType(id int, name string, size SizeKind) SqlType {
	if size < 0 {
		size = SizeUnknown
	}
	return SqlType{ID: id, Name: name, Size: size}
}

func (t SqlType) String() string {
	return "[id:" + fmt.Sprint(t.ID) + ", name:" + t.Name + "]"
}

//============================================================================
// Type predicates
//============================================================================

func (t SqlType) IsBoolean() bool { return t.ID == JdbcBit || t.ID == JdbcBoolean }
func (t SqlType) IsInteger() bool {
	return t.ID == JdbcBigInt || t.ID == JdbcInteger || t.ID == JdbcSmallInt || t.ID == JdbcTinyint
}
func (t SqlType) IsReal() bool {
	return t.ID == JdbcDecimal || t.ID == JdbcDouble || t.ID == JdbcFloat || t.ID == JdbcNumeric || t.ID == JdbcReal
}
func (t SqlType) IsNumber() bool       { return t.IsInteger() || t.IsReal() }
func (t SqlType) IsString() bool       { return t.ID == JdbcChar || t.ID == JdbcVarchar }
func (t SqlType) IsDate() bool         { return t.ID == JdbcDate }
func (t SqlType) IsTime() bool         { return t.ID == JdbcTime }
func (t SqlType) IsTimestamp() bool    { return t.ID == JdbcTimestamp }
func (t SqlType) IsTemporalType() bool { return t.IsDate() || t.IsTime() || t.IsTimestamp() }

func (t SqlType) IsBinary() bool        { return t.ID == JdbcBinary }
func (t SqlType) IsVarBinary() bool     { return t.ID == JdbcVarBinary }
func (t SqlType) IsLongVarBinary() bool { return t.ID == JdbcLongVarBinary }
func (t SqlType) IsBinaryType() bool    { return t.IsBinary() || t.IsVarBinary() || t.IsLongVarBinary() }
func (t SqlType) IsLongVarChar() bool   { return t.ID == JdbcLongVarChar }
func (t SqlType) IsBlob() bool          { return t.ID == JdbcBlob }
func (t SqlType) IsClob() bool          { return t.ID == JdbcClob }

//============================================================================
