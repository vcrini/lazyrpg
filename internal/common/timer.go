package common

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TurnTimer manages the visual turn timer shared by all systems.
type TurnTimer struct {
	Duration  int // seconds; 0 = disabled
	Running   bool
	Bar       *tview.TextView
	cancel    chan struct{}
	prevFocus tview.Primitive
}

// NewTurnTimer creates a TurnTimer with a pre-built bar widget.
func NewTurnTimer(duration int) *TurnTimer {
	bar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	bar.SetBorder(true).
		SetBorderColor(tcell.ColorGold).
		SetTitleColor(tcell.ColorGold)
	return &TurnTimer{Duration: duration, Bar: bar}
}

// TimerBarText renders the colored block progress bar string.
func TimerBarText(progress float64, remaining int, barWidth int) string {
	filled := int(progress * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	var color string
	switch {
	case progress < 0.5:
		color = "[green:black]"
	case progress < 0.75:
		color = "[yellow:black]"
	default:
		color = "[red:black]"
	}

	return fmt.Sprintf(" %s%s[-:-]%s  %ds ",
		color,
		strings.Repeat("█", filled),
		strings.Repeat("░", empty),
		remaining,
	)
}

// Start begins the timer. showFn is called immediately (in the caller's goroutine)
// to add the bar to the layout. hideFn is called when the timer expires or is stopped,
// and must NOT call SetFocus — focus restoration is handled internally via QueueUpdate.
func (t *TurnTimer) Start(app *tview.Application, showFn, hideFn func()) {
	if t.Duration <= 0 {
		return
	}
	t.Stop(app, hideFn)

	t.prevFocus = app.GetFocus()
	t.Running = true
	showFn()
	app.SetFocus(t.prevFocus)

	cancel := make(chan struct{})
	t.cancel = cancel
	start := time.Now()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-cancel:
				return
			case <-ticker.C:
				elapsed := time.Since(start).Seconds()
				remaining := float64(t.Duration) - elapsed
				if remaining <= 0 {
					app.QueueUpdateDraw(func() {
						t.Stop(app, hideFn)
					})
					return
				}
				progress := elapsed / float64(t.Duration)
				rem := int(remaining) + 1
				app.QueueUpdateDraw(func() {
					_, _, w, _ := t.Bar.GetRect()
					barW := w - 12
					if barW < 10 {
						barW = 30
					}
					t.Bar.SetTitle(fmt.Sprintf(" Turno: %ds ", rem))
					t.Bar.SetText(TimerBarText(progress, rem, barW))
				})
			}
		}
	}()
}

// Stop cancels the timer. hideFn removes the bar from the layout.
// Focus is restored to the primitive that was focused when Start was called.
// Focus restoration is scheduled via QueueUpdate so it runs after any
// layout redraws triggered by hideFn (e.g. tview Pages.AddPage focus side effects).
func (t *TurnTimer) Stop(app *tview.Application, hideFn func()) {
	if t.cancel != nil {
		close(t.cancel)
		t.cancel = nil
	}
	t.Running = false
	if hideFn != nil {
		hideFn()
	}
	if t.prevFocus != nil {
		app.SetFocus(t.prevFocus)
		t.prevFocus = nil
	}
}
