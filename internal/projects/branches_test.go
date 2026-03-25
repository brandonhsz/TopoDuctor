package projects

import (
	"reflect"
	"testing"
)

func TestApplyPreferredFirst(t *testing.T) {
	all := []string{"z", "main", "develop", "origin/main"}
	pref := []string{"develop", "main", "missing"}
	got := ApplyPreferredFirst(all, pref)
	want := []string{"develop", "main", "z", "origin/main"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
