package lint

import (
	"fmt"
	"strings"
)

func init() {
	register(&Rule{
		ID: "long-sentence", Category: CatSentences, Scored: true,
		Doc: "One instruction per sentence. STE caps a sentence at 20 words for a procedure and 25 for description.",
		Check: func(d *Document, opt Options) []Finding {
			r := registry["long-sentence"]
			var out []Finding
			for _, s := range d.Sentences {
				n := s.Words()
				if n <= opt.MaxSentenceWords {
					continue
				}
				out = append(out, finding(d, r, s.Start, truncate(s.Text, 60),
					fmt.Sprintf("sentence of %d words (limit %d)", n, opt.MaxSentenceWords),
					"split it into two sentences"))
			}
			return out
		},
	})

	register(&Rule{
		ID: "long-paragraph", Category: CatStructure, Scored: true,
		Doc: "One topic per paragraph, six sentences at most.",
		Check: func(d *Document, opt Options) []Finding {
			r := registry["long-paragraph"]
			var out []Finding
			for _, p := range d.Paragraphs {
				n := 0
				for _, s := range p.Sentences {
					if !s.InList {
						n++
					}
				}
				if n <= opt.MaxParagraphSentences {
					continue
				}
				out = append(out, finding(d, r, p.Start, "",
					fmt.Sprintf("paragraph of %d sentences (limit %d)", n, opt.MaxParagraphSentences),
					"split it, or make it a numbered list"))
			}
			return out
		},
	})

	register(&Rule{
		ID: "semicolon", Category: CatPunctuation, Scored: true,
		Doc: "STE does not allow the semicolon. Write two sentences.",
		Check: func(d *Document, opt Options) []Finding {
			r := registry["semicolon"]
			var out []Finding
			for i := 0; i < len(d.Masked); i++ {
				if d.Masked[i] == ';' {
					out = append(out, finding(d, r, i, ";", "semicolon", "write two sentences"))
				}
			}
			return out
		},
	})

	register(&Rule{
		ID: "em-dash", Category: CatPunctuation, Scored: true, DefaultOff: true,
		Doc: "STE does not ban the em dash, but heavy use of it marks generated text. Off by default.",
		Check: func(d *Document, opt Options) []Finding {
			r := registry["em-dash"]
			var out []Finding
			for _, dash := range []string{"—", "–"} {
				for i := 0; i < len(d.Masked); {
					j := strings.Index(d.Masked[i:], dash)
					if j < 0 {
						break
					}
					off := i + j
					out = append(out, finding(d, r, off, dash, "em dash",
						"use a comma, a period, or parentheses"))
					i = off + len(dash)
				}
			}
			return out
		},
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
