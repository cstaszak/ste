package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cstaszak/ste/internal/lint"
)

func write(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, Name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadAndApply(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, `
mode: strict
max_per_100w: 1.5
max_sentence_words: 12
rules:
  em-dash: true
  semicolon: false
words:
  banned-word:
    add:
      "going forward": "from now on"
    remove: [begin, begins]
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}

	m, err := cfg.ResolveMode("")
	if err != nil {
		t.Fatal(err)
	}
	if m != lint.ModeStrict {
		t.Errorf("mode = %v, want strict", m)
	}
	// A flag beats the file.
	if m, _ := cfg.ResolveMode("flavored"); m != lint.ModeFlavored {
		t.Errorf("flag mode = %v, want flavored", m)
	}

	if v, ok := cfg.Threshold(); !ok || v != 1.5 {
		t.Errorf("threshold = %v %v, want 1.5 true", v, ok)
	}

	opt := cfg.Apply(lint.DefaultOptions(m))
	if opt.MaxSentenceWords != 12 {
		t.Errorf("max sentence words = %d, want 12", opt.MaxSentenceWords)
	}

	rep := lint.RunText("We go forward now; the plan begins today - it works.", opt)
	if rep.Counts["semicolon"] != 0 {
		t.Error("semicolon was turned off but still fired")
	}
	if rep.Counts["banned-word"] != 0 {
		t.Errorf("banned-word = %d, want 0 after removing \"begin\": %+v",
			rep.Counts["banned-word"], rep.Findings)
	}

	added := lint.RunText("Going forward we ship weekly.", opt)
	if added.Counts["banned-word"] != 1 {
		t.Errorf("the added phrase did not match: %+v", added.Findings)
	}
}

// Turning on em-dash must make it count toward the score.
func TestEnablingEmDashScoresIt(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(write(t, dir, "rules:\n  em-dash: true\n"))
	if err != nil {
		t.Fatal(err)
	}
	opt := cfg.Apply(lint.DefaultOptions(lint.ModeFlavored))
	rep := lint.RunText("A clean sentence — with a dash.", opt)
	if rep.Counts["em-dash"] != 1 || rep.Total != 1 {
		t.Fatalf("em-dash = %d, total = %d, want 1 and 1", rep.Counts["em-dash"], rep.Total)
	}
}

// The embedded defaults must survive a configuration that edits a word list.
func TestApplyDoesNotChangeTheDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(write(t, dir, "words:\n  banned-word:\n    remove: [utilize]\n"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Apply(lint.DefaultOptions(lint.ModeFlavored))

	rep := lint.RunText("We utilize it.", lint.DefaultOptions(lint.ModeFlavored))
	if rep.Counts["banned-word"] != 1 {
		t.Fatal("the configuration changed the embedded default list")
	}
}

func TestUnknownFieldsAndRulesFail(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"unknown field": "mode: strict\nnot_a_field: 1\n",
		"unknown rule":  "rules:\n  no-such-rule: true\n",
		"bad mode":      "mode: banana\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(write(t, t.TempDir(), body)); err == nil {
				t.Error("expected an error")
			}
		})
	}
	_ = dir
}

func TestFindWalksUp(t *testing.T) {
	root := t.TempDir()
	write(t, root, "mode: strict\n")
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Find(deep); got != filepath.Join(root, Name) {
		t.Errorf("Find = %q, want the file in the root", got)
	}
}

func TestEmptyPathGivesDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Threshold(); ok {
		t.Error("an empty configuration must set no threshold")
	}
	if m, _ := cfg.ResolveMode(""); m != lint.ModeFlavored {
		t.Error("the default mode must be flavored")
	}
}
