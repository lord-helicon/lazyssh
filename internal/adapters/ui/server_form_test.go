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
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestIsPrevTabKey(t *testing.T) {
	tests := []struct {
		name  string
		event *tcell.EventKey
		goos  string
		want  bool
	}{
		{"unix ctrl+h (0x08)", tcell.NewEventKey(tcell.KeyCtrlH, 0, tcell.ModNone), "linux", true},
		{"unix backspace (DEL)", tcell.NewEventKey(tcell.KeyBackspace2, 0, tcell.ModNone), "darwin", false},
		{"windows backspace", tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone), "windows", false},
		{"windows ctrl+h", tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModCtrl), "windows", true},
		{"other key", tcell.NewEventKey(tcell.KeyCtrlL, 0, tcell.ModCtrl), "linux", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPrevTabKey(tt.event, tt.goos); got != tt.want {
				t.Errorf("isPrevTabKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
