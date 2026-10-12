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
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

const generateKeyOption = "Generate a new key…"

// keyTypes are the types offered by the generate option: ssh-keygen type and
// the label shown in the dropdown.
var keyTypes = []struct{ keyType, label string }{
	{"ed25519", "ed25519"},
	{"rsa", "rsa (4096 bits)"},
	{"ecdsa", "ecdsa"},
}

var keyNameRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// defaultKeyName returns id_<type>_<alias>, with characters outside
// [A-Za-z0-9._-] in the alias replaced by '_'.
func defaultKeyName(keyType, alias string) string {
	var b strings.Builder
	for _, r := range alias {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return "id_" + keyType + "_" + b.String()
}

// nameAfterTypeChange returns the name to show after the key type changed:
// the new default if name is still the old default, otherwise name unchanged.
func nameAfterTypeChange(name, alias, oldType, newType string) string {
	if name == defaultKeyName(oldType, alias) {
		return defaultKeyName(newType, alias)
	}
	return name
}

// validateKeyName checks a key file name for ~/.ssh and that neither the
// private nor the public key file exists in sshDir yet.
func validateKeyName(sshDir, name string) error {
	if name == "" {
		return errors.New("name is empty")
	}
	if !keyNameRe.MatchString(name) {
		return errors.New("name may only contain letters, digits, '.', '_' and '-'")
	}
	if strings.HasSuffix(name, ".pub") {
		return errors.New("name must not end in .pub")
	}
	for _, file := range []string{name, name + ".pub"} {
		if _, err := os.Lstat(filepath.Join(sshDir, file)); err == nil {
			return fmt.Errorf("~/.ssh/%s already exists", file)
		}
	}
	return nil
}

// addIdentityFile returns files with path appended unless already present.
func addIdentityFile(files []string, path string) []string {
	for _, f := range files {
		if f == path {
			return files
		}
	}
	return append(append([]string{}, files...), path)
}

// defaultKeyComment returns <user>@<hostname>, falling back to placeholders.
func defaultKeyComment() string {
	name := ""
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	if name == "" {
		name = os.Getenv("USER")
	}
	if name == "" {
		name = os.Getenv("USERNAME")
	}
	if name == "" {
		name = "user"
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "localhost"
	}
	return name + "@" + host
}

// generateAndInstallKey creates a key pair in sshDir, installs the public key
// on server and optionally adds the private key to the server's IdentityFiles.
// It runs with the TUI suspended (ssh-keygen asks for the passphrase itself).
func (t *tui) generateAndInstallKey(server domain.Server, sshDir, name, keyType, comment string, useKey bool) error {
	alias := server.Alias
	return t.runSuspended(
		fmt.Sprintf("Generated ~/.ssh/%s and installed it on %s", name, alias),
		"Key setup for "+alias+" failed",
		func() error {
			if err := os.MkdirAll(sshDir, 0o700); err != nil {
				return fmt.Errorf("create ~/.ssh: %w", err)
			}
			path := filepath.Join(sshDir, name)
			if err := t.serverService.GenerateKey(path, keyType, comment); err != nil {
				return err
			}
			if err := t.serverService.InstallPublicKey(alias, path+".pub"); err != nil {
				return fmt.Errorf("%w (the key was still created at ~/.ssh/%s)", err, name)
			}
			if !useKey {
				return nil
			}
			identity := "~/.ssh/" + name
			newServer := server
			newServer.IdentityFiles = addIdentityFile(server.IdentityFiles, identity)
			if err := t.serverService.UpdateServer(server, newServer); err != nil {
				return fmt.Errorf("key installed, but adding it as IdentityFile failed: %w", err)
			}
			return nil
		})
}
