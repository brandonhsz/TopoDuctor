package git

import (
	"reflect"
	"testing"

	"github.com/macpro/topoductor/internal/worktree"
)

func TestParsePorcelain(t *testing.T) {
	input := `worktree /home/user/proj
HEAD 1111111111111111111111111111111111111111
branch refs/heads/main

worktree /home/user/proj-wt-abc
HEAD 2222222222222222222222222222222222222222
branch refs/heads/feature/foo

worktree /tmp/detached
HEAD 3333333333333333333333333333333333333333
detached
`

	got, err := ParsePorcelain(input)
	if err != nil {
		t.Fatal(err)
	}
	want := []worktree.Worktree{
		{Path: "/home/user/proj", Head: "1111111111111111111111111111111111111111", Branch: "main"},
		{Path: "/home/user/proj-wt-abc", Head: "2222222222222222222222222222222222222222", Branch: "feature/foo"},
		{Path: "/tmp/detached", Head: "3333333333333333333333333333333333333333", Branch: ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parse mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestParsePorcelain_empty(t *testing.T) {
	got, err := ParsePorcelain("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %#v", got)
	}
}
