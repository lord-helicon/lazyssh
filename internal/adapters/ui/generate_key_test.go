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
	"strings"
	"testing"
)

func TestDefaultKeyName(t *testing.T) {
	tests := []struct {
		keyType, alias, want string
	}{
		{"ed25519", "web1", "id_ed25519_web1"},
		{"rsa", "web1", "id_rsa_web1"},
		{"ecdsa", "web1", "id_ecdsa_web1"},
		{"ed25519", "my server/prod:22", "id_ed25519_my_server_prod_22"},
		{"ed25519", "a.b-c_d", "id_ed25519_a.b-c_d"},
	}
	for _, tt := range tests {
		if got := defaultKeyName(tt.keyType, tt.alias); got != tt.want {
			t.Errorf("defaultKeyName(%q, %q) = %q, want %q", tt.keyType, tt.alias, got, tt.want)
		}
	}
}

func TestNameAfterTypeChange(t *testing.T) {
	// Still the previous default: follows the new type.
	if got := nameAfterTypeChange("id_ed25519_web1", "web1", "ed25519", "rsa"); got != "id_rsa_web1" {
		t.Errorf("got %q, want id_rsa_web1", got)
	}
	// Edited by the user: left alone.
	if got := nameAfterTypeChange("my_key", "web1", "ed25519", "rsa"); got != "my_key" {
		t.Errorf("got %q, want my_key", got)
	}
}

func TestValidateKeyName(t *testing.T) {
	dir := t.TempDir()
	write := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("has_private")
	write("has_pub.pub")

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"valid", "id_ed25519_web1", ""},
		{"valid dots and dashes", "my-key.v2", ""},
		{"empty", "", "empty"},
		{"slash", "a/b", "letters"},
		{"space", "my key", "letters"},
		{"tilde", "~key", "letters"},
		{"pub suffix", "mykey.pub", ".pub"},
		{"existing private key", "has_private", "already exists"},
		{"existing pub key only", "has_pub", "already exists"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKeyName(dir, tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateKeyNameMissingDir(t *testing.T) {
	if err := validateKeyName(filepath.Join(t.TempDir(), "nope"), "id_ed25519_x"); err != nil {
		t.Errorf("missing ssh dir should be fine, got %v", err)
	}
}

func TestAddIdentityFile(t *testing.T) {
	got := addIdentityFile(nil, "~/.ssh/k")
	if want := []string{"~/.ssh/k"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	got = addIdentityFile([]string{"~/.ssh/a"}, "~/.ssh/k")
	if want := []string{"~/.ssh/a", "~/.ssh/k"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	got = addIdentityFile([]string{"~/.ssh/k", "~/.ssh/a"}, "~/.ssh/k")
	if want := []string{"~/.ssh/k", "~/.ssh/a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("duplicate added: got %v, want %v", got, want)
	}
}

func TestDefaultKeyComment(t *testing.T) {
	if got := defaultKeyComment(); got == "" {
		t.Error("comment should never be empty")
	}
}
