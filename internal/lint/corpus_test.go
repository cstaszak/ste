package lint

import (
	"os"
	"path/filepath"
	"testing"
)

// The corpus test locks in the scores of a small set of documents. It catches
// any change in the rules that moves the numbers, and it proves the central
// claim of the tool: an STE rewrite scores lower than the draft it replaces.
func TestCorpusScores(t *testing.T) {
	cases := []struct {
		file    string
		mode    Mode
		per100w float64
	}{
		{"readme-slop.md", ModeFlavored, 29.27},
		{"readme-ste.md", ModeFlavored, 0},
		{"procedure-ste.md", ModeStrict, 0},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			rep := lintFile(t, c.file, c.mode)
			if rep.Per100W != c.per100w {
				t.Errorf("per100w = %v, want %v\n%s", rep.Per100W, c.per100w, dumpFindings(rep))
			}
		})
	}
}

func TestRewriteScoresLowerThanDraft(t *testing.T) {
	slop := lintFile(t, "readme-slop.md", ModeFlavored)
	ste := lintFile(t, "readme-ste.md", ModeFlavored)
	if ste.Per100W >= slop.Per100W {
		t.Fatalf("the STE rewrite scored %v, which is not below the draft score %v",
			ste.Per100W, slop.Per100W)
	}
}

// A procedure written in STE must pass the strict limits as well as the
// relaxed ones.
func TestProcedurePassesBothModes(t *testing.T) {
	for _, m := range []Mode{ModeFlavored, ModeStrict} {
		if rep := lintFile(t, "procedure-ste.md", m); rep.Total != 0 {
			t.Errorf("mode %s: total = %d, want 0\n%s", m, rep.Total, dumpFindings(rep))
		}
	}
}

func lintFile(t *testing.T, name string, mode Mode) *Report {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus", name))
	if err != nil {
		t.Fatal(err)
	}
	return RunText(string(b), DefaultOptions(mode))
}

func dumpFindings(r *Report) string {
	s := ""
	for _, f := range r.Findings {
		s += "  " + f.Rule + " " + f.Message + "\n"
	}
	return s
}
