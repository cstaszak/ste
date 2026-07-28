// Package dict builds and reads the ASD-STE100 approved-word index.
//
// The standard is under copyright. This package holds no part of it. It reads a
// local copy of the specification that you supply, and writes an index into a
// cache directory on your own machine. Do not commit that index.
package dict

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FormatVersion changes when the shape of the index file changes.
const FormatVersion = 1

// Entry is one headword of the dictionary.
type Entry struct {
	Word string `json:"word"`
	// POS holds the parts of speech given for the word.
	POS []string `json:"pos,omitempty"`
	// Alternatives holds approved words to use instead. It is set for an
	// unapproved word only, and it is a best effort.
	Alternatives []string `json:"alternatives,omitempty"`
}

// Index is the built dictionary.
type Index struct {
	Format int `json:"format"`
	// Source is the file the index came from.
	Source string `json:"source"`
	// Pages is the page count of the source, as a weak check of the edition.
	Pages int `json:"pages"`
	// Approved holds each approved word, keyed by its lowercase form.
	Approved map[string]Entry `json:"approved"`
	// Unapproved holds each word the standard lists and does not approve.
	Unapproved map[string]Entry `json:"unapproved"`
}

// NewIndex returns an empty index.
func NewIndex() *Index {
	return &Index{
		Format:     FormatVersion,
		Approved:   map[string]Entry{},
		Unapproved: map[string]Entry{},
	}
}

// suffixRules strip a regular inflection to find the base word. The standard
// approves a set of forms for each word. This tool accepts the regular forms of
// an approved word, so that "starts" and "started" pass when "start" is
// approved.
var suffixRules = []struct{ suffix, add string }{
	{"s", ""}, {"es", ""}, {"ies", "y"},
	{"ed", ""}, {"ed", "e"}, {"ied", "y"},
	{"ing", ""}, {"ing", "e"},
	{"ly", ""}, {"er", ""}, {"est", ""},
}

// baseForms returns the word itself and every base form a regular inflection
// rule can produce from it.
func baseForms(word string) []string {
	w := strings.ToLower(word)
	out := []string{w}
	for _, r := range suffixRules {
		if !strings.HasSuffix(w, r.suffix) {
			continue
		}
		base := strings.TrimSuffix(w, r.suffix) + r.add
		if len(base) < 3 {
			continue
		}
		out = append(out, base)
		// A doubled final consonant, as in "stopped" from "stop".
		if n := len(base); n >= 2 && base[n-1] == base[n-2] {
			out = append(out, base[:n-1])
		}
	}
	return out
}

// IsApproved reports whether a word is approved, or is a regular form of an
// approved word.
func (ix *Index) IsApproved(word string) bool {
	for _, f := range baseForms(word) {
		if _, ok := ix.Approved[f]; ok {
			return true
		}
	}
	return false
}

// Rejected reports whether the standard lists the word and does not approve it.
// It returns the approved words to use instead and the parts of speech the
// standard rejects it for. This satisfies the Approver interface of the lint
// package.
//
// The match is on the exact word. A regular form is not enough here, because
// stripping a suffix invents words: "housing" is not a form of "house", and
// "warning" is a noun of its own. A wrong suffix guess in this rule reports a
// correct word as an error, so the rule stays exact.
func (ix *Index) Rejected(word string) (alternatives, partsOfSpeech []string, listed bool) {
	if ix.IsApproved(word) {
		return nil, nil, false
	}
	e, ok := ix.Unapproved[strings.ToLower(word)]
	if !ok {
		return nil, nil, false
	}
	return e.Alternatives, e.POS, true
}

// Known reports whether the standard lists the word at all, in any regular
// form.
func (ix *Index) Known(word string) bool {
	for _, f := range baseForms(word) {
		if _, ok := ix.Approved[f]; ok {
			return true
		}
		if _, ok := ix.Unapproved[f]; ok {
			return true
		}
	}
	return false
}

// DefaultPath returns the path of the index in the user cache directory.
func DefaultPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ste", "dict.json"), nil
}

// Save writes the index. It creates the parent directory.
func (ix *Index) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ix, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Load reads an index from disk.
func Load(path string) (*Index, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ix := NewIndex()
	if err := json.Unmarshal(b, ix); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if ix.Format != FormatVersion {
		return nil, fmt.Errorf("%s: index format %d, this build reads %d; build it again",
			path, ix.Format, FormatVersion)
	}
	return ix, nil
}

// Stats describes a built index.
type Stats struct {
	Approved   int
	Unapproved int
	WithAlts   int
	Source     string
	Pages      int
}

// Stats summarizes the index.
func (ix *Index) Stats() Stats {
	s := Stats{
		Approved:   len(ix.Approved),
		Unapproved: len(ix.Unapproved),
		Source:     ix.Source,
		Pages:      ix.Pages,
	}
	for _, e := range ix.Unapproved {
		if len(e.Alternatives) > 0 {
			s.WithAlts++
		}
	}
	return s
}

// Words returns the sorted keys of a map of entries.
func Words(m map[string]Entry) []string {
	out := make([]string, 0, len(m))
	for w := range m {
		out = append(out, w)
	}
	sort.Strings(out)
	return out
}
