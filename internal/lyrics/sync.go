package lyrics

import "sort"

// ActiveLine returns the index into l.Lines whose timestamp is the latest
// one at or before positionMS, via binary search (SPECS §8) rather than a
// linear scan — this runs on every ~500ms position tick.
//
// Returns -1 when l is unsynced, empty, or positionMS is before the first
// line's timestamp (nothing is active yet).
func ActiveLine(l Lyrics, positionMS int) int {
	if !l.Synced || len(l.Lines) == 0 {
		return -1
	}

	// sort.Search finds the first index where Lines[i].StartMS > positionMS;
	// the active line is the one just before it.
	i := sort.Search(len(l.Lines), func(i int) bool {
		return l.Lines[i].StartMS > positionMS
	})

	if i == 0 {
		return -1 // position is before the first line
	}
	return i - 1
}
