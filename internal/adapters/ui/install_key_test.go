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

func TestValidatePublicKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"ed25519 with comment", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIC9 me@laptop", false},
		{"rsa without comment", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7", false},
		{"ecdsa", "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY=", false},
		{"security key", "sk-ssh-ed25519@openssh.com AAAAGnNrLXNzaC1lZDI1NTE5QG9wZW5zc2guY29t", false},
		{"empty", "  ", true},
		{"private key pasted", "-----BEGIN OPENSSH PRIVATE KEY-----", true},
		{"unknown type", "foo AAAA", true},
		{"missing key data", "ssh-ed25519", true},
		{"key data not base64", "ssh-ed25519 not-base64!", true},
		{"multiple lines", "ssh-ed25519 AAAA\nssh-rsa AAAA", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePublicKey(tt.key); (err != nil) != tt.wantErr {
				t.Errorf("validatePublicKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
