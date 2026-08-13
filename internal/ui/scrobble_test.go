package ui

import (
	"testing"
	"time"
)

func TestShouldSubmit(t *testing.T) {
	tests := []struct {
		name               string
		position, duration time.Duration
		want               bool
	}{
		{"below both thresholds", 30 * time.Second, 5 * time.Minute, false},
		{"past 50 percent, under 4 min", 2*time.Minute + 40*time.Second, 5 * time.Minute, true},
		{"exactly 50 percent", 150 * time.Second, 5 * time.Minute, true},
		{"past 4 min on a long track", 4*time.Minute + 1*time.Second, 20 * time.Minute, true},
		{"exactly 4 min", 4 * time.Minute, 20 * time.Minute, true},
		{"short track never reaches 4 min but crosses 50 percent", 90 * time.Second, 3 * time.Minute, true},
		{"zero duration never submits", 10 * time.Second, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSubmit(tt.position, tt.duration)
			if got != tt.want {
				t.Errorf("shouldSubmit(%s, %s) = %v, want %v", tt.position, tt.duration, got, tt.want)
			}
		})
	}
}
