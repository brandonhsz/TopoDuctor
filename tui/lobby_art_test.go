package tui

import "testing"

func TestFitTopoASCII_scaleDown(t *testing.T) {
	art := "abcd\nefgh"
	out := fitTopoASCII(art, 2, 2)
	if len(out) != 2 {
		t.Fatalf("height: got %d", len(out))
	}
	if len([]rune(out[0])) != 2 {
		t.Fatalf("width: %q", out[0])
	}
}

func TestFitTopoASCII_center(t *testing.T) {
	art := "ab"
	out := fitTopoASCII(art, 6, 5)
	if len(out) != 5 {
		t.Fatalf("height: %d", len(out))
	}
}
