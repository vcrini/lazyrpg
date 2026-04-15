package common

import (
	"fmt"

	"github.com/rivo/tview"
)

// UpdateOrAppendDiceEntry updates or appends an entry in the dice log and sets
// the list's current item so that a subsequent renderDiceList() applies the
// gold highlight to the correct row.
//
//   - If idx >= 0 the entry at that index is replaced and the list current item
//     is set to idx.
//   - If idx < 0 the entry is appended; if maxLog > 0 the oldest entries are
//     trimmed to keep the log within that limit; the list current item is set to
//     the new last index.
//
// The caller must call renderDiceList() after this function returns.
func UpdateOrAppendDiceEntry(log *[]DiceResult, list *tview.List, maxLog int, idx int, entry DiceResult) {
	if idx >= 0 {
		(*log)[idx] = entry
		list.SetCurrentItem(idx)
		return
	}
	*log = append(*log, entry)
	if maxLog > 0 && len(*log) > maxLog {
		*log = (*log)[len(*log)-maxLog:]
	}
	list.SetCurrentItem(len(*log) - 1)
}

// DiceRenderOptions configures system-specific behaviour of RenderDiceList.
type DiceRenderOptions struct {
	// EmptyMsg is shown as a placeholder when the log is empty.
	// If empty, no item is added and the function returns immediately.
	EmptyMsg string

	// Prefix returns the row prefix string for 0-based index i (e.g. "1) " or "1 ").
	// If nil, defaults to fmt.Sprintf("%d) ", i+1).
	Prefix func(i int) string

	// StyleItem optionally applies tview colour tags to expr and output for
	// non-compact rows. i is the row index, current is the selected row index.
	// If nil, plain text is used.
	StyleItem func(i, current int, expr, output string) (styledExpr, styledOutput string)

	// StyleCompact optionally applies tview colour tags to texpr and final for
	// compact (truncated) rows.
	// If nil, plain text is used.
	StyleCompact func(i, current int, texpr, final string) (styledTexpr, styledFinal string)
}

// RenderDiceList populates list from log applying truncation and optional
// per-item styling. The caller must manage any render-lock around this call.
func RenderDiceList(list *tview.List, log []DiceResult, opts DiceRenderOptions) {
	current := list.GetCurrentItem()
	if current < 0 {
		current = 0
	}
	list.Clear()

	if len(log) == 0 {
		if opts.EmptyMsg != "" {
			list.AddItem(opts.EmptyMsg, "", 0, nil)
			list.SetCurrentItem(0)
		}
		return
	}

	prefix := opts.Prefix
	if prefix == nil {
		prefix = func(i int) string { return fmt.Sprintf("%d) ", i+1) }
	}

	_, _, diceW, _ := list.GetInnerRect()
	if diceW < 10 {
		diceW = 0 // layout not yet computed or implausibly small: skip truncation
	}

	const sep = " = "
	for i, row := range log {
		p := prefix(i)
		_, needsCompact := TruncateDiceExpr(p, row.Expression, sep+row.Output, diceW)
		var label string
		if needsCompact {
			final := ExtractFinalResult(row.Output)
			texpr, _ := TruncateDiceExpr(p, row.Expression, sep+"... "+final, diceW)
			exprDisplay, finalDisplay := texpr, final
			if opts.StyleCompact != nil {
				exprDisplay, finalDisplay = opts.StyleCompact(i, current, texpr, final)
			}
			label = BuildDiceLabel(p, exprDisplay, sep, true, finalDisplay)
		} else {
			exprDisplay, outDisplay := row.Expression, row.Output
			if opts.StyleItem != nil {
				exprDisplay, outDisplay = opts.StyleItem(i, current, row.Expression, row.Output)
			}
			label = BuildDiceLabel(p, exprDisplay, sep, false, outDisplay)
		}
		list.AddItem(label, "", 0, nil)
	}

	if current >= len(log) {
		current = len(log) - 1
	}
	list.SetCurrentItem(current)
}
