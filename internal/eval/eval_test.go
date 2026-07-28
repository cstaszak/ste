package eval

import (
	"strings"
	"testing"

	"github.com/cstaszak/ste/internal/lint"
)

func TestConditionsLoad(t *testing.T) {
	if len(Conditions) != 4 {
		t.Fatalf("conditions = %d, want 4", len(Conditions))
	}
	if Conditions[0].ID != "baseline" || Conditions[0].System != "" {
		t.Error("the first condition must be the baseline with no system prompt")
	}
	for _, c := range Conditions[1:] {
		if len(c.System) < 100 {
			t.Errorf("condition %q has a system prompt of %d characters", c.ID, len(c.System))
		}
	}
	if !strings.Contains(ConditionSystem(t, "ste"), "Simplified Technical English") {
		t.Error("the STE condition does not name the standard")
	}
}

func ConditionSystem(t *testing.T, id string) string {
	t.Helper()
	c, ok := ConditionByID(id)
	if !ok {
		t.Fatalf("no condition %q", id)
	}
	return c.System
}

// The STE condition states the rules the linter checks, so it must pass them.
// Words it has to name, such as the banned adjectives, are quoted, and the
// linter ignores quoted text. A prompt that breaks its own rules would be a
// poor advertisement for them.
func TestSteConditionPassesItsOwnRules(t *testing.T) {
	rep := lint.RunText(ConditionSystem(t, "ste"), lint.DefaultOptions(lint.ModeFlavored))
	if rep.Total != 0 {
		for _, f := range rep.Findings {
			t.Errorf("the STE prompt breaks its own rule: L%d %s", f.Position.Line, f.Message)
		}
	}
}

func TestTasksAreDistinct(t *testing.T) {
	seen := map[string]bool{}
	for _, task := range Tasks {
		if seen[task.ID] {
			t.Errorf("duplicate task ID %q", task.ID)
		}
		seen[task.ID] = true
		if task.Prompt == "" || task.Name == "" {
			t.Errorf("task %q is incomplete", task.ID)
		}
	}
	if _, ok := TaskByID("readme"); !ok {
		t.Error("TaskByID did not find a known task")
	}
	if _, ok := TaskByID("nope"); ok {
		t.Error("TaskByID found a task that does not exist")
	}
}

func TestCallsAndEstimate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Models = []string{"claude-opus-5", "claude-haiku-4-5"}
	if got, want := cfg.Calls(), 2*len(Tasks)*len(Conditions); got != want {
		t.Errorf("calls = %d, want %d", got, want)
	}
	if cfg.Estimate() <= 0 {
		t.Error("the cost estimate must be above zero")
	}
	// A cheaper model must give a lower estimate.
	cheap := cfg
	cheap.Models = []string{"claude-haiku-4-5"}
	dear := cfg
	dear.Models = []string{"claude-opus-5"}
	if cheap.Estimate() >= dear.Estimate() {
		t.Error("Haiku must cost less than Opus")
	}
}

func TestModelByIDFallsBack(t *testing.T) {
	m := ModelByID("some-future-model")
	if m.ID != "some-future-model" || m.InPerMTok != 0 {
		t.Errorf("unexpected fallback: %+v", m)
	}
}

// Summarize pools violations over words, so one long output does not weigh
// more than its length deserves.
func TestSummarizePools(t *testing.T) {
	results := []Result{
		{Model: "m", Condition: "baseline", Report: &lint.Report{Words: 100, Total: 4}},
		{Model: "m", Condition: "baseline", Report: &lint.Report{Words: 300, Total: 8}},
		{Model: "m", Condition: "ste", Report: &lint.Report{Words: 200, Total: 2}},
		{Model: "m", Condition: "ste", Err: "boom", Report: &lint.Report{}},
	}
	sums := Summarize(results)
	if len(sums) != 2 {
		t.Fatalf("summaries = %d, want 2", len(sums))
	}
	// baseline: 12 violations over 400 words = 3.00
	if sums[0].Condition != "baseline" || sums[0].Per100W != 3 {
		t.Errorf("baseline = %+v, want 3.00", sums[0])
	}
	// ste: 2 over 200 = 1.00, and one error
	if sums[1].Condition != "ste" || sums[1].Per100W != 1 || sums[1].Errors != 1 {
		t.Errorf("ste = %+v, want 1.00 with 1 error", sums[1])
	}
}

func TestWriteMarkdown(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tasks = Tasks[:1]
	cfg.Conditions = Conditions
	results := []Result{
		{Model: "claude-opus-5", Task: "readme", Condition: "baseline",
			Report: &lint.Report{Words: 100, Total: 4, Per100W: 4}},
		{Model: "claude-opus-5", Task: "readme", Condition: "ste",
			Report: &lint.Report{Words: 100, Total: 1, Per100W: 1}},
	}
	var sb strings.Builder
	WriteMarkdown(&sb, results, cfg)
	out := sb.String()

	for _, want := range []string{"Claude Opus 5", "4.00", "1.00", "(-75%)", "Total cost"} {
		if !strings.Contains(out, want) {
			t.Errorf("the table does not contain %q\n%s", want, out)
		}
	}
}

func TestWriteMarkdownReportsErrors(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tasks = Tasks[:1]
	results := []Result{
		{Model: "m", Task: "readme", Condition: "baseline", Err: "rate limited", Report: &lint.Report{}},
	}
	var sb strings.Builder
	WriteMarkdown(&sb, results, cfg)
	if !strings.Contains(sb.String(), "rate limited") {
		t.Error("the table does not report the error")
	}
}
