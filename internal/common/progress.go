package common

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ProgressFunc reports the current loading step (e.g. while reading YAML
// data at startup) so the caller can render a progress indicator.
type ProgressFunc func(step string, current, total int)

// ConsoleProgress returns a ProgressFunc that prints a single updating
// progress line (bar + step name + elapsed time) to stdout. It is meant to
// be used before the tview application takes over the terminal: tview
// switches to the alternate screen buffer, so this output is hidden once
// the UI starts and reappears in the scrollback after the app quits.
func ConsoleProgress(label string) ProgressFunc {
	start := time.Now()
	return func(step string, current, total int) {
		const width = 24
		filled := 0
		if total > 0 {
			filled = width * current / total
			if filled > width {
				filled = width
			}
		}
		bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
		fmt.Fprintf(os.Stdout, "\r\033[K[%s] %s: %s (%d/%d) %.1fs",
			bar, label, step, current, total, time.Since(start).Seconds())
		if current >= total {
			fmt.Fprint(os.Stdout, "\r\033[K")
		}
	}
}
