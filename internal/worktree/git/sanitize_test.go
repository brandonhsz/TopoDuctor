package git

import "testing"

func TestSanitizeWorktreeLabel(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"   ", ""},
		{"feature-1", "feature-1"},
		{"Mi Rama", "Mi-Rama"},
		{"a  b", "a-b"},
		{"foo@bar", "foo-bar"},
		{"-x-", "x"},
	}
	for _, tc := range tests {
		got := SanitizeWorktreeLabel(tc.in)
		if got != tc.want {
			t.Errorf("SanitizeWorktreeLabel(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
