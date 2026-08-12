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

package migrate

import "testing"

//=============================================================================

func TestParseMigrationFile(t *testing.T) {
	cases := []struct {
		name    string
		version int
		kind    string
		wantErr bool
	}{
		{name: "001_create_users.sql", version: 1, kind: ".sql"},
		{name: "002_seed_data.ddf", version: 2, kind: ".ddf"},
		{name: "10_zoo.sql", version: 10, kind: ".sql"},
		{name: "1_000_leading.ddf", version: 1, kind: ".ddf"},
		{name: "create_users.sql", wantErr: true},
		{name: "001_users.txt", wantErr: true},
		{name: "_users.sql", wantErr: true},
		{name: "a_users.sql", wantErr: true},
		{name: "001_users.ddf.gz", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			version, kind, err := parseMigrationFile(c.name)

			if c.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got version=%d kind=%q", version, kind)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if version != c.version || kind != c.kind {
				t.Errorf("parseMigrationFile(%q) = %d, %q; want %d, %q", c.name, version, kind, c.version, c.kind)
			}
		})
	}
}

//=============================================================================