package common

import (
	"github.com/rivo/tview"
)

// OpenHelpOverlay displays a help page over the current tview pages.
// content is the text to show (already built by the caller).
// If centered is true the modal is shown as a centred popup (1:5:1 ratio);
// otherwise it takes the full screen.
// Returns immediately if helpVisible is already true.
func OpenHelpOverlay(
	app *tview.Application,
	pages *tview.Pages,
	helpVisible *bool,
	helpReturnFocus *tview.Primitive,
	focus tview.Primitive,
	content string,
	centered bool,
) {
	if *helpVisible {
		return
	}
	*helpVisible = true
	*helpReturnFocus = focus

	text := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	text.SetBorder(true).SetTitle("Help")
	text.SetText(content)

	var modal tview.Primitive
	if centered {
		modal = tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 1, false).
				AddItem(text, 0, 5, true).
				AddItem(nil, 0, 1, false), 0, 5, true).
			AddItem(nil, 0, 1, false)
	} else {
		modal = tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(text, 0, 1, true)
	}

	pages.AddAndSwitchToPage("help", modal, true)
	app.SetFocus(text)
}

// CloseHelpOverlay removes the help page and restores focus.
// No-op if helpVisible is false.
func CloseHelpOverlay(
	app *tview.Application,
	pages *tview.Pages,
	helpVisible *bool,
	helpReturnFocus tview.Primitive,
) {
	if !*helpVisible {
		return
	}
	*helpVisible = false
	pages.RemovePage("help")
	if helpReturnFocus != nil {
		app.SetFocus(helpReturnFocus)
	}
}
