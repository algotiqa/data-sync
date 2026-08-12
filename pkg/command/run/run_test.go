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

package run

import (
	"strings"
	"testing"
)

//=============================================================================

func TestParseStatements(t *testing.T) {
	cases := []struct {
		name    string
		script  string
		want    []string
		wantErr bool
	}{
		{
			name:   "single statement",
			script: "SELECT 1;\n",
			want:   []string{"SELECT 1"},
		},
		{
			name:   "multiple statements",
			script: "SELECT 1;\nSELECT 2;\nSELECT 3;\n",
			want:   []string{"SELECT 1", "SELECT 2", "SELECT 3"},
		},
		{
			name:   "multiline statement",
			script: "CREATE TABLE t (\n  id INT,\n  name VARCHAR(20)\n);\n",
			want:   []string{"CREATE TABLE t (\n  id INT,\n  name VARCHAR(20)\n)"},
		},
		{
			name:   "trailing whitespace",
			script: "  SELECT 1 ;  \n",
			want:   []string{"SELECT 1"},
		},
		{
			name:   "empty statements skipped",
			script: ";\nSELECT 1;\n\n;\n",
			want:   []string{"SELECT 1"},
		},
		{
			name:   "blank lines ignored",
			script: "\n\nSELECT 1;\n\nSELECT 2;\n",
			want:   []string{"SELECT 1", "SELECT 2"},
		},
		{
			name:   "no semicolon at eof",
			script: "SELECT 1",
			wantErr: true,
		},
		{
			name:   "empty script",
			script: "\n\n",
			want:   nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseStatements(strings.NewReader(c.script))

			if c.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(c.want) {
				t.Fatalf("statements = %#v, want %#v", got, c.want)
			}

			for i := range c.want {
				if got[i] != c.want[i] {
					t.Errorf("statement %d = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

//=============================================================================