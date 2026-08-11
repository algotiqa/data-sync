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

import "strings"

//=============================================================================

const unknownID = -999

var unknownType = SqlType{ID: unknownID, Name: "(unknown)", Size: SizeUnknown}

//=============================================================================
// canonicalTypes is the table of standard JDBC type names. These are the only
// names allowed in the [FIELDS] section of a DDF file.
//=============================================================================

var canonicalTypes = []SqlType{
	{ID: JdbcArray,         Name: "ARRAY",         Size: SizeUnknown},
	{ID: JdbcBigInt,        Name: "BIGINT",        Size: SizeConst},
	{ID: JdbcBinary,        Name: "BINARY",        Size: SizeUnknown},
	{ID: JdbcBit,           Name: "BIT",           Size: SizeBoth},
	{ID: JdbcBlob,          Name: "BLOB",          Size: SizeConst},
	{ID: JdbcChar,          Name: "CHAR",          Size: SizeBoth},
	{ID: JdbcClob,          Name: "CLOB",          Size: SizeConst},
	{ID: JdbcDate,          Name: "DATE",          Size: SizeConst},
	{ID: JdbcDecimal,       Name: "DECIMAL",       Size: SizeBoth},
	{ID: JdbcDistinct,      Name: "DISTINCT",      Size: SizeUnknown},
	{ID: JdbcDouble,        Name: "DOUBLE",        Size: SizeConst},
	{ID: JdbcFloat,         Name: "FLOAT",         Size: SizeBoth},
	{ID: JdbcInteger,       Name: "INTEGER",       Size: SizeConst},
	{ID: JdbcJavaObject,    Name: "JAVA_OBJECT",   Size: SizeUnknown},
	{ID: JdbcLongVarBinary, Name: "LONGVARBINARY", Size: SizeConst},
	{ID: JdbcLongVarChar,   Name: "LONGVARCHAR",   Size: SizeConst},
	{ID: JdbcNull,          Name: "NULL",          Size: SizeUnknown},
	{ID: JdbcNumeric,       Name: "NUMERIC",       Size: SizeBoth},
	{ID: JdbcOther,         Name: "OTHER",         Size: SizeUnknown},
	{ID: JdbcReal,          Name: "REAL",          Size: SizeConst},
	{ID: JdbcRef,           Name: "REF",           Size: SizeUnknown},
	{ID: JdbcSmallInt,      Name: "SMALLINT",      Size: SizeConst},
	{ID: JdbcStruct,        Name: "STRUCT",        Size: SizeUnknown},
	{ID: JdbcTime,          Name: "TIME",          Size: SizeBoth},
	{ID: JdbcTimestamp,     Name: "TIMESTAMP",     Size: SizeBoth},
	{ID: JdbcTinyint,       Name: "TINYINT",       Size: SizeConst},
	{ID: JdbcVarBinary,     Name: "VARBINARY",     Size: SizeVar},
	{ID: JdbcVarchar,       Name: "VARCHAR",       Size: SizeVar},
	{ID: JdbcBoolean,       Name: "BOOLEAN",       Size: SizeConst},
}

//=============================================================================
// dbTypes maps the database specific type names reported by each driver to a
// canonical SqlType. Keys are lower case.
//=============================================================================

var dbTypes = map[string]SqlType{
	// MySQL
	"bit":        {ID: JdbcBit,         Name: "BIT",         Size: SizeBoth},
	"tinyint":    {ID: JdbcTinyint,     Name: "TINYINT",     Size: SizeConst},
	"smallint":   {ID: JdbcSmallInt,    Name: "SMALLINT",    Size: SizeConst},
	"mediumint":  {ID: JdbcInteger,     Name: "INTEGER",     Size: SizeConst},
	"int":        {ID: JdbcInteger,     Name: "INTEGER",     Size: SizeConst},
	"integer":    {ID: JdbcInteger,     Name: "INTEGER",     Size: SizeConst},
	"bigint":     {ID: JdbcBigInt,      Name: "BIGINT",      Size: SizeConst},
	"decimal":    {ID: JdbcDecimal,     Name: "DECIMAL",     Size: SizeBoth},
	"numeric":    {ID: JdbcNumeric,     Name: "NUMERIC",     Size: SizeBoth},
	"float":      {ID: JdbcReal,        Name: "REAL",        Size: SizeConst},
	"double":     {ID: JdbcDouble,      Name: "DOUBLE",      Size: SizeConst},
	"real":       {ID: JdbcReal,        Name: "REAL",        Size: SizeConst},
	"boolean":    {ID: JdbcBoolean,     Name: "BOOLEAN",     Size: SizeConst},
	"bool":       {ID: JdbcBoolean,     Name: "BOOLEAN",     Size: SizeConst},
	"char":       {ID: JdbcChar,        Name: "CHAR",        Size: SizeBoth},
	"varchar":    {ID: JdbcVarchar,     Name: "VARCHAR",     Size: SizeVar},
	"tinytext":   {ID: JdbcLongVarChar, Name: "LONGVARCHAR", Size: SizeConst},
	"text":       {ID: JdbcLongVarChar, Name: "LONGVARCHAR", Size: SizeConst},
	"mediumtext": {ID: JdbcLongVarChar, Name: "LONGVARCHAR", Size: SizeConst},
	"longtext":   {ID: JdbcLongVarChar, Name: "LONGVARCHAR", Size: SizeConst},
	"json":       {ID: JdbcLongVarChar, Name: "LONGVARCHAR", Size: SizeConst},
	"enum":       {ID: JdbcVarchar,     Name: "VARCHAR",     Size: SizeVar},
	"set":        {ID: JdbcVarchar,     Name: "VARCHAR",     Size: SizeVar},
	"date":       {ID: JdbcDate,        Name: "DATE",        Size: SizeConst},
	"time":       {ID: JdbcTime,        Name: "TIME",        Size: SizeBoth},
	"datetime":   {ID: JdbcTimestamp,   Name: "TIMESTAMP",   Size: SizeBoth},
	"timestamp":  {ID: JdbcTimestamp,   Name: "TIMESTAMP",   Size: SizeBoth},
	"binary":     {ID: JdbcBinary,      Name: "BINARY",      Size: SizeUnknown},
	"varbinary":  {ID: JdbcVarBinary,   Name: "VARBINARY",   Size: SizeVar},
	"tinyblob":   {ID: JdbcBlob,        Name: "BLOB",        Size: SizeConst},
	"blob":       {ID: JdbcBlob,        Name: "BLOB",        Size: SizeConst},
	"mediumblob": {ID: JdbcBlob,        Name: "BLOB",        Size: SizeConst},
	"longblob":   {ID: JdbcBlob,        Name: "BLOB",        Size: SizeConst},

	// PostgreSQL
	"int2":        {ID: JdbcSmallInt,    Name: "SMALLINT",    Size: SizeConst},
	"int4":        {ID: JdbcInteger,     Name: "INTEGER",     Size: SizeConst},
	"int8":        {ID: JdbcBigInt,      Name: "BIGINT",      Size: SizeConst},
	"float4":      {ID: JdbcReal,        Name: "REAL",        Size: SizeConst},
	"float8":      {ID: JdbcDouble,      Name: "DOUBLE",      Size: SizeConst},
	"money":       {ID: JdbcDouble,      Name: "DOUBLE",      Size: SizeConst},
	"bpchar":      {ID: JdbcChar,        Name: "CHAR",        Size: SizeBoth},
	"name":        {ID: JdbcVarchar,     Name: "VARCHAR",     Size: SizeVar},
	"citext":      {ID: JdbcLongVarChar, Name: "LONGVARCHAR", Size: SizeConst},
	"jsonb":       {ID: JdbcLongVarChar, Name: "LONGVARCHAR", Size: SizeConst},
	"xml":         {ID: JdbcLongVarChar, Name: "LONGVARCHAR", Size: SizeConst},
	"timetz":      {ID: JdbcTime,        Name: "TIME",        Size: SizeBoth},
	"timestamptz": {ID: JdbcTimestamp,   Name: "TIMESTAMP",   Size: SizeBoth},
	"bytea":       {ID: JdbcBlob,        Name: "BLOB",        Size: SizeConst},
}

//=============================================================================
// MapName returns the canonical type whose name matches name. It is used to
// decode the [FIELDS] section of a DDF file.
//=============================================================================

func MapName(name string) SqlType {
	for _, t := range canonicalTypes {
		if name == t.Name {
			return t
		}
	}

	return unknownType
}

//=============================================================================
// MapID returns the canonical type whose JDBC code matches the given id.
//=============================================================================

func MapID(id int) SqlType {
	for _, t := range canonicalTypes {
		if id == t.ID {
			return t
		}
	}

	return unknownType
}

//=============================================================================
// MapDBType converts a database specific type name, as reported by a driver,
// into its canonical SqlType. Decimals tells whether the type carries a
// non-zero scale; decimal/numeric types with no decimals are remapped to
// INTEGER because some DBMSs (like Oracle or MySQL) report them as integers.
//=============================================================================

func MapDBType(dbType string, hasDecimals bool) SqlType {
	key := strings.ToLower(strings.TrimSpace(dbType))

	t, ok := dbTypes[key]
	if !ok {
		return unknownType
	}

	if (t.ID == JdbcDecimal || t.ID == JdbcNumeric) && !hasDecimals {
		return MapID(JdbcInteger)
	}

	return t
}

//=============================================================================
