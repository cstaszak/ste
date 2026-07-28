package dict

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// The dictionary of the standard is a four-column table. A text extractor
// flattens the columns, so the parser works on one signal that survives the
// flattening: a headword followed by its part of speech in parentheses. An
// approved word is set in capitals. A word the standard does not approve is set
// in lower case.
var (
	// joinBrokenLines pulls together the character-level line breaks that the
	// extractor emits inside a table cell.
	joinBrokenLines = regexp.MustCompile(`[ \t]*\n[ \t]*`)

	partOfSpeech = `adj|adv|n|v|prep|conj|pron|det|art|int|pref|abbr|num|aux|part|inf`

	approvedRe   = regexp.MustCompile(`(?m)^([A-Z][A-Z0-9 \-/'.]{0,40}?)\s*\((` + partOfSpeech + `)[^)]*\)`)
	unapprovedRe = regexp.MustCompile(`(?m)^([a-z][a-z0-9 \-/'.]{0,40}?)\s*\((` + partOfSpeech + `)[^)]*\)`)

	// headwordRe finds both kinds in one pass, so their order on the page is
	// known. The alternatives of an unapproved word follow it.
	headwordRe = regexp.MustCompile(`(?m)^([A-Za-z][A-Za-z0-9 \-/'.]{0,40}?)\s*\((` + partOfSpeech + `)[^)]*\)`)
)

// maxAlternatives caps how many approved words the parser attributes to one
// unapproved word. The column flattening makes this a best effort.
const maxAlternatives = 4

// Build reads the specification and returns the index.
func Build(path string) (*Index, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	ix := NewIndex()
	ix.Source = path
	ix.Pages = r.NumPage()

	for pn := 1; pn <= r.NumPage(); pn++ {
		p := r.Page(pn)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue // a page that will not render is not fatal
		}
		ix.addPage(joinBrokenLines.ReplaceAllString(text, "\n"))
	}
	if len(ix.Approved) == 0 {
		return nil, fmt.Errorf("%s: found no dictionary entries; is this the ASD-STE100 specification?", path)
	}
	return ix, nil
}

// addPage parses one page of extracted text into the index.
func (ix *Index) addPage(text string) {
	matches := headwordRe.FindAllStringSubmatch(text, -1)

	// pending is the unapproved word whose alternatives may follow.
	var pending string
	for _, m := range matches {
		word := strings.TrimSpace(m[1])
		pos := m[2]
		if len(word) < 2 || isTableHeader(word) {
			continue
		}
		upper := word == strings.ToUpper(word)
		key := strings.ToLower(word)

		if upper {
			ix.addApproved(key, word, pos)
			if pending != "" {
				ix.addAlternative(pending, word)
			}
			continue
		}
		ix.addUnapproved(key, word, pos)
		pending = key
	}
}

// isTableHeader drops the column titles that repeat on every page.
func isTableHeader(w string) bool {
	switch strings.ToLower(w) {
	case "word", "alternatives", "approved meaning", "ste example", "example",
		"part of speech", "issue", "page", "part":
		return true
	}
	return false
}

func (ix *Index) addApproved(key, word, pos string) {
	e := ix.Approved[key]
	e.Word = word
	e.POS = addOnce(e.POS, pos)
	ix.Approved[key] = e
	// A word can appear as an alternative before it appears as a headword.
	// Approval wins, so drop any earlier unapproved record.
	delete(ix.Unapproved, key)
}

func (ix *Index) addUnapproved(key, word, pos string) {
	if _, ok := ix.Approved[key]; ok {
		return
	}
	e := ix.Unapproved[key]
	e.Word = word
	e.POS = addOnce(e.POS, pos)
	ix.Unapproved[key] = e
}

func (ix *Index) addAlternative(key, alt string) {
	e, ok := ix.Unapproved[key]
	if !ok || len(e.Alternatives) >= maxAlternatives {
		return
	}
	e.Alternatives = addOnce(e.Alternatives, alt)
	ix.Unapproved[key] = e
}

func addOnce(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

// CountEntries reports how many entries each regular expression finds. It is
// used by the build command to report what it saw.
func CountEntries(text string) (approved, unapproved int) {
	return len(approvedRe.FindAllString(text, -1)), len(unapprovedRe.FindAllString(text, -1))
}
