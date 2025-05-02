/*
StyleHelpers.go

Style helpers for the TUI package
*/

package tui

func truncateString(s string, maxWidth int) string {
	runes := []rune(s)
	if len(runes) > maxWidth {
		return string(runes[:maxWidth-3]) + "..."
	}
	return s
}
