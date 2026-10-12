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
	"bufio"
	"fmt"
	"os"
)

// runSuspended runs fn with the TUI suspended, so the external command it
// starts (ssh-copy-id, scp, sftp, ssh-keygen, ...) can use the terminal.
// If fn fails, the output stays on screen until the user presses Enter.
// Afterwards it returns to the main view and shows okMsg, or failMsg with
// the error, in the status bar. It returns fn's error.
func (t *tui) runSuspended(okMsg, failMsg string, fn func() error) error {
	var err error
	t.app.Suspend(func() {
		if err = fn(); err != nil {
			fmt.Printf("\n%s: %v\nPress Enter to return to lazyssh...", failMsg, err)
			_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		}
	})
	t.returnToMain()
	if err != nil {
		t.showStatusTempColor(failMsg+": "+err.Error(), "#FF6B6B")
		return err
	}
	t.showStatusTemp(okMsg)
	return nil
}
