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

package services

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.uber.org/zap"
)

func TestSSHCopyIDArgs(t *testing.T) {
	dir := t.TempDir()
	withPrivate := filepath.Join(dir, "id_ed25519.pub")
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	pubOnly := filepath.Join(dir, "colleague.pub")

	tests := []struct {
		name string
		path string
		want []string
	}{
		{"private key next to it", withPrivate, []string{"-i", withPrivate, "web1"}},
		{"no private key", pubOnly, []string{"-f", "-i", pubOnly, "web1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sshCopyIDArgs("web1", tt.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sshCopyIDArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInstallPublicKeyContentUsesPrivateTempFile(t *testing.T) {
	const key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIC9 someone@laptop"
	var gotArgs []string
	var keyPath string

	s := &serverService{logger: zap.NewNop().Sugar()}
	s.runInteractive = func(name string, args ...string) error {
		gotArgs = append([]string{name}, args...)
		keyPath = args[len(args)-2]
		data, err := os.ReadFile(keyPath)
		if err != nil {
			t.Fatalf("temp key file not readable during install: %v", err)
		}
		if string(data) != key+"\n" {
			t.Errorf("temp key file content = %q", data)
		}
		info, err := os.Stat(keyPath)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("temp key file perms = %o, want 600", perm)
		}
		return nil
	}

	if err := s.InstallPublicKeyContent("web1", "  "+key+"  \n"); err != nil {
		t.Fatal(err)
	}
	want := []string{"ssh-copy-id", "-f", "-i", keyPath, "web1"}
	if !reflect.DeepEqual(gotArgs, want) {
		t.Errorf("command = %v, want %v", gotArgs, want)
	}
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Errorf("temp key file %s was not removed", keyPath)
	}
}
