package common

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// UpdateOrAppendDiceEntry updates or appends an entry in the dice log and
// returns the log index that should become the current selection (the
// updated entry's index, or the new last index on append). The caller must
// pass this index to RenderDiceList (via renderDiceList) so the list's
// current item is set correctly; the list itself is not touched here since
// its item count may no longer match the log length (wrapped entries occupy
// two rows).
//
//   - If idx >= 0 the entry at that index is replaced.
//   - If idx < 0 the entry is appended; if maxLog > 0 the oldest entries are
//     trimmed to keep the log within that limit.
func UpdateOrAppendDiceEntry(log *[]DiceResult, maxLog int, idx int, entry DiceResult) int {
	if idx >= 0 {
		(*log)[idx] = entry
		return idx
	}
	*log = append(*log, entry)
	if maxLog > 0 && len(*log) > maxLog {
		*log = (*log)[len(*log)-maxLog:]
	}
	return len(*log) - 1
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
	// row i (log index). current is the current log index. If nil, plain
	// text is used.
	StyleItem func(i, current int, expr, output string) (styledExpr, styledOutput string)
}

// LogIndexForItem translates a tview.List item index into the corresponding
// dice-log index, using the mapping most recently returned by RenderDiceList.
// Entries too wide to fit on one line occupy two consecutive list items that
// both map to the same log index. Out-of-range item indices are returned
// unchanged.
func LogIndexForItem(logForItem []int, itemIndex int) int {
	if itemIndex < 0 || itemIndex >= len(logForItem) {
		return itemIndex
	}
	return logForItem[itemIndex]
}

// RenderDiceList populates list from log applying optional per-item styling.
// Entries whose "prefix + expression + = + output" doesn't fit within the
// list's inner width are split across two list rows (expression, then the
// result indented on the next row) instead of being truncated.
//
// currentLog is the dice-log index that should end up selected; it is
// typically obtained by translating the list's previous current item through
// LogIndexForItem before calling this function.
//
// The caller must manage any render-lock around this call. Returns the
// logForItem mapping to keep for subsequent LogIndexForItem calls.
func RenderDiceList(list *tview.List, log []DiceResult, currentLog int, opts DiceRenderOptions) []int {
	list.Clear()

	if len(log) == 0 {
		if opts.EmptyMsg != "" {
			list.AddItem(opts.EmptyMsg, "", 0, nil)
			list.SetCurrentItem(0)
		}
		return nil
	}

	prefix := opts.Prefix
	if prefix == nil {
		prefix = func(i int) string { return fmt.Sprintf("%d) ", i+1) }
	}

	_, _, diceW, _ := list.GetInnerRect()
	if diceW < 10 {
		diceW = 0 // layout not yet computed or implausibly small: skip wrapping
	}

	const sep = " = "
	logForItem := make([]int, 0, len(log))
	for i, row := range log {
		p := prefix(i)
		exprDisplay, outDisplay := row.Expression, row.Output
		if opts.StyleItem != nil {
			exprDisplay, outDisplay = opts.StyleItem(i, currentLog, row.Expression, row.Output)
		}

		oneLine := p + row.Expression + sep + row.Output
		if diceW <= 0 || len([]rune(oneLine)) <= diceW {
			list.AddItem(p+exprDisplay+sep+outDisplay, "", 0, nil)
			logForItem = append(logForItem, i)
			continue
		}

		list.AddItem(p+exprDisplay, "", 0, nil)
		logForItem = append(logForItem, i)
		indent := strings.Repeat(" ", len([]rune(p)))
		list.AddItem(indent+"= "+outDisplay, "", 0, nil)
		logForItem = append(logForItem, i)
	}

	if currentLog < 0 {
		currentLog = 0
	}
	if currentLog >= len(log) {
		currentLog = len(log) - 1
	}
	itemIdx := 0
	for idx, l := range logForItem {
		if l == currentLog {
			itemIdx = idx
			break
		}
	}
	list.SetCurrentItem(itemIdx)
	return logForItem
}
