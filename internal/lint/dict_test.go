package lint

import "testing"

// fakeDict stands in for the real index. The specification is under copyright
// and is never in this repository, so the tests supply their own data.
type fakeDict struct {
	approved map[string]bool
	rejected map[string]struct {
		alts []string
		pos  []string
	}
}

func (f *fakeDict) IsApproved(w string) bool { return f.approved[w] }

func (f *fakeDict) Rejected(w string) ([]string, []string, bool) {
	e, ok := f.rejected[w]
	if !ok || f.approved[w] {
		return nil, nil, false
	}
	return e.alts, e.pos, true
}

func (f *fakeDict) Known(w string) bool {
	_, r := f.rejected[w]
	return f.approved[w] || r
}

func testDict() *fakeDict {
	f := &fakeDict{
		approved: map[string]bool{"use": true, "start": true, "set": true, "open": true, "the": true},
		rejected: map[string]struct {
			alts []string
			pos  []string
		}{},
	}
	f.rejected["utilize"] = struct {
		alts []string
		pos  []string
	}{[]string{"USE"}, []string{"v"}}
	f.rejected["filter"] = struct {
		alts []string
		pos  []string
	}{[]string{"FILTERED"}, []string{"v"}}
	f.rejected["main"] = struct {
		alts []string
		pos  []string
	}{[]string{"PRIMARY"}, []string{"adj"}}
	f.rejected["order"] = struct {
		alts []string
		pos  []string
	}{[]string{"SEQUENCE"}, []string{"n", "v"}}
	return f
}

func dictOptions() Options {
	o := DefaultOptions(ModeStrict)
	o.Dict = testDict()
	o.Enabled["non-approved-word"] = true
	return o
}

func dictCount(t *testing.T, text string) int {
	t.Helper()
	return RunText(text, dictOptions()).Counts["non-approved-word"]
}

// The standard permits a technical name. A technical name is a noun, so a word
// the standard rejects for its verb sense only is allowed as a noun.
func TestTechnicalNameAsANounIsAllowed(t *testing.T) {
	cases := map[string]int{
		"Remove the filter.":            0, // noun, permitted technical name
		"Set the main switch to OFF.":   1, // "main" is rejected as an adjective
		"Filter the fuel.":              1, // verb, rejected
		"Remove the two filter bolts.":  0, // noun inside a noun phrase
		"The system will filter it.":    1, // after a verb, so not a noun phrase
		"Do not utilize the interface.": 1,
	}
	for text, want := range cases {
		if got := dictCount(t, text); got != want {
			t.Errorf("%q = %d, want %d", text, got, want)
		}
	}
}

// A word the standard rejects as a noun stays rejected in a noun phrase.
func TestRejectedNounIsStillReported(t *testing.T) {
	if got := dictCount(t, "Check the order of the steps."); got != 1 {
		t.Errorf("got %d, want 1", got)
	}
}

func TestDictionaryRuleNeedsAnIndex(t *testing.T) {
	o := DefaultOptions(ModeStrict)
	o.Enabled["non-approved-word"] = true // on, but with no index
	if rep := RunText("Do not utilize it.", o); rep.Counts["non-approved-word"] != 0 {
		t.Fatal("the rule ran without an index")
	}
}

func TestAllowListSkipsTechnicalNames(t *testing.T) {
	o := dictOptions()
	o.Allow["utilize"] = true
	if rep := RunText("Do not utilize it.", o); rep.Counts["non-approved-word"] != 0 {
		t.Fatal("the allow list did not skip the word")
	}
}

// A capital letter away from the start of a sentence marks a name, and a
// dictionary cannot judge a name.
func TestProperNounsAreSkipped(t *testing.T) {
	if got := dictCount(t, "The Filter service is ready."); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

// The dictionary rules are off unless the caller asks for them.
func TestDictionaryRulesAreOffByDefault(t *testing.T) {
	for _, m := range []Mode{ModeFlavored, ModeStrict} {
		o := DefaultOptions(m)
		if o.Enabled["non-approved-word"] || o.Enabled["unknown-word"] {
			t.Errorf("mode %s: a dictionary rule is on by default", m)
		}
	}
}
