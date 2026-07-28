package report

import "testing"

// The output says "1 violation", not "1 violations".
func TestCountPluralizes(t *testing.T) {
	cases := []struct {
		n    int
		noun string
		want string
	}{
		{0, "word", "0 words"},
		{1, "word", "1 word"},
		{2, "word", "2 words"},
		{1, "violation", "1 violation"},
		{3, "violation", "3 violations"},
		{1, "file", "1 file"},
	}
	for _, c := range cases {
		if got := count(c.n, c.noun); got != c.want {
			t.Errorf("count(%d, %q) = %q, want %q", c.n, c.noun, got, c.want)
		}
	}
}
