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
	"testing"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// formDriver sends keys to whatever has focus, the way tview's event loop
// does, so dialogs can be tested without a terminal.
type formDriver struct{ app *tview.Application }

func newFormDriver(open func(t *tui)) formDriver {
	app := tview.NewApplication()
	open(&tui{app: app, root: tview.NewFlex()})
	return formDriver{app: app}
}

func (d formDriver) press(keys ...tcell.Key) {
	for _, k := range keys {
		if h := d.app.GetFocus().InputHandler(); h != nil {
			h(tcell.NewEventKey(k, 0, tcell.ModNone), func(p tview.Primitive) { d.app.SetFocus(p) })
		}
	}
}

// pick opens the focused dropdown and selects the option at index.
func (d formDriver) pick(index int) {
	d.press(tcell.KeyEnter, tcell.KeyHome)
	for i := 0; i < index; i++ {
		d.press(tcell.KeyDown)
	}
	d.press(tcell.KeyEnter)
}

func (d formDriver) focused() string {
	switch p := d.app.GetFocus().(type) {
	case *tview.InputField:
		return p.GetLabel()
	case *tview.DropDown:
		return p.GetLabel()
	case *tview.Checkbox:
		return p.GetLabel()
	case *tview.Button:
		return p.GetLabel()
	}
	return ""
}

func TestTransferFormFocusAfterChoosingMode(t *testing.T) {
	open := func(t *tui) { t.showFileTransferForm(domain.Server{Alias: "web1"}) }
	tests := []struct {
		name  string
		modes []int
		want  string
	}{
		{"opened", nil, "Mode"},
		{"download", []int{transferDownload}, "Local path"},
		{"sftp", []int{transferSFTP}, "Start"},
		{"sftp then upload", []int{transferSFTP, transferUpload}, "Local path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newFormDriver(open)
			for _, mode := range tt.modes {
				for d.focused() != "Mode" {
					d.press(tcell.KeyBacktab)
				}
				d.pick(mode)
			}
			if got := d.focused(); got != tt.want {
				t.Errorf("focus = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInstallKeyFormFocusAfterChoosingKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "id_ed25519.pub"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	open := func(t *tui) { t.showInstallKeyForm(domain.Server{Alias: "web1"}) }

	// Options: 0 = ~/.ssh/id_ed25519.pub, 1 = paste, 2 = generate.
	tests := []struct {
		name    string
		options []int
		want    string
	}{
		{"opened", nil, "Key"},
		{"existing key", []int{0}, "Install"},
		{"paste", []int{1}, "Public key"},
		{"generate", []int{2}, "Name"},
		{"generate then paste", []int{2, 1}, "Public key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newFormDriver(open)
			for _, option := range tt.options {
				for d.focused() != "Key" {
					d.press(tcell.KeyBacktab)
				}
				d.pick(option)
			}
			if got := d.focused(); got != tt.want {
				t.Errorf("focus = %q, want %q", got, tt.want)
			}
		})
	}
}
