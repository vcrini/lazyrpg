package swade

func (ui *tviewUI) buildTimerOverlay() {
	// Bar widget is created by common.NewTurnTimer; nothing else to initialize.
}

func (ui *tviewUI) startTurnTimer() {
	ui.timer.Start(ui.app,
		func() { // show: insert timer bar before status
			ui.mainFlex.RemoveItem(ui.status)
			ui.mainFlex.AddItem(ui.timer.Bar, 3, 0, false)
			ui.mainFlex.AddItem(ui.status, 1, 0, false)
		},
		func() { // hide: remove timer bar
			ui.mainFlex.RemoveItem(ui.timer.Bar)
		},
	)
}

func (ui *tviewUI) stopTurnTimer() {
	ui.timer.Stop(ui.app, func() {
		ui.mainFlex.RemoveItem(ui.timer.Bar)
	})
}
