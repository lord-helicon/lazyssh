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
	"path/filepath"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/rivo/tview"
)

const (
	transferUpload = iota
	transferDownload
	transferSFTP
)

const maxPathSuggestions = 20

const remoteHint = "[#AAAAAA](empty remote path = home directory)[-]"

func (t *tui) handleFileTransfer() {
	server, ok := t.serverList.GetSelectedServer()
	if !ok {
		return
	}
	t.showFileTransferForm(server)
}

// showFileTransferForm asks what to transfer, then runs scp or sftp with the
// TUI suspended.
func (t *tui) showFileTransferForm(server domain.Server) {
	home, _ := os.UserHomeDir()
	alias := server.Alias
	mode := transferUpload

	form := tview.NewForm()

	// The status bar is hidden while the form is shown, so messages go below it.
	// It sits outside the form so it doesn't take a Tab stop.
	message := tview.NewTextView().SetDynamicColors(true)
	message.SetBorderPadding(0, 0, 2, 2)
	message.SetText(remoteHint)

	localField := tview.NewInputField().
		SetLabel("Local path").
		SetFieldWidth(60).
		SetAutocompleteFunc(func(text string) []string { return completeLocalPath(text, home) })
	remoteField := tview.NewInputField().
		SetLabel("Remote path").
		SetFieldWidth(60)
	recursiveBox := tview.NewCheckbox().SetLabel("Recursive (folders)")

	modeDropDown := tview.NewDropDown().SetLabel("Mode")
	modeDropDown.SetOptions(
		[]string{"Upload (local → server)", "Download (server → local)", "Open SFTP session"},
		func(_ string, index int) {
			mode = index
			sftp := index == transferSFTP
			localField.SetDisabled(sftp)
			remoteField.SetDisabled(sftp)
			recursiveBox.SetDisabled(sftp)
		})

	form.AddFormItem(modeDropDown)
	form.AddFormItem(localField)
	form.AddFormItem(remoteField)
	form.AddFormItem(recursiveBox)
	modeDropDown.SetCurrentOption(transferUpload)

	form.AddButton("Start", func() {
		remote := strings.TrimSpace(remoteField.GetText())
		local, recursive, err := validateTransfer(mode, localField.GetText(), remote, recursiveBox.IsChecked(), home)
		if err != nil {
			message.SetText("[#FF6B6B]" + tview.Escape(err.Error()) + "[-]")
			return
		}
		switch mode {
		case transferUpload:
			dest := remote
			if dest == "" {
				dest = "~"
			}
			_ = t.runSuspended(
				fmt.Sprintf("Uploaded %s to %s:%s", local, alias, dest),
				"Upload to "+alias+" failed",
				func() error { return t.serverService.UploadFiles(alias, local, remote, recursive) })
		case transferDownload:
			_ = t.runSuspended(
				fmt.Sprintf("Downloaded %s:%s to %s", alias, remote, local),
				"Download from "+alias+" failed",
				func() error { return t.serverService.DownloadFiles(alias, remote, local, recursive) })
		default:
			_ = t.runSuspended(
				"SFTP session to "+alias+" closed",
				"SFTP session to "+alias+" failed",
				func() error { return t.serverService.OpenSFTP(alias) })
		}
	})
	form.AddButton("Cancel", func() { t.returnToMain() })
	form.SetCancelFunc(func() { t.returnToMain() })

	dialog := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true).
		AddItem(message, 1, 0, false)
	dialog.SetBorder(true).
		SetTitle(fmt.Sprintf(" Transfer files: %s ", alias)).
		SetTitleAlign(tview.AlignCenter)

	t.app.SetRoot(dialog, true)
	t.app.SetFocus(form)
}

// expandLocalPath expands a leading "~/" to home.
func expandLocalPath(path, home string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok && home != "" {
		return filepath.Join(home, rest)
	}
	return path
}

// completeLocalPath suggests local paths for the typed text: the entries of
// its directory that start with its last element. Directories get a trailing
// "/", dotfiles are only offered when the typed element starts with a dot, and
// a typed leading "~/" is kept in the suggestions.
func completeLocalPath(text, home string) []string {
	if text == "" {
		return nil
	}
	cut := strings.LastIndexAny(text, "/"+string(filepath.Separator))
	typedDir, base := text[:cut+1], text[cut+1:]
	dir := "."
	if typedDir != "" {
		dir = expandLocalPath(typedDir, home)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	showHidden := strings.HasPrefix(base, ".")
	out := make([]string, 0, maxPathSuggestions)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, base) || (!showHidden && strings.HasPrefix(name, ".")) {
			continue
		}
		suggestion := typedDir + name
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && info.IsDir() {
			suggestion += "/"
		}
		out = append(out, suggestion)
		if len(out) == maxPathSuggestions {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// validateTransfer checks the form input for the given mode. It returns the
// local path with "~/" expanded and the effective recursive flag (forced on
// when uploading a directory).
func validateTransfer(mode int, local, remote string, recursive bool, home string) (string, bool, error) {
	if mode == transferSFTP {
		return "", false, nil
	}
	local = strings.TrimSpace(local)
	remote = strings.TrimSpace(remote)
	if local == "" {
		return "", false, errors.New("enter a local path")
	}
	local = expandLocalPath(local, home)

	switch mode {
	case transferUpload:
		info, err := os.Stat(local)
		if err != nil {
			return "", false, fmt.Errorf("local path does not exist: %s", local)
		}
		if info.IsDir() {
			recursive = true
		}
	case transferDownload:
		if remote == "" {
			return "", false, errors.New("enter the remote path to download")
		}
		if info, err := os.Stat(local); err == nil && info.IsDir() {
			break
		}
		parent := filepath.Dir(local)
		if info, err := os.Stat(parent); err != nil || !info.IsDir() {
			return "", false, fmt.Errorf("local directory does not exist: %s", parent)
		}
	}
	return local, recursive, nil
}
