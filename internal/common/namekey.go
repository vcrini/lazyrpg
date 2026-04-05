package common

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// BindRandomNameInput attaches Ctrl+N random-name generation to field.
// The field's label is updated with a " (^N)" hint to show the shortcut.
func BindRandomNameInput(field *tview.InputField, generate func() string) {
	label := strings.TrimRight(field.GetLabel(), " ")
	field.SetLabel(label + " (^N) ")
	field.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlN {
			field.SetText(generate())
			return nil
		}
		return event
	})
}
