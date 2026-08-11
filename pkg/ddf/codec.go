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

const hexDigits = "0123456789ABCDEF"

//=============================================================================
// EncodeString converts a string into its DDF representation. Printable ASCII
// characters are kept as-is; every other code point is replaced by '~' followed
// by its 4 digit hexadecimal value. Code points above the BMP are encoded as a
// UTF-16 surrogate pair so that the representation is unambiguous.
//=============================================================================

func EncodeString(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		switch {
			case r >= 32 && r < 126:
				b.WriteRune(r)
			case r >= 0x10000:
				r1 := 0xD800 + ((r - 0x10000) >> 10)
				r2 := 0xDC00 + ((r - 0x10000) & 0x3FF)
				b.WriteByte('~')
				writeHex(&b, uint32(r1), 4)
				b.WriteByte('~')
				writeHex(&b, uint32(r2), 4)
			default:
				b.WriteByte('~')
				writeHex(&b, uint32(r), 4)
		}
	}

	return b.String()
}

//=============================================================================

func writeHex(b *strings.Builder, n uint32, digits int) {
	for i := 0; i < digits; i++ {
		b.WriteByte(hexDigits[(n>>(4*(digits-1-i)))&0xF])
	}
}

//=============================================================================
// DecodeString is the inverse of EncodeString.
//=============================================================================

func DecodeString(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] != '~' {
			b.WriteByte(s[i])
			continue
		}

		if i+5 > len(s) {
			break
		}

		v := convertFromHex(s[i+1 : i+5])
		i += 4

		if v >= 0xD800 && v <= 0xDBFF && i+1 < len(s) && s[i+1] == '~' && i+6 <= len(s) {
			lv := convertFromHex(s[i+2 : i+6])
			if lv >= 0xDC00 && lv <= 0xDFFF {
				r := 0x10000 + (v-0xD800)<<10 + (lv - 0xDC00)
				b.WriteRune(rune(r))
				i += 5
				continue
			}
		}

		b.WriteRune(rune(v))
	}

	return b.String()
}

//=============================================================================
// EncodeBytes converts a byte slice into its hexadecimal DDF representation.
//=============================================================================

func EncodeBytes(data []byte) string {
	var b strings.Builder
	b.Grow(len(data) * 2)

	for _, x := range data {
		b.WriteByte(hexDigits[(x>>4)&0xF])
		b.WriteByte(hexDigits[x&0xF])
	}

	return b.String()
}

//=============================================================================
// DecodeBytes is the inverse of EncodeBytes.
//=============================================================================

func DecodeBytes(data string) []byte {
	arr := make([]byte, len(data)/2)

	for i := 0; i < len(arr); i++ {
		arr[i] = byte(convertFromHex(data[i*2 : i*2+2]))
	}

	return arr
}

//=============================================================================

func convertFromHex(s string) int {
	var n int

	for i := 0; i < len(s); i++ {
		c := s[i]
		n <<= 4

		switch {
		case c >= '0' && c <= '9':
			n += int(c - '0')
		case c >= 'a' && c <= 'f':
			n += int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			n += int(c-'A') + 10
		}
	}

	return n
}

//=============================================================================
