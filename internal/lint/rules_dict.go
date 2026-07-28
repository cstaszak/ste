package lint

import (
	"fmt"
	"strings"
	"unicode"
)

func init() {
	register(&Rule{
		ID: "non-approved-word", Category: CatDictionary, Scored: true,
		StrictOnly: true, DefaultOff: true,
		Doc: "The standard lists this word and does not approve it. Needs a dictionary index.",
		Check: func(d *Document, opt Options) []Finding {
			if opt.Dict == nil {
				return nil
			}
			r := registry["non-approved-word"]
			var out []Finding
			for _, s := range d.Sentences {
				for i, t := range s.Tokens {
					if skipForDict(s, i, opt) {
						continue
					}
					alts, pos, listed := opt.Dict.Rejected(t.Lower)
					if !listed || opt.Dict.IsApproved(t.Lower) {
						continue
					}
					// The standard permits a technical name, and a technical
					// name is a noun. When the standard rejects a word for its
					// verb sense only, and the word reads as a noun here, the
					// use is allowed. An adjective is not a technical name, so
					// this exemption covers verbs only.
					if verbOnly(pos) && looksLikeNoun(s, i) {
						continue
					}
					suggest := "use an approved word"
					if len(alts) > 0 {
						suggest = "use " + strings.ToLower(strings.Join(alts, ", "))
					}
					out = append(out, finding(d, r, t.Start, t.Text,
						fmt.Sprintf("not an approved STE word: %q", t.Text), suggest))
				}
			}
			return out
		},
	})

	register(&Rule{
		ID: "unknown-word", Category: CatDictionary, Scored: true,
		StrictOnly: true, DefaultOff: true,
		Doc: "The standard does not list this word at all. Noisy, because a technical name is not in the dictionary.",
		Check: func(d *Document, opt Options) []Finding {
			if opt.Dict == nil {
				return nil
			}
			r := registry["unknown-word"]
			var out []Finding
			for _, s := range d.Sentences {
				for i, t := range s.Tokens {
					if skipForDict(s, i, opt) {
						continue
					}
					if opt.Dict.Known(t.Lower) || opt.Dict.IsApproved(t.Lower) {
						continue
					}
					out = append(out, finding(d, r, t.Start, t.Text,
						fmt.Sprintf("not in the STE dictionary: %q", t.Text),
						"use an approved word, or add it to \"allow\" in .ste.yml"))
				}
			}
			return out
		},
	})
}

// determiners start a noun phrase.
var determiners = map[string]bool{
	"a": true, "an": true, "the": true, "this": true, "that": true,
	"these": true, "those": true, "each": true, "every": true, "any": true,
	"some": true, "no": true, "its": true, "his": true, "her": true,
	"their": true, "our": true, "your": true, "my": true, "both": true,
	"one": true, "two": true, "three": true, "four": true, "five": true,
	"another": true, "other": true, "new": true, "old": true, "same": true,
}

// verbHeads are the words a verb follows. A determiner before one of these
// belongs to an earlier noun phrase, not to the word being checked.
var verbHeads = map[string]bool{
	"will": true, "would": true, "can": true, "could": true, "shall": true,
	"should": true, "may": true, "might": true, "must": true, "to": true,
	"do": true, "does": true, "did": true, "not": true, "and": true, "or": true,
}

// nounPhraseWindow is how far back the check looks for a determiner.
const nounPhraseWindow = 3

// looksLikeNoun reports whether the token at i reads as the head of a noun
// phrase. A determiner a short way back, with no verb between, is the signal.
// This is a heuristic. The tool has no part-of-speech tagger, and the standard
// needs one to judge a word such as "filter", which is a permitted technical
// name as a noun and an unapproved word as a verb.
func looksLikeNoun(s Sentence, i int) bool {
	for k := i - 1; k >= 0 && k >= i-nounPhraseWindow; k-- {
		w := s.Tokens[k].Lower
		if beVerbs[w] || verbHeads[w] {
			return false
		}
		if determiners[w] {
			return true
		}
	}
	return false
}

// verbOnly reports whether the standard rejects the word for its verb sense and
// for no other. With no part of speech recorded, the answer is yes, so that the
// rule does not report a permitted technical name.
func verbOnly(pos []string) bool {
	if len(pos) == 0 {
		return true
	}
	sawVerb := false
	for _, p := range pos {
		switch p {
		case "v":
			sawVerb = true
		default:
			return false
		}
	}
	return sawVerb
}

// skipForDict drops the tokens a dictionary check cannot judge: a technical
// name the project allows, a number, a code placeholder, and a proper noun.
func skipForDict(s Sentence, i int, opt Options) bool {
	t := s.Tokens[i]
	if opt.Allow[t.Lower] {
		return true
	}
	if len(t.Lower) < 3 {
		return true
	}
	for _, r := range t.Text {
		if unicode.IsDigit(r) {
			return true
		}
	}
	// An inline code span becomes a run of the placeholder letter.
	if strings.Trim(t.Text, string(codeWord)) == "" {
		return true
	}
	// A capital letter away from the start of the sentence marks a name.
	if i > 0 && unicode.IsUpper(rune(t.Text[0])) {
		return true
	}
	return false
}
