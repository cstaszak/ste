package lint

import (
	"math"
	"sort"
	"strings"
)

// Options controls one lint run.
type Options struct {
	Mode                  Mode
	MaxSentenceWords      int
	MaxParagraphSentences int
	// Enabled holds the rule IDs to run. A nil map means "the defaults for
	// this mode".
	Enabled map[string]bool
	// Lists holds the word lists, keyed by rule ID.
	Lists map[string]*PhraseList
	// Dict, when set, checks each word against the STE approved dictionary.
	Dict Approver
	// Allow holds technical names the dictionary check accepts, in lower case.
	Allow map[string]bool
}

// Approver answers questions about the ASD-STE100 dictionary. The package
// internal/dict provides the implementation. The lint package does not import
// it, so the lint core stays free of the PDF reader.
type Approver interface {
	// IsApproved reports whether the word, or a regular form of it, is an
	// approved word.
	IsApproved(word string) bool
	// Rejected reports whether the standard lists the word and does not
	// approve it. It returns approved words to use instead, and the parts of
	// speech the standard rejects the word for.
	Rejected(word string) (alternatives, partsOfSpeech []string, listed bool)
	// Known reports whether the standard lists the word at all.
	Known(word string) bool
}

// DefaultOptions returns the options for a mode.
func DefaultOptions(mode Mode) Options {
	o := Options{
		Mode:                  mode,
		MaxParagraphSentences: 6,
		Lists:                 map[string]*PhraseList{},
		Enabled:               map[string]bool{},
		Allow:                 map[string]bool{},
	}
	if mode == ModeStrict {
		o.MaxSentenceWords = 20
	} else {
		o.MaxSentenceWords = 25
	}
	for id, l := range defaultLists {
		o.Lists[id] = l.Clone()
	}
	for _, r := range Rules() {
		o.Enabled[r.ID] = !r.DefaultOff && (mode == ModeStrict || !r.StrictOnly)
	}
	return o
}

// list returns the word list for a rule, or an empty list.
func (o Options) list(id string) *PhraseList {
	if l, ok := o.Lists[id]; ok {
		return l
	}
	return NewPhraseList()
}

// Report is the result of linting one document.
type Report struct {
	Path                 string         `json:"path,omitempty"`
	Words                int            `json:"words"`
	Sentences            int            `json:"sentences"`
	Total                int            `json:"total"`
	Per100W              float64        `json:"per_100w"`
	Counts               map[string]int `json:"counts"`
	Findings             []Finding      `json:"findings"`
	LongestSentenceWords int            `json:"longest_sentence_words"`
	EmDashes             int            `json:"em_dashes"`
}

// CountsPer100W returns the per-rule rate for each rule with a violation.
func (r *Report) CountsPer100W() map[string]float64 {
	out := map[string]float64{}
	for id, n := range r.Counts {
		out[id] = round2(float64(n) * 100 / float64(max(r.Words, 1)))
	}
	return out
}

// Run lints a document and returns the report.
func Run(d *Document, opt Options) *Report {
	rep := &Report{
		Words:     d.Words(),
		Sentences: len(d.Sentences),
		Counts:    map[string]int{},
	}
	for _, s := range d.Sentences {
		if n := s.Words(); n > rep.LongestSentenceWords {
			rep.LongestSentenceWords = n
		}
	}
	for _, r := range Rules() {
		if !opt.Enabled[r.ID] {
			continue
		}
		found := r.Check(d, opt)
		if len(found) == 0 {
			continue
		}
		rep.Counts[r.ID] = len(found)
		rep.Findings = append(rep.Findings, found...)
		if r.Scored {
			rep.Total += len(found)
		}
	}
	// The em-dash count is always reported, whether or not the rule is on.
	rep.EmDashes = countEmDashes(d)

	sort.SliceStable(rep.Findings, func(i, j int) bool {
		return rep.Findings[i].Offset < rep.Findings[j].Offset
	})
	rep.Per100W = round2(float64(rep.Total) * 100 / float64(max(rep.Words, 1)))
	return rep
}

// RunText parses and lints raw text in one step.
func RunText(text string, opt Options) *Report {
	return Run(Parse(text), opt)
}

func countEmDashes(d *Document) int {
	return strings.Count(d.Masked, "—") + strings.Count(d.Masked, "–")
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
