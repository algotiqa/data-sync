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
)

//=============================================================================

func TestEncodeDecodeString(t *testing.T) {
	cases := []string{
		"",
		"hello world",
		"tab\tand\nnewline",
		"tilde ~ and tilde",
		"accents àèìòù",
		"emoji 😀 and japan 日本",
		"\x00\x01\x1f\x7f",
		"mixed plain text with symbols !@#$%^&*()",
	}

	for _, in := range cases {
		enc := EncodeString(in)
		dec := DecodeString(enc)

		if dec != in {
			t.Errorf("round trip failed for %q: encoded %q decoded %q", in, enc, dec)
		}
	}
}

//=============================================================================

func TestEncodeStringControlChars(t *testing.T) {
	if got := EncodeString("a\nb"); got != "a~000Ab" {
		t.Errorf("expected a~000Ab, got %q", got)
	}

	// '~' (0x7e) is not printable ASCII per the DDF codec
	if got := EncodeString("~"); got != "~007E" {
		t.Errorf("expected ~007E, got %q", got)
	}
}

//=============================================================================

func TestEncodeDecodeBytes(t *testing.T) {
	cases := []struct {
		data []byte
		hex  string
	}{
		{nil, ""},
		{[]byte{0x00}, "00"},
		{[]byte{0x0A, 0xFF, 0x00, 0x80}, "0AFF0080"},
		{[]byte("abc"), "616263"},
	}

	for _, c := range cases {
		if got := EncodeBytes(c.data); got != c.hex {
			t.Errorf("EncodeBytes(%v) = %q, want %q", c.data, got, c.hex)
		}

		if got := DecodeBytes(c.hex); !bytes.Equal(got, c.data) {
			t.Errorf("DecodeBytes(%q) = %v, want %v", c.hex, got, c.data)
		}
	}
}

//=============================================================================
