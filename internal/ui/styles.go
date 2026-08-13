package ui

import "github.com/charmbracelet/lipgloss"

// Shared styles. Panes share one frame size (border + padding) so App's
// pane-width split lines up with what each submodel actually renders at —
// a mismatch here is what made the library list collapse to nothing while
// cover art overflowed its column.
var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	metaStyle  = lipgloss.NewStyle().Faint(true)

	paneBorderColor    = lipgloss.Color("240")
	focusedBorderColor = lipgloss.Color("212")

	// paneStyle wraps the library column, unfocused by default; App swaps
	// in focusedBorderColor when the library pane has keyboard focus.
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(paneBorderColor).
			Padding(0, 1)

	// borderStyle is used by the first-run setup form (firstrun.go), the
	// one remaining full panel with padding-1,2 proportions.
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(paneBorderColor).
			Padding(1, 2)

	// barStyle wraps the bottom now-playing bar: a single thin horizontal
	// strip, so it gets less vertical padding than a full panel. Its
	// border color is overridden per-render, same as paneStyle.
	barStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(paneBorderColor).
			Padding(0, 1)

	// centeredStyle center-aligns multi-line text within whatever width
	// it's rendered at — used for the fullscreen lyrics block.
	centeredStyle = lipgloss.NewStyle().Align(lipgloss.Center)
)
