package update

import "testing"

func TestNormalize(t *testing.T) {
	if got := Normalize(" v1.2.3 "); got != "v1.2.3" {
		t.Fatalf("got %q", got)
	}
	if got := Normalize("dev"); got != "v0.0.0" {
		t.Fatalf("got %q", got)
	}
	if !IsNewer("1.0.0", "v1.0.1") {
		t.Fatal("expected newer")
	}
	if IsNewer("1.0.1", "v1.0.1") {
		t.Fatal("expected same")
	}
}
