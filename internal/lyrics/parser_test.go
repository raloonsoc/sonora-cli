package lyrics

import (
	"reflect"
	"testing"

	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

func TestParse_structuredSortsByTimestamp(t *testing.T) {
	list := &subsonic.LyricsList{
		StructuredLyrics: []subsonic.StructuredLyrics{
			{
				Synced: true,
				Line: []subsonic.LyricLine{
					{Start: 2000, Value: "second"},
					{Start: 0, Value: "first"},
					{Start: 1000, Value: "middle"},
				},
			},
		},
	}

	got := Parse(list, nil)
	if !got.Synced {
		t.Fatal("expected Synced to be true")
	}

	want := []Line{
		{StartMS: 0, Text: "first"},
		{StartMS: 1000, Text: "middle"},
		{StartMS: 2000, Text: "second"},
	}
	if !reflect.DeepEqual(got.Lines, want) {
		t.Errorf("Lines = %+v, want %+v", got.Lines, want)
	}
}

func TestParse_fallsBackToPlainWhenStructuredEmpty(t *testing.T) {
	list := &subsonic.LyricsList{}
	plain := &subsonic.LyricsPlain{Value: "line one\nline two\nline three"}

	got := Parse(list, plain)
	if got.Synced {
		t.Fatal("expected Synced to be false for plain-text fallback")
	}

	want := []Line{{Text: "line one"}, {Text: "line two"}, {Text: "line three"}}
	if !reflect.DeepEqual(got.Lines, want) {
		t.Errorf("Lines = %+v, want %+v", got.Lines, want)
	}
}

func TestParse_bothEmptyYieldsNoLines(t *testing.T) {
	got := Parse(&subsonic.LyricsList{}, &subsonic.LyricsPlain{})
	if len(got.Lines) != 0 {
		t.Errorf("Lines = %+v, want empty", got.Lines)
	}
}

func TestParse_nilInputs(t *testing.T) {
	got := Parse(nil, nil)
	if len(got.Lines) != 0 {
		t.Errorf("Lines = %+v, want empty", got.Lines)
	}
}

func TestParsePlain_handlesCRLF(t *testing.T) {
	got := parsePlain("one\r\ntwo\r\nthree")
	want := []Line{{Text: "one"}, {Text: "two"}, {Text: "three"}}
	if !reflect.DeepEqual(got.Lines, want) {
		t.Errorf("Lines = %+v, want %+v", got.Lines, want)
	}
}
