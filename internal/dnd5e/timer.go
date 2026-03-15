package dnd5e

func (ui *UI) buildTimerOverlay() {
	// Bar widget is created by common.NewTurnTimer; nothing to add to pages.
}

func (ui *UI) startTurnTimer() {
	ui.timer.Start(ui.app,
		func() { // show: insert bar between content and status
			ui.mainFlex.RemoveItem(ui.status)
			ui.mainFlex.AddItem(ui.timer.Bar, 3, 0, false)
			ui.mainFlex.AddItem(ui.status, 1, 0, false)
		},
		func() { // hide: remove bar (status stays)
			ui.mainFlex.RemoveItem(ui.timer.Bar)
		},
	)
}

func (ui *UI) stopTurnTimer() {
	ui.timer.Stop(ui.app, func() {
		ui.mainFlex.RemoveItem(ui.timer.Bar)
	})
}
