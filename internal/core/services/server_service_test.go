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
	"errors"
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

func TestSCPArgs(t *testing.T) {
	tests := []struct {
		name string
		got  []string
		want []string
	}{
		{"upload", scpUploadArgs("web1", "/tmp/a.txt", "/srv/a.txt", false), []string{"/tmp/a.txt", "web1:/srv/a.txt"}},
		{"upload recursive", scpUploadArgs("web1", "/tmp/dir", "/srv", true), []string{"-r", "/tmp/dir", "web1:/srv"}},
		{"upload empty remote is home", scpUploadArgs("web1", "/tmp/a.txt", "", false), []string{"/tmp/a.txt", "web1:"}},
		{"upload local starting with dash", scpUploadArgs("web1", "-weird", "", false), []string{"./-weird", "web1:"}},
		{"download", scpDownloadArgs("web1", "/srv/a.txt", "/tmp/a.txt", false), []string{"web1:/srv/a.txt", "/tmp/a.txt"}},
		{"download recursive", scpDownloadArgs("web1", "logs", "/tmp", true), []string{"-r", "web1:logs", "/tmp"}},
		{"download local starting with dash", scpDownloadArgs("web1", "a", "-out", false), []string{"web1:a", "./-out"}},
		{"sftp", sftpArgs("web1"), []string{"web1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestTransferCommands(t *testing.T) {
	var got []string
	s := &serverService{logger: zap.NewNop().Sugar()}
	s.runInteractive = func(name string, args ...string) error {
		got = append([]string{name}, args...)
		return nil
	}

	if err := s.UploadFiles("web1", "/tmp/d", "", true); err != nil {
		t.Fatal(err)
	}
	if want := []string{"scp", "-r", "/tmp/d", "web1:"}; !reflect.DeepEqual(got, want) {
		t.Errorf("upload ran %v, want %v", got, want)
	}
	if err := s.DownloadFiles("web1", "/srv/a", "/tmp/a", false); err != nil {
		t.Fatal(err)
	}
	if want := []string{"scp", "web1:/srv/a", "/tmp/a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("download ran %v, want %v", got, want)
	}
	if err := s.OpenSFTP("web1"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"sftp", "web1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("sftp ran %v, want %v", got, want)
	}

	wantErr := errors.New("boom")
	s.runInteractive = func(string, ...string) error { return wantErr }
	if err := s.UploadFiles("web1", "/tmp/a", "", false); !errors.Is(err, wantErr) {
		t.Errorf("UploadFiles error = %v, want %v", err, wantErr)
	}
	if err := s.DownloadFiles("web1", "a", "/tmp/a", false); !errors.Is(err, wantErr) {
		t.Errorf("DownloadFiles error = %v, want %v", err, wantErr)
	}
	if err := s.OpenSFTP("web1"); !errors.Is(err, wantErr) {
		t.Errorf("OpenSFTP error = %v, want %v", err, wantErr)
	}
}

func TestSSHKeygenArgs(t *testing.T) {
	tests := []struct {
		name    string
		keyType string
		want    []string
		wantErr bool
	}{
		{"ed25519", "ed25519", []string{"-t", "ed25519", "-f", "/k/id", "-C", "me@host"}, false},
		{"rsa", "rsa", []string{"-t", "rsa", "-b", "4096", "-f", "/k/id", "-C", "me@host"}, false},
		{"ecdsa", "ecdsa", []string{"-t", "ecdsa", "-f", "/k/id", "-C", "me@host"}, false},
		{"unknown", "dsa", nil, true},
		{"empty", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sshKeygenArgs("/k/id", tt.keyType, "me@host")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sshKeygenArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateKeyRunsSSHKeygen(t *testing.T) {
	var got []string
	s := &serverService{logger: zap.NewNop().Sugar()}
	s.runInteractive = func(name string, args ...string) error {
		got = append([]string{name}, args...)
		return nil
	}
	if err := s.GenerateKey("/k/id", "ed25519", "me@host"); err != nil {
		t.Fatal(err)
	}
	want := []string{"ssh-keygen", "-t", "ed25519", "-f", "/k/id", "-C", "me@host"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("command = %v, want %v", got, want)
	}

	if err := s.GenerateKey("/k/id", "bogus", "c"); err == nil {
		t.Error("expected error for unknown key type")
	}
}
