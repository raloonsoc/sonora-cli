package lyrics

import "testing"

func TestActiveLine(t *testing.T) {
	synced := Lyrics{
		Synced: true,
		Lines: []Line{
			{StartMS: 0, Text: "line0"},
			{StartMS: 1000, Text: "line1"},
			{StartMS: 2000, Text: "line2"},
			{StartMS: 3000, Text: "line3"},
		},
	}

	tests := []struct {
		name       string
		lyrics     Lyrics
		positionMS int
		want       int
	}{
		{"exact match on a timestamp", synced, 1000, 1},
		{"between two timestamps", synced, 1500, 1},
		{"at the very first timestamp", synced, 0, 0},
		{"before the first line", synced, -1, -1},
		{"at the last timestamp", synced, 3000, 3},
		{"after the last line", synced, 999999, 3},
		{"unsynced lyrics never have an active line", Lyrics{Synced: false, Lines: synced.Lines}, 1500, -1},
		{"empty lyrics", Lyrics{Synced: true}, 1500, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ActiveLine(tt.lyrics, tt.positionMS)
			if got != tt.want {
				t.Errorf("ActiveLine(%dms) = %d, want %d", tt.positionMS, got, tt.want)
			}
		})
	}
}

func TestActiveLine_duplicateTimestamps(t *testing.T) {
	// Duplicate stamps (e.g. a backing-vocal line sharing a timestamp with
	// the lead) should resolve to the last line at that timestamp, matching
	// sort.Search's behavior on a non-strictly-increasing sequence.
	l := Lyrics{
		Synced: true,
		Lines: []Line{
			{StartMS: 1000, Text: "lead"},
			{StartMS: 1000, Text: "backing"},
			{StartMS: 2000, Text: "next"},
		},
	}

	got := ActiveLine(l, 1000)
	if got != 1 {
		t.Errorf("ActiveLine(1000ms) = %d, want 1 (last line at that timestamp)", got)
	}
}
