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

	// borderStyle wraps the now-playing column. Its border color is
	// overridden per-render: by the album's accent color when one has
	// resolved, or by focusedBorderColor while that pane has focus.
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(paneBorderColor).
			Padding(1, 2)
)
