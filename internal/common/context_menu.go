package common

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ContextItem is a single entry in a context menu.
type ContextItem struct {
	Label   string
	Handler func()
}

// ContextMenuState holds the render and interaction state for an open context menu.
type ContextMenuState struct {
	Items       []ContextItem
	Selected    int
	X, Y        int
	Width       int
	Height      int
	ReturnFocus tview.Primitive
}

// DrawContextMenu renders menu onto screen. It is a no-op when m is nil.
func DrawContextMenu(m *ContextMenuState, screen tcell.Screen) {
	if m == nil {
		return
	}
	borderSt := tcell.StyleDefault.Foreground(tcell.ColorGold).Background(tcell.ColorBlack)
	normalSt := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	selectedSt := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGold)

	// Top border with title
	screen.SetContent(m.X, m.Y, '┌', nil, borderSt)
	for i := 1; i < m.Width-1; i++ {
		screen.SetContent(m.X+i, m.Y, '─', nil, borderSt)
	}
	screen.SetContent(m.X+m.Width-1, m.Y, '┐', nil, borderSt)
	title := " Context Menu "
	titleX := m.X + (m.Width-len(title))/2
	for i, ch := range title {
		screen.SetContent(titleX+i, m.Y, ch, nil, borderSt)
	}

	// Item rows
	innerW := m.Width - 2
	for i, item := range m.Items {
		y := m.Y + 1 + i
		st := normalSt
		if i == m.Selected {
			st = selectedSt
		}
		screen.SetContent(m.X, y, '│', nil, borderSt)
		runes := []rune(" " + item.Label)
		for j := 0; j < innerW; j++ {
			ch := ' '
			if j < len(runes) {
				ch = runes[j]
			}
			screen.SetContent(m.X+1+j, y, ch, nil, st)
		}
		screen.SetContent(m.X+m.Width-1, y, '│', nil, borderSt)
	}

	// Bottom border
	bY := m.Y + 1 + len(m.Items)
	screen.SetContent(m.X, bY, '└', nil, borderSt)
	for i := 1; i < m.Width-1; i++ {
		screen.SetContent(m.X+i, bY, '─', nil, borderSt)
	}
	screen.SetContent(m.X+m.Width-1, bY, '┘', nil, borderSt)
}

// CloseContextMenu clears the active context menu and restores focus.
func CloseContextMenu(menu **ContextMenuState, app *tview.Application) {
	if *menu == nil {
		return
	}
	returnFocus := (*menu).ReturnFocus
	*menu = nil
	app.SetFocus(returnFocus)
}

// ShowContextMenu opens a context menu anchored near (clickCol, clickRow).
// It uses pages to obtain the available screen area for overflow clamping.
func ShowContextMenu(menu **ContextMenuState, items []ContextItem, returnFocus tview.Primitive, clickCol, clickRow int, pages *tview.Pages) {
	maxLen := 0
	for _, item := range items {
		if l := len([]rune(item.Label)); l > maxLen {
			maxLen = l
		}
	}
	width := maxLen + 4
	if width < 30 {
		width = 30
	}
	height := len(items) + 2

	_, _, screenW, screenH := pages.GetRect()
	menuX := clickCol
	menuY := clickRow + 1
	if menuX+width > screenW {
		menuX = screenW - width
	}
	if menuX < 0 {
		menuX = 0
	}
	if menuY+height > screenH {
		menuY = clickRow - height
	}
	if menuY < 0 {
		menuY = 0
	}

	*menu = &ContextMenuState{
		Items:       items,
		Selected:    0,
		X:           menuX,
		Y:           menuY,
		Width:       width,
		Height:      height,
		ReturnFocus: returnFocus,
	}
}
