// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestCompleteLocalPath(t *testing.T) {
	home := t.TempDir()
	mustWrite := func(rel string) {
		t.Helper()
		p := filepath.Join(home, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("docs/readme.txt")
	mustWrite("docs/report.pdf")
	mustWrite("docs/notes.md")
	mustWrite("docs/.hidden")
	mustWrite("docs/sub/file")

	d := home + "/docs/"
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"all non-hidden in dir", d, []string{d + "notes.md", d + "readme.txt", d + "report.pdf", d + "sub/"}},
		{"basename prefix", d + "re", []string{d + "readme.txt", d + "report.pdf"}},
		{"dir gets trailing slash", d + "su", []string{d + "sub/"}},
		{"dotfiles when prefix is dot", d + ".", []string{d + ".hidden"}},
		{"tilde kept", "~/docs/re", []string{"~/docs/readme.txt", "~/docs/report.pdf"}},
		{"tilde alone", "~/", []string{"~/docs/"}},
		{"missing dir", home + "/nope/x", nil},
		{"empty", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := completeLocalPath(tt.text, home); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("completeLocalPath(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestCompleteLocalPathCapsResults(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 30; i++ {
		if err := os.WriteFile(filepath.Join(dir, "f"+strconv.Itoa(i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got := completeLocalPath(dir+"/f", dir); len(got) != maxPathSuggestions {
		t.Errorf("got %d suggestions, want %d", len(got), maxPathSuggestions)
	}
}

func TestExpandLocalPath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"~/a/b", filepath.Join("/home/u", "a/b")},
		{"/abs/x", "/abs/x"},
		{"rel/x", "rel/x"},
		{"~other/x", "~other/x"},
	}
	for _, tt := range tests {
		if got := expandLocalPath(tt.in, "/home/u"); got != tt.want {
			t.Errorf("expandLocalPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidateTransfer(t *testing.T) {
	home := t.TempDir()
	file := filepath.Join(home, "a.txt")
	dir := filepath.Join(home, "d")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	newFile := filepath.Join(home, "new.txt")

	tests := []struct {
		name          string
		mode          int
		local, remote string
		recursive     bool
		wantLocal     string
		wantRecursive bool
		wantErr       bool
	}{
		{"sftp needs nothing", transferSFTP, "", "", false, "", false, false},
		{"upload ok", transferUpload, file, "", false, file, false, false},
		{"upload tilde expanded", transferUpload, "~/a.txt", "x", false, file, false, false},
		{"upload dir forces recursive", transferUpload, dir, "", false, dir, true, false},
		{"upload empty local", transferUpload, "  ", "", false, "", false, true},
		{"upload missing local", transferUpload, filepath.Join(home, "nope"), "", false, "", false, true},
		{"download to new file in existing dir", transferDownload, newFile, "/srv/a", false, newFile, false, false},
		{"download to existing dir", transferDownload, dir, "/srv/a", true, dir, true, false},
		{"download empty remote", transferDownload, dir, "", false, "", false, true},
		{"download empty local", transferDownload, "", "/srv/a", false, "", false, true},
		{"download parent missing", transferDownload, filepath.Join(home, "no", "x"), "/srv/a", false, "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLocal, gotRec, err := validateTransfer(tt.mode, tt.local, tt.remote, tt.recursive, home)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if gotLocal != tt.wantLocal || gotRec != tt.wantRecursive {
				t.Errorf("got (%q, %v), want (%q, %v)", gotLocal, gotRec, tt.wantLocal, tt.wantRecursive)
			}
		})
	}
}
