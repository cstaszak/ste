package lint

import (
	"strings"
	"testing"
)

func lintDefault(t *testing.T, text string) *Report {
	t.Helper()
	return RunText(text, DefaultOptions(ModeFlavored))
}

func countFor(t *testing.T, text, rule string) int {
	t.Helper()
	return lintDefault(t, text).Counts[rule]
}

// Hyphenated terms are the case a naive port of the Python regexes gets wrong.
func TestHyphenatedMarketingTerms(t *testing.T) {
	text := "The cutting-edge and best-in-class design is lightning-fast."
	if got := countFor(t, text, "marketing-adjective"); got != 3 {
		t.Fatalf("marketing-adjective = %d, want 3", got)
	}
}

func TestWholeWordMatchingOnly(t *testing.T) {
	// "beginner" contains "begin"; "ensured" contains "ensure". Neither is a hit.
	text := "The beginner ensured nothing. A beginning is not a match."
	if got := countFor(t, text, "banned-word"); got != 0 {
		t.Fatalf("banned-word = %d, want 0", got)
	}
}

func TestMultiWordPhrasePrefersLongestMatch(t *testing.T) {
	rep := lintDefault(t, "Run the check prior to the build in order to save time.")
	if got := rep.Counts["banned-word"]; got != 2 {
		t.Fatalf("banned-word = %d, want 2 (prior to, in order to)", got)
	}
	for _, f := range rep.Findings {
		if f.Rule != "banned-word" {
			continue
		}
		if f.Text != "prior to" && f.Text != "in order to" {
			t.Errorf("unexpected match %q", f.Text)
		}
	}
}

func TestCodeIsIgnoredButOffsetsSurvive(t *testing.T) {
	text := "Intro line.\n\n```\nutilize this seamless robust code\n```\n\nWe utilize it.\n"
	rep := lintDefault(t, text)
	if got := rep.Counts["banned-word"]; got != 1 {
		t.Fatalf("banned-word = %d, want 1 (code fence must be ignored)", got)
	}
	f := rep.Findings[0]
	if f.Position.Line != 7 {
		t.Errorf("line = %d, want 7 (offsets must survive code masking)", f.Position.Line)
	}
	if got := strings.Index(text, "utilize it"); f.Offset != got {
		t.Errorf("offset = %d, want %d", f.Offset, got)
	}
}

func TestInlineCodeIsIgnored(t *testing.T) {
	if got := countFor(t, "Call `utilize()` here.", "banned-word"); got != 0 {
		t.Fatalf("banned-word = %d, want 0", got)
	}
}

func TestPassiveVoice(t *testing.T) {
	cases := map[string]int{
		"The file is read by the parser.":                  1,
		"The parser reads the file.":                       0,
		"Results were shown and data was written to disk.": 2,
	}
	for text, want := range cases {
		if got := countFor(t, text, "passive-voice"); got != want {
			t.Errorf("passive-voice(%q) = %d, want %d", text, got, want)
		}
	}
}

func TestContractionsExcludePossessives(t *testing.T) {
	cases := map[string]int{
		"The parser's output is fine.":    0, // possessive
		"It's broken.":                    1,
		"We don't know and they're late.": 2,
	}
	for text, want := range cases {
		if got := countFor(t, text, "contraction"); got != want {
			t.Errorf("contraction(%q) = %d, want %d", text, got, want)
		}
	}
}

func TestNominalization(t *testing.T) {
	if got := countFor(t, "Perform an analysis of the log.", "nominalization"); got == 0 {
		t.Error("expected a nominalization for \"perform\"")
	}
	if got := countFor(t, "The execution of the job failed.", "nominalization"); got != 1 {
		t.Errorf("nominalization = %d, want 1 for \"execution of\"", got)
	}
	if got := countFor(t, "Analyze the log.", "nominalization"); got != 0 {
		t.Errorf("nominalization = %d, want 0", got)
	}
}

func TestSentenceLengthDependsOnMode(t *testing.T) {
	// 22 words.
	text := "This one sentence has quite a lot of words in it and it keeps going on well past the point of clarity."
	if got := RunText(text, DefaultOptions(ModeStrict)).Counts["long-sentence"]; got != 1 {
		t.Errorf("strict long-sentence = %d, want 1", got)
	}
	if got := RunText(text, DefaultOptions(ModeFlavored)).Counts["long-sentence"]; got != 0 {
		t.Errorf("flavored long-sentence = %d, want 0 (limit is 25)", got)
	}
}

func TestLongParagraph(t *testing.T) {
	p := strings.Repeat("Short line here. ", 7)
	if got := countFor(t, p, "long-paragraph"); got != 1 {
		t.Fatalf("long-paragraph = %d, want 1", got)
	}
	if got := countFor(t, strings.Repeat("Short line here. ", 3), "long-paragraph"); got != 0 {
		t.Fatalf("long-paragraph = %d, want 0", got)
	}
}

func TestEmDashIsReportedButNotScored(t *testing.T) {
	rep := lintDefault(t, "A clean sentence — with a dash.")
	if rep.EmDashes != 1 {
		t.Errorf("em dashes = %d, want 1", rep.EmDashes)
	}
	if rep.Total != 0 {
		t.Errorf("total = %d, want 0 (em dash must not score by default)", rep.Total)
	}
}

func TestScoreIsPer100Words(t *testing.T) {
	// Ten words, one violation -> 10 per 100 words.
	rep := lintDefault(t, "We utilize the tool to make the thing go now.")
	if rep.Words != 10 {
		t.Fatalf("words = %d, want 10", rep.Words)
	}
	if rep.Per100W != 10 {
		t.Fatalf("per100w = %v, want 10", rep.Per100W)
	}
}

func TestCleanSteTextScoresZero(t *testing.T) {
	text := "Start the server. The parser reads the file. " +
		"If the check fails, the tool prints an error.\n"
	rep := lintDefault(t, text)
	if rep.Total != 0 {
		t.Fatalf("total = %d, want 0; findings: %+v", rep.Total, rep.Findings)
	}
}

func TestSlopScoresHigherThanRewrite(t *testing.T) {
	slop := "It is important to note that we leverage a comprehensive, robust solution " +
		"in order to seamlessly facilitate the utilization of cutting-edge tooling; " +
		"additionally, results are demonstrated by the system."
	clean := "The tool uses standard libraries. " +
		"The system shows the results. Start it with one command."
	a, b := lintDefault(t, slop), lintDefault(t, clean)
	if !(a.Per100W > b.Per100W) {
		t.Fatalf("slop %v must score above clean %v", a.Per100W, b.Per100W)
	}
	if b.Per100W != 0 {
		t.Errorf("clean text scored %v, want 0; findings: %+v", b.Per100W, b.Findings)
	}
}

func TestPositionsPointAtTheRightPlace(t *testing.T) {
	text := "Line one is fine.\nWe utilize it.\n"
	rep := lintDefault(t, text)
	if len(rep.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(rep.Findings))
	}
	f := rep.Findings[0]
	if f.Position.Line != 2 || f.Position.Col != 4 {
		t.Errorf("position = %d:%d, want 2:4", f.Position.Line, f.Position.Col)
	}
}

func TestHeadingsAndListMarkersAreStripped(t *testing.T) {
	text := "## We utilize this\n\n- We utilize that\n1. We utilize other\n"
	if got := countFor(t, text, "banned-word"); got != 3 {
		t.Fatalf("banned-word = %d, want 3", got)
	}
}

func TestFindingsAreSortedByOffset(t *testing.T) {
	rep := lintDefault(t, "We utilize a seamless approach; it is important to note this.")
	for i := 1; i < len(rep.Findings); i++ {
		if rep.Findings[i-1].Offset > rep.Findings[i].Offset {
			t.Fatal("findings are not sorted by offset")
		}
	}
}

func TestDeterministic(t *testing.T) {
	text := "We utilize a robust, seamless approach; results are shown by the tool."
	first := lintDefault(t, text)
	for i := 0; i < 20; i++ {
		if got := lintDefault(t, text); got.Total != first.Total || got.Per100W != first.Per100W {
			t.Fatal("lint output is not deterministic")
		}
	}
}

// A vertical list is what STE asks for in a procedure. List items must not
// count against the paragraph limit.
func TestListItemsDoNotCountAsParagraphSentences(t *testing.T) {
	text := "Do these steps:\n\n1. Open the panel.\n2. Remove the filter.\n3. Install the new filter.\n" +
		"4. Close the panel.\n5. Set the switch to ON.\n6. Check the light.\n7. Start the pump.\n"
	if got := countFor(t, text, "long-paragraph"); got != 0 {
		t.Fatalf("long-paragraph = %d, want 0", got)
	}
}

// Inline code is part of the sentence around it, and it can start a sentence.
// A blank placeholder would join two sentences into one long one.
func TestInlineCodeCanStartASentence(t *testing.T) {
	text := "The rate fell to 5.5 per 100 words. `ste` is a Go rewrite of that tool.\n"
	d := Parse(text)
	if len(d.Sentences) != 2 {
		t.Fatalf("sentences = %d, want 2: %+v", len(d.Sentences), d.Sentences)
	}
}

// A fenced block is not prose and must not count toward the word total.
func TestFencedCodeDoesNotCountAsWords(t *testing.T) {
	plain := lintDefault(t, "Start the server.\n")
	withCode := lintDefault(t, "Start the server.\n\n```\nutilize a b c d e f g\n```\n")
	if plain.Words != withCode.Words {
		t.Fatalf("words = %d with a code fence, want %d", withCode.Words, plain.Words)
	}
}

// A markdown table is a set of rows, not one long sentence.
func TestTableRowsAreSeparateSentences(t *testing.T) {
	text := "| Format | Use |\n|---|---|\n| text | The default output. |\n| json | Full detail for a script. |\n"
	if got := countFor(t, text, "long-sentence"); got != 0 {
		t.Fatalf("long-sentence = %d, want 0", got)
	}
}

// A noun that names a thing is not a nominalization, even when it ends in one
// of the suffixes the rule looks for.
func TestConcreteNounsAreNotNominalizations(t *testing.T) {
	for _, s := range []string{
		"Read the function of the parser.",
		"Check the version of the file.",
		"Give the location of the error.",
	} {
		if got := countFor(t, s, "nominalization"); got != 0 {
			t.Errorf("nominalization(%q) = %d, want 0", s, got)
		}
	}
	if got := countFor(t, "The execution of the job failed.", "nominalization"); got != 1 {
		t.Errorf("nominalization = %d, want 1", got)
	}
}
