package lint

import (
	"fmt"
	"sort"
)

// Category groups rules for reporting.
type Category string

const (
	CatWords       Category = "words"
	CatVerbs       Category = "verbs"
	CatSentences   Category = "sentences"
	CatPunctuation Category = "punctuation"
	CatStructure   Category = "structure"
	CatDictionary  Category = "dictionary"
)

// Mode selects how strictly the rules apply. It mirrors the two modes of the
// ste-writing skill.
type Mode string

const (
	// ModeFlavored applies the structural and voice rules to general prose.
	ModeFlavored Mode = "flavored"
	// ModeStrict applies every rule, for procedures, runbooks, and error text.
	ModeStrict Mode = "strict"
)

// ParseMode converts a mode name to a Mode.
func ParseMode(s string) (Mode, error) {
	switch Mode(s) {
	case ModeFlavored, ModeStrict:
		return Mode(s), nil
	default:
		return "", fmt.Errorf("unknown mode %q (use flavored or strict)", s)
	}
}

// Finding is one rule violation at one place in the document.
type Finding struct {
	Rule     string   `json:"rule"`
	Category Category `json:"category"`
	Message  string   `json:"message"`
	Suggest  string   `json:"suggest,omitempty"`
	Position Position `json:"position"`
	Offset   int      `json:"-"`
	Text     string   `json:"text,omitempty"`
}

// Rule reports every place in a document that breaks one writing rule.
type Rule struct {
	ID       string
	Category Category
	Doc      string
	// Scored rules count toward the per-100-word score. A rule that only
	// marks a signal, such as em-dash, can stay off the score.
	Scored bool
	// StrictOnly rules run only in strict mode.
	StrictOnly bool
	// DefaultOff rules run only when the configuration turns them on.
	DefaultOff bool
	Check      func(d *Document, opt Options) []Finding
}

var registry = map[string]*Rule{}
var registryOrder []string

func register(r *Rule) {
	if _, dup := registry[r.ID]; dup {
		panic("duplicate rule id: " + r.ID)
	}
	registry[r.ID] = r
	registryOrder = append(registryOrder, r.ID)
}

// Rules returns every registered rule, sorted by ID.
func Rules() []*Rule {
	out := make([]*Rule, 0, len(registry))
	for _, id := range registryOrder {
		out = append(out, registry[id])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Lookup returns the rule with the given ID.
func Lookup(id string) (*Rule, bool) {
	r, ok := registry[id]
	return r, ok
}

// finding builds a Finding for a token span. Whitespace in the quoted text is
// collapsed, so a match that spans a line break still prints on one line.
func finding(d *Document, r *Rule, offset int, text, msg, suggest string) Finding {
	return Finding{
		Rule:     r.ID,
		Category: r.Category,
		Message:  normalizeSpace(msg),
		Suggest:  suggest,
		Position: d.Position(offset),
		Offset:   offset,
		Text:     normalizeSpace(text),
	}
}
