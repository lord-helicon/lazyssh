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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/rivo/tview"
)

const pasteKeyOption = "Paste a key…"

func (t *tui) handleInstallKey() {
	server, ok := t.serverList.GetSelectedServer()
	if !ok {
		return
	}
	if _, err := exec.LookPath("ssh-copy-id"); err != nil {
		t.showStatusTempColor("ssh-copy-id not found; install OpenSSH client tools", "#FF6B6B")
		return
	}
	t.showInstallKeyForm(server)
}

// showInstallKeyForm lets the user pick a public key from ~/.ssh or paste one,
// then installs it on the server with ssh-copy-id.
func (t *tui) showInstallKeyForm(server domain.Server) {
	home, _ := os.UserHomeDir()
	keys := listPublicKeys(filepath.Join(home, ".ssh"))
	options := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		options = append(options, displayPath(key, home))
	}
	options = append(options, pasteKeyOption)

	selected := preselectKey(keys, server.IdentityFiles, home)
	pasted := ""

	form := tview.NewForm()

	// The status bar is hidden while the form is shown, so messages go below it.
	// It sits outside the form so it doesn't take a Tab stop.
	message := tview.NewTextView().SetDynamicColors(true)
	message.SetBorderPadding(0, 0, 2, 2)
	if len(keys) == 0 {
		message.SetText("[#AAAAAA]No public keys found in ~/.ssh[-]")
	}

	keyField := tview.NewInputField().
		SetLabel("Public key").
		SetFieldWidth(60).
		SetChangedFunc(func(text string) { pasted = text })
	keyDropDown := tview.NewDropDown().SetLabel("Key")
	keyDropDown.SetOptions(options, func(_ string, index int) {
		selected = index
		keyField.SetDisabled(index != len(keys))
	})
	keyDropDown.SetCurrentOption(selected)

	form.AddFormItem(keyDropDown)
	form.AddFormItem(keyField)

	alias := server.Alias
	form.AddButton("Install", func() {
		if selected < len(keys) {
			key := keys[selected]
			_ = t.runSuspended("Key installed on "+alias, "Key install on "+alias+" failed", func() error {
				return t.serverService.InstallPublicKey(alias, key)
			})
			return
		}
		if err := validatePublicKey(pasted); err != nil {
			message.SetText("[#FF6B6B]Invalid public key: " + tview.Escape(err.Error()) + "[-]")
			return
		}
		key := pasted
		_ = t.runSuspended("Key installed on "+alias, "Key install on "+alias+" failed", func() error {
			return t.serverService.InstallPublicKeyContent(alias, key)
		})
	})
	form.AddButton("Cancel", func() { t.returnToMain() })
	form.SetCancelFunc(func() { t.returnToMain() })

	dialog := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true).
		AddItem(message, 1, 0, false)
	dialog.SetBorder(true).
		SetTitle(fmt.Sprintf(" Install SSH key: %s ", server.Alias)).
		SetTitleAlign(tview.AlignCenter)

	t.app.SetRoot(dialog, true)
	t.app.SetFocus(form)
}

// listPublicKeys returns the .pub files in dir, sorted by name.
// Private keys are never opened.
func listPublicKeys(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var keys []string
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".pub") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			keys = append(keys, path)
		}
	}
	return keys
}

// preselectKey returns the index of the first key whose private key is one of
// the server's IdentityFiles, or 0 when none matches.
func preselectKey(keys, identityFiles []string, home string) int {
	for _, identityFile := range identityFiles {
		if rest, ok := strings.CutPrefix(identityFile, "~/"); ok {
			identityFile = filepath.Join(home, rest)
		}
		pub := filepath.Clean(identityFile) + ".pub"
		for i, key := range keys {
			if filepath.Clean(key) == pub {
				return i
			}
		}
	}
	return 0
}

// displayPath shortens paths under home to ~/... for display.
func displayPath(path, home string) string {
	if home != "" {
		if rest, ok := strings.CutPrefix(path, home+string(filepath.Separator)); ok {
			return "~/" + filepath.ToSlash(rest)
		}
	}
	return path
}

// validatePublicKey checks that key looks like a single OpenSSH public key line.
func validatePublicKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("paste a public key first")
	}
	if strings.ContainsAny(key, "\r\n") {
		return errors.New("paste a single key line")
	}
	if strings.Contains(key, "PRIVATE KEY") {
		return errors.New("this is a private key; paste the .pub key instead")
	}
	fields := strings.Fields(key)
	keyType := fields[0]
	if !strings.HasPrefix(keyType, "ssh-") && !strings.HasPrefix(keyType, "ecdsa-sha2-") && !strings.HasPrefix(keyType, "sk-") {
		return fmt.Errorf("unknown key type %q", keyType)
	}
	if len(fields) < 2 {
		return errors.New("key data is missing")
	}
	blob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return errors.New("key data is not valid base64")
	}
	// The key data starts with the key type again, as an SSH string
	// (4-byte big-endian length, then the bytes).
	if len(blob) < 4 {
		return errors.New("key data is too short")
	}
	// Assembled from bytes so an oversized length can't wrap on 32-bit platforms.
	n := int(blob[0])<<24 | int(blob[1])<<16 | int(blob[2])<<8 | int(blob[3])
	if n < 0 || n > len(blob)-4 || string(blob[4:4+n]) != keyType {
		return errors.New("key data does not match the key type")
	}
	return nil
}
