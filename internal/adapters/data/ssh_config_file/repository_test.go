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

package ssh_config_file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func newTestRepo(t *testing.T, configPath string) *Repository {
	t.Helper()
	meta := filepath.Join(t.TempDir(), "metadata.json")
	return NewRepository(zap.NewNop().Sugar(), configPath, meta).(*Repository)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestAddServerKeepsSymlinkedConfig(t *testing.T) {
	sshDir := t.TempDir()
	dotfiles := t.TempDir()
	target := filepath.Join(dotfiles, "ssh_config")
	link := filepath.Join(sshDir, "config")
	writeFile(t, target, "Host a\n    HostName 10.0.0.1\n")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	repo := newTestRepo(t, link)
	if err := repo.AddServer(domain.Server{Alias: "b", Host: "10.0.0.2"}); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("config symlink was replaced by a regular file")
	}
	if got := readFile(t, target); !strings.Contains(got, "Host b") {
		t.Errorf("symlink target was not updated:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(sshDir, OriginalBackupName)); err != nil {
		t.Errorf("expected original backup next to the symlink: %v", err)
	}
	entries, err := os.ReadDir(dotfiles)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected only the config in the symlink target dir, got %d entries", len(entries))
	}
}
