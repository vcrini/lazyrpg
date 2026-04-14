package common

import "github.com/rivo/tview"

// CloseModal removes the active modal page and resets modal state.
// modalConfirmFunc is cleared to nil on close.
func CloseModal(pages *tview.Pages, modalVisible *bool, modalName *string, modalConfirmFunc *func()) {
	if !*modalVisible {
		return
	}
	if *modalName != "" {
		pages.RemovePage(*modalName)
	}
	*modalVisible = false
	*modalName = ""
	*modalConfirmFunc = nil
}
