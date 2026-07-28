package lint

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed data/*.txt
var dataFS embed.FS

// PhraseList maps a tokenized phrase to its suggested replacement.
type PhraseList struct {
	entries map[string]string // space-joined lowercase tokens -> suggestion
	maxLen  int               // longest entry, in tokens
}

// NewPhraseList builds an empty list.
func NewPhraseList() *PhraseList {
	return &PhraseList{entries: map[string]string{}}
}

// Add puts one phrase in the list. An empty suggestion is allowed.
func (p *PhraseList) Add(phrase, suggest string) {
	toks := tokenizePhrase(phrase)
	if len(toks) == 0 {
		return
	}
	key := strings.Join(toks, " ")
	p.entries[key] = suggest
	if len(toks) > p.maxLen {
		p.maxLen = len(toks)
	}
}

// Remove takes one phrase out of the list.
func (p *PhraseList) Remove(phrase string) {
	delete(p.entries, strings.Join(tokenizePhrase(phrase), " "))
}

// Len returns the number of phrases in the list.
func (p *PhraseList) Len() int { return len(p.entries) }

// Phrases returns every phrase in the list, sorted.
func (p *PhraseList) Phrases() []string {
	out := make([]string, 0, len(p.entries))
	for k := range p.entries {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Clone copies the list so configuration can change it without touching the
// embedded defaults.
func (p *PhraseList) Clone() *PhraseList {
	c := &PhraseList{entries: make(map[string]string, len(p.entries)), maxLen: p.maxLen}
	for k, v := range p.entries {
		c.entries[k] = v
	}
	return c
}

// match reports the longest list entry that starts at token i, if any. It
// returns the number of tokens matched and the suggestion.
func (p *PhraseList) match(toks []Token, i int) (int, string, bool) {
	limit := p.maxLen
	if rest := len(toks) - i; limit > rest {
		limit = rest
	}
	// Prefer the longest match, so "prior to" wins over a bare "prior".
	for n := limit; n >= 1; n-- {
		var sb strings.Builder
		for k := 0; k < n; k++ {
			if k > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(toks[i+k].Lower)
		}
		if s, ok := p.entries[sb.String()]; ok {
			return n, s, true
		}
	}
	return 0, "", false
}

// scan walks every sentence and reports each list entry it finds.
func (p *PhraseList) scan(d *Document, r *Rule, verb string) []Finding {
	var out []Finding
	for _, s := range d.Sentences {
		for i := 0; i < len(s.Tokens); {
			n, suggest, ok := p.match(s.Tokens, i)
			if !ok {
				i++
				continue
			}
			text := d.span(s.Tokens[i].Start, s.Tokens[i+n-1].End)
			out = append(out, finding(d, r, s.Tokens[i].Start, text,
				fmt.Sprintf("%s: %q", verb, text), suggest))
			i += n
		}
	}
	return out
}

// loadPhraseList reads one embedded data file.
func loadPhraseList(name string) *PhraseList {
	b, err := dataFS.ReadFile("data/" + name)
	if err != nil {
		panic(err)
	}
	p := NewPhraseList()
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		phrase, suggest, _ := strings.Cut(line, "\t")
		p.Add(strings.TrimSpace(phrase), strings.TrimSpace(suggest))
	}
	return p
}

// The default word lists. Options.Lists copies these, so configuration edits
// never change the embedded defaults.
var defaultLists = map[string]*PhraseList{}

func init() {
	for id, file := range map[string]string{
		"banned-word":         "banned.txt",
		"marketing-adjective": "marketing.txt",
		"phrasal-verb":        "phrasal.txt",
		"modal-hedge":         "hedge.txt",
	} {
		defaultLists[id] = loadPhraseList(file)
	}

	register(&Rule{
		ID: "banned-word", Category: CatWords, Scored: true,
		Doc: "Use the short common word. STE gives one approved word for each meaning.",
		Check: func(d *Document, opt Options) []Finding {
			return opt.list("banned-word").scan(d, registry["banned-word"], "banned word")
		},
	})
	register(&Rule{
		ID: "marketing-adjective", Category: CatWords, Scored: true,
		Doc: "Technical text states facts. Marketing adjectives carry no information.",
		Check: func(d *Document, opt Options) []Finding {
			return opt.list("marketing-adjective").scan(d, registry["marketing-adjective"], "marketing word")
		},
	})
	register(&Rule{
		ID: "phrasal-verb", Category: CatVerbs, Scored: true,
		Doc: "Use one plain verb, not a verb plus a preposition.",
		Check: func(d *Document, opt Options) []Finding {
			return opt.list("phrasal-verb").scan(d, registry["phrasal-verb"], "phrasal verb")
		},
	})
	register(&Rule{
		ID: "modal-hedge", Category: CatWords, Scored: true,
		Doc: "Delete the hedge and state the fact.",
		Check: func(d *Document, opt Options) []Finding {
			return opt.list("modal-hedge").scan(d, registry["modal-hedge"], "hedge")
		},
	})
}
