package dict

import (
	"os"
	"path/filepath"
	"testing"
)

// testIndex builds a small index by hand. The tests must not need the
// specification, which is under copyright and is never in this repository.
func testIndex() *Index {
	ix := NewIndex()
	for _, w := range []string{"use", "start", "stop", "install", "make sure", "get", "operate"} {
		ix.Approved[w] = Entry{Word: w, POS: []string{"v"}}
	}
	ix.Unapproved["utilize"] = Entry{Word: "utilize", POS: []string{"v"}, Alternatives: []string{"USE"}}
	ix.Unapproved["commence"] = Entry{Word: "commence", POS: []string{"v"}, Alternatives: []string{"START"}}
	ix.Unapproved["run"] = Entry{Word: "run", POS: []string{"v"}, Alternatives: []string{"OPERATE"}}
	ix.Unapproved["ensure"] = Entry{Word: "ensure"}
	ix.Unapproved["house"] = Entry{Word: "house", POS: []string{"v"}}
	return ix
}

func TestIsApprovedAcceptsRegularForms(t *testing.T) {
	ix := testIndex()
	for _, w := range []string{"use", "uses", "used", "using", "starts", "started",
		"starting", "stopped", "stopping", "installing", "installed"} {
		if !ix.IsApproved(w) {
			t.Errorf("IsApproved(%q) = false, want true", w)
		}
	}
	for _, w := range []string{"utilize", "commence", "banana"} {
		if ix.IsApproved(w) {
			t.Errorf("IsApproved(%q) = true, want false", w)
		}
	}
}

func TestRejectedGivesAlternatives(t *testing.T) {
	ix := testIndex()
	alts, pos, ok := ix.Rejected("utilize")
	if !ok || len(alts) != 1 || alts[0] != "USE" {
		t.Fatalf("Rejected(\"utilize\") = %v %v, want [USE] true", alts, ok)
	}
	if len(pos) != 1 || pos[0] != "v" {
		t.Errorf("parts of speech = %v, want [v]", pos)
	}
	// The match is exact. Stripping a suffix invents words, and a wrong guess
	// here reports a correct word as an error.
	if _, _, ok := ix.Rejected("housing"); ok {
		t.Error("Rejected(\"housing\") = true; it must not match \"house\"")
	}
	// An approved word is never rejected, even when a base form is listed.
	if _, _, ok := ix.Rejected("used"); ok {
		t.Error("Rejected(\"used\") = true, want false")
	}
	// A rejected word with no alternatives still reports as rejected.
	if alts, _, ok := ix.Rejected("ensure"); !ok || len(alts) != 0 {
		t.Errorf("Rejected(\"ensure\") = %v %v, want nil true", alts, ok)
	}
}

func TestKnown(t *testing.T) {
	ix := testIndex()
	for _, w := range []string{"use", "utilize", "started", "runs"} {
		if !ix.Known(w) {
			t.Errorf("Known(%q) = false, want true", w)
		}
	}
	if ix.Known("kubernetes") {
		t.Error("Known(\"kubernetes\") = true, want false")
	}
}

func TestSaveAndLoad(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "dict.json")
	in := testIndex()
	in.Source = "spec.pdf"
	in.Pages = 434
	if err := in.Save(p); err != nil {
		t.Fatal(err)
	}
	out, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Approved) != len(in.Approved) || out.Pages != 434 {
		t.Fatalf("round trip lost data: %+v", out.Stats())
	}
	if !out.IsApproved("using") {
		t.Error("the loaded index does not accept a regular form")
	}
}

func TestLoadRejectsAnOldFormat(t *testing.T) {
	p := filepath.Join(t.TempDir(), "dict.json")
	if err := os.WriteFile(p, []byte(`{"format":0,"approved":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err == nil {
		t.Error("expected an error for an old index format")
	}
}

func TestStats(t *testing.T) {
	s := testIndex().Stats()
	if s.Approved != 7 || s.Unapproved != 5 || s.WithAlts != 3 {
		t.Fatalf("stats = %+v", s)
	}
}

// The page parser must find headwords in flattened table text and attach the
// approved alternatives that follow an unapproved word.
func TestAddPage(t *testing.T) {
	ix := NewIndex()
	// The alternatives column carries a part of speech, exactly as the
	// headword column does. That marker is what keeps the all-capital example
	// sentences out of the index.
	ix.addPage(`Word
ABOUT (prep)
Concerned with
utilize (v)
USE (v)
commence (v)
START (v)
ABRASIVE (adj)
That can remove material
`)
	if _, ok := ix.Approved["about"]; !ok {
		t.Error("ABOUT was not recorded as approved")
	}
	if _, ok := ix.Approved["abrasive"]; !ok {
		t.Error("ABRASIVE was not recorded as approved")
	}
	if _, ok := ix.Approved["word"]; ok {
		t.Error("the column header was recorded as a word")
	}
	alts, _, ok := ix.Rejected("utilize")
	if !ok {
		t.Fatal("utilize was not recorded as unapproved")
	}
	if len(alts) == 0 || alts[0] != "USE" {
		t.Errorf("alternatives = %v, want [USE ...]", alts)
	}
}

// A word that appears first as an alternative and later as a headword must end
// up approved, not unapproved.
func TestApprovalWinsOverAnEarlierRecord(t *testing.T) {
	ix := NewIndex()
	ix.addUnapproved("start", "start", "v")
	ix.addApproved("start", "START", "v")
	if !ix.IsApproved("start") {
		t.Error("START must be approved")
	}
	if _, _, ok := ix.Rejected("start"); ok {
		t.Error("START must not stay in the unapproved map")
	}
}
