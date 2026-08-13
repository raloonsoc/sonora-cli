package ui

import "testing"

func TestWindowAround(t *testing.T) {
	tests := []struct {
		name               string
		center, total, sz  int
		wantStart, wantEnd int
	}{
		{"centered mid-list", 10, 20, 7, 7, 14},
		{"near start clamps to 0", 1, 20, 7, 0, 7},
		{"near end clamps to total", 18, 20, 7, 13, 20},
		{"no active line clamps center to 0", -1, 20, 7, 0, 7},
		{"window larger than total", 2, 4, 7, 0, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := windowAround(tt.center, tt.total, tt.sz)
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Errorf("windowAround(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.center, tt.total, tt.sz, gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestLyricsState_View_emptyWhenNoLines(t *testing.T) {
	s := lyricsState{}
	if got := s.View(1000); got != "" {
		t.Errorf("View() = %q, want empty for no lyrics", got)
	}
}
