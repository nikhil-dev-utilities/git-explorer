package tui

import "testing"

func TestDetailForWidth(t *testing.T) {
	tests := []struct {
		repoWidth int
		want      repoDetail
	}{
		{200, repoDetailFull},
		{55, repoDetailFull},
		{54, repoDetailNoDate},
		{46, repoDetailNoDate},
		{45, repoDetailNameOnly},
		{0, repoDetailNameOnly},
	}
	for _, tt := range tests {
		if got := detailForWidth(tt.repoWidth); got != tt.want {
			t.Errorf("detailForWidth(%d) = %v, want %v", tt.repoWidth, got, tt.want)
		}
	}
}
