// Package artwork renders album cover art in the terminal: a native
// graphics protocol when the terminal supports one, colored ASCII as a
// fallback everywhere else (SPECS §6).
package artwork

import "github.com/BourgeoisBear/rasterm"

// TermType is the terminal's detected graphics capability.
type TermType int

const (
	TermUnknown TermType = iota
	TermKitty
	TermITerm2
	TermSixel
)

// DetectTermType probes the terminal once for its graphics capability.
// Callers should cache the result for the session — re-detecting per
// render adds latency for no benefit (SPECS §6.1).
func DetectTermType() TermType {
	if rasterm.IsKittyCapable() {
		return TermKitty
	}
	if rasterm.IsItermCapable() {
		return TermITerm2
	}
	if ok, _ := rasterm.IsSixelCapable(); ok {
		return TermSixel
	}
	return TermUnknown
}
