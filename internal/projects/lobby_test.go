package projects

import "testing"

func TestShouldShowLobby_emptyPaths(t *testing.T) {
	if !ShouldShowLobby("/any", nil) {
		t.Fatal("empty paths → lobby")
	}
}
