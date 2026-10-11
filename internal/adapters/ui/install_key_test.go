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
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListPublicKeys(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"id_rsa.pub", "id_ed25519.pub", "id_ed25519", "config", "known_hosts"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "old.pub"), 0o700); err != nil {
		t.Fatal(err)
	}

	want := []string{filepath.Join(dir, "id_ed25519.pub"), filepath.Join(dir, "id_rsa.pub")}
	if got := listPublicKeys(dir); !reflect.DeepEqual(got, want) {
		t.Errorf("listPublicKeys() = %v, want %v", got, want)
	}
	if got := listPublicKeys(filepath.Join(dir, "missing")); len(got) != 0 {
		t.Errorf("listPublicKeys(missing dir) = %v, want empty", got)
	}
}

func TestPreselectKey(t *testing.T) {
	home := "/home/me"
	keys := []string{"/home/me/.ssh/id_ed25519.pub", "/home/me/.ssh/work.pub"}
	tests := []struct {
		name          string
		identityFiles []string
		want          int
	}{
		{"no identity file", nil, 0},
		{"tilde path", []string{"~/.ssh/work"}, 1},
		{"absolute path", []string{"/home/me/.ssh/work"}, 1},
		{"first match wins", []string{"~/.ssh/other", "~/.ssh/work"}, 1},
		{"no matching pub", []string{"~/.ssh/other"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := preselectKey(keys, tt.identityFiles, home); got != tt.want {
				t.Errorf("preselectKey() = %d, want %d", got, tt.want)
			}
		})
	}
}

// Real public keys generated with ssh-keygen for these tests.
const (
	testEd25519Key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKPfhNgTg54wBWl8/nbpoG1L2H7aQ3K9d+17ZFHGIT/a"
	testECDSAKey   = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHC/9hZbAs5tGa2w0eucEF/AG9AxoDQWBmCT93Y0JQpAuxfQbEKs1V1RH6ygikFJoWuhN6VbHbb2CieR5YQrrmA="
	testRSAKey     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQC3tHSGwst8mCxgRfvUJ4Un4PoPaEhry7LnJEv5UBh1rXeGy4DZRGW0RVjfi1n4/KZgwKfG0e1WPu07MtazU7ripfK5SZDqL3MoUbxwcPgrb5pehJb6DzB2OuzpwhGaqbyNbqTIizGFdsL95dsSc9FRwMDIxbu/bdnBuKDy8iHcNw=="
)

// keyBlob encodes SSH wire-format strings as base64 key data.
func keyBlob(parts ...string) string {
	var b []byte
	for _, p := range parts {
		b = binary.BigEndian.AppendUint32(b, uint32(len(p)))
		b = append(b, p...)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func TestValidatePublicKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"ed25519 with comment", testEd25519Key + " me@laptop", false},
		{"ecdsa", testECDSAKey, false},
		{"rsa with spaces around", "  " + testRSAKey + "  ", false},
		{"security key", "sk-ssh-ed25519@openssh.com " + keyBlob("sk-ssh-ed25519@openssh.com", "key", "ssh:"), false},
		{"empty", "  ", true},
		{"private key pasted", "-----BEGIN OPENSSH PRIVATE KEY-----", true},
		{"unknown type", "foo " + keyBlob("foo", "key"), true},
		{"missing key data", "ssh-ed25519", true},
		{"key data not base64", "ssh-ed25519 not-base64!", true},
		{"only padding", "ssh-ed25519 ====", true},
		{"too short", "ssh-ed25519 AAA=", true},
		{"random base64", "ssh-ed25519 aGVsbG8gd29ybGQ=", true},
		{"type mismatch", "ssh-rsa " + keyBlob("ssh-ed25519", "key"), true},
		{"length beyond data", "ssh-ed25519 " + base64.StdEncoding.EncodeToString([]byte{0xff, 0xff, 0xff, 0xff, 's'}), true},
		{"multiple lines", testEd25519Key + "\n" + testRSAKey, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePublicKey(tt.key); (err != nil) != tt.wantErr {
				t.Errorf("validatePublicKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
