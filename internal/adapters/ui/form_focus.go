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

import "github.com/rivo/tview"

// focusFormItem moves focus to the form item with the given label, or to the
// button with that label. Calling SetDisabled on a form item makes tview
// repeat the form's last focus move (once per call), so after a dropdown
// enables or disables other fields, focus has to be set explicitly.
func (t *tui) focusFormItem(form *tview.Form, label string) {
	index := form.GetFormItemIndex(label)
	if index < 0 {
		button := form.GetButtonIndex(label)
		if button < 0 {
			return
		}
		index = form.GetFormItemCount() + button
	}
	form.SetFocus(index)
	t.app.SetFocus(form)
}
