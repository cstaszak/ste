package lint

import (
	"fmt"
	"strings"
)

var beVerbs = map[string]bool{
	"am": true, "is": true, "are": true, "was": true, "were": true,
	"be": true, "been": true, "being": true,
}

// irregularPP holds past participles that do not end in "ed".
var irregularPP = map[string]bool{
	"done": true, "made": true, "sent": true, "read": true, "built": true,
	"kept": true, "held": true, "set": true, "put": true, "run": true,
	"written": true, "shown": true, "given": true, "taken": true, "found": true,
	"got": true, "gotten": true, "seen": true, "known": true, "thrown": true,
	"drawn": true, "chosen": true, "driven": true, "left": true, "lost": true,
	"meant": true, "paid": true, "told": true,
}

// nominalVerbs are verbs that usually carry a nominalization behind them, as
// in "perform an analysis of" instead of "analyze".
var nominalVerbs = map[string]string{
	"perform": "use the verb for the action", "performs": "use the verb for the action",
	"performed": "use the verb for the action",
	"conduct":   "use the verb for the action", "conducts": "use the verb for the action",
	"conducted": "use the verb for the action",
	"provide":   "use the verb for the action", "provides": "use the verb for the action",
	"provided": "use the verb for the action",
}

// nominalSuffixes end an abstract noun that hides a verb.
var nominalSuffixes = []string{"tion", "ment", "ance", "ence"}

// concreteNouns end in a nominalization suffix but name a thing, not an action.
// "the function of the parser" hides no verb, so the rule must not fire on it.
var concreteNouns = map[string]bool{
	"function": true, "information": true, "position": true, "condition": true,
	"direction": true, "section": true, "question": true, "connection": true,
	"extension": true, "exception": true, "option": true, "version": true,
	"station": true, "portion": true, "fraction": true, "location": true,
	"population": true, "solution": true, "junction": true, "convention": true,
	"document": true, "environment": true, "argument": true, "instrument": true,
	"element": true, "moment": true, "department": true, "equipment": true,
	"component": true, "segment": true, "increment": true, "fragment": true,
	"distance": true, "instance": true, "balance": true, "appliance": true,
	"sentence": true, "reference": true, "sequence": true, "experience": true,
	"difference": true, "license": true, "presence": true, "absence": true,
	"science": true, "audience": true, "evidence": true, "influence": true,
}

// isParticiple reports whether a word can be a past participle.
func isParticiple(w string) bool {
	return strings.HasSuffix(w, "ed") || irregularPP[w]
}

func init() {
	register(&Rule{
		ID: "passive-voice", Category: CatVerbs, Scored: true,
		Doc: "Write in the active voice. Name the actor, then the action.",
		Check: func(d *Document, opt Options) []Finding {
			r := registry["passive-voice"]
			var out []Finding
			for _, s := range d.Sentences {
				for i := 0; i+1 < len(s.Tokens); i++ {
					if !beVerbs[s.Tokens[i].Lower] || !isParticiple(s.Tokens[i+1].Lower) {
						continue
					}
					text := d.span(s.Tokens[i].Start, s.Tokens[i+1].End)
					out = append(out, finding(d, r, s.Tokens[i].Start, text,
						fmt.Sprintf("passive voice: %q", text),
						"name the actor and use the active voice"))
					i++
				}
			}
			return out
		},
	})

	register(&Rule{
		ID: "ing-main-verb", Category: CatVerbs, Scored: true,
		Doc: "Use a simple tense where one works. Do not build the verb from \"be\" plus \"-ing\".",
		Check: func(d *Document, opt Options) []Finding {
			r := registry["ing-main-verb"]
			var out []Finding
			for _, s := range d.Sentences {
				for i := 0; i+1 < len(s.Tokens); i++ {
					next := s.Tokens[i+1].Lower
					if !beVerbs[s.Tokens[i].Lower] || !strings.HasSuffix(next, "ing") || len(next) < 5 {
						continue
					}
					text := d.span(s.Tokens[i].Start, s.Tokens[i+1].End)
					out = append(out, finding(d, r, s.Tokens[i].Start, text,
						fmt.Sprintf("\"-ing\" main verb: %q", text),
						"use the simple present or simple past"))
					i++
				}
			}
			return out
		},
	})

	register(&Rule{
		ID: "nominalization", Category: CatVerbs, Scored: true,
		Doc: "Use a verb for the action. Do not turn the action into a noun.",
		Check: func(d *Document, opt Options) []Finding {
			r := registry["nominalization"]
			var out []Finding
			for _, s := range d.Sentences {
				for i, t := range s.Tokens {
					if suggest, ok := nominalVerbs[t.Lower]; ok {
						out = append(out, finding(d, r, t.Start, t.Text,
							fmt.Sprintf("nominalization: %q", t.Text), suggest))
						continue
					}
					// "<abstract noun> of" hides a verb: "the execution of" -> "run".
					if i+1 < len(s.Tokens) && s.Tokens[i+1].Lower == "of" &&
						len(t.Lower) >= 8 && !concreteNouns[t.Lower] {
						for _, suf := range nominalSuffixes {
							if strings.HasSuffix(t.Lower, suf) {
								text := d.span(t.Start, s.Tokens[i+1].End)
								out = append(out, finding(d, r, t.Start, text,
									fmt.Sprintf("nominalization: %q", text),
									"replace the noun with its verb"))
								break
							}
						}
					}
				}
			}
			return out
		},
	})

	register(&Rule{
		ID: "contraction", Category: CatWords, Scored: true,
		Doc: "Do not use contractions. Write the words in full.",
		Check: func(d *Document, opt Options) []Finding {
			r := registry["contraction"]
			var out []Finding
			for _, s := range d.Sentences {
				for _, t := range s.Tokens {
					if suffix, ok := contractionSuffix(t.Lower); ok {
						out = append(out, finding(d, r, t.Start, t.Text,
							fmt.Sprintf("contraction: %q", t.Text),
							"write "+expandHint(suffix)))
					}
				}
			}
			return out
		},
	})
}

// contractionSuffix reports whether a word is a contraction, and returns the
// part after the apostrophe. A possessive such as "the parser's output" is not
// a contraction, so a bare "'s" does not count.
func contractionSuffix(w string) (string, bool) {
	for _, apo := range []string{"'", "’"} {
		head, tail, found := strings.Cut(w, apo)
		if !found || head == "" {
			continue
		}
		switch tail {
		case "t", "re", "ve", "ll", "d", "m":
			return tail, true
		case "s":
			// "it's", "that's", "here's" and "there's" are contractions.
			// Anything else with "'s" is a possessive.
			switch head {
			case "it", "that", "there", "here", "what", "who", "let":
				return tail, true
			}
		}
	}
	return "", false
}

func expandHint(suffix string) string {
	switch suffix {
	case "t":
		return "\"not\" in full"
	case "re":
		return "\"are\" in full"
	case "ve":
		return "\"have\" in full"
	case "ll":
		return "\"will\" in full"
	case "d":
		return "\"would\" or \"had\" in full"
	case "m":
		return "\"am\" in full"
	default:
		return "\"is\" or \"has\" in full"
	}
}
