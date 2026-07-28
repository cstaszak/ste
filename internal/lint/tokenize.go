package lint

import (
	"regexp"
	"strings"
	"unicode"
)

// Position is a 1-based line and column in the original document.
type Position struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

// Token is one word of prose with its location in the original document.
type Token struct {
	Text  string // as written
	Lower string // lowercased, for matching
	Start int    // byte offset in the original document
	End   int    // byte offset one past the last byte
}

// Sentence is one unit of prose. Start is the byte offset of the first
// character in the original document.
type Sentence struct {
	Text  string
	Start int
	// InList marks a sentence that came from a bullet or numbered list item.
	// STE asks for procedures as vertical lists, so list items do not count
	// against the paragraph length limit.
	InList bool
	Tokens []Token
}

// Words returns the number of words in the sentence.
func (s Sentence) Words() int { return len(s.Tokens) }

// Paragraph is a run of lines with no blank line between them.
type Paragraph struct {
	Start     int
	Sentences []Sentence
}

// Document holds the parsed form of one input file.
type Document struct {
	Raw        string // the original text, unchanged
	Masked     string // code spans replaced by spaces; byte offsets still line up
	Sentences  []Sentence
	Paragraphs []Paragraph
	lineStarts []int
}

// Words returns the total word count of the document.
func (d *Document) Words() int {
	n := 0
	for _, s := range d.Sentences {
		n += s.Words()
	}
	return n
}

// Position converts a byte offset into a 1-based line and column.
func (d *Document) Position(offset int) Position {
	if offset < 0 {
		offset = 0
	}
	// Binary search for the last line start at or before offset.
	lo, hi := 0, len(d.lineStarts)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if d.lineStarts[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return Position{Line: lo + 1, Col: offset - d.lineStarts[lo] + 1}
}

// Line returns the text of the given 1-based line number.
func (d *Document) Line(n int) string {
	if n < 1 || n > len(d.lineStarts) {
		return ""
	}
	start := d.lineStarts[n-1]
	end := len(d.Raw)
	if n < len(d.lineStarts) {
		end = d.lineStarts[n] - 1
	}
	if end > len(d.Raw) {
		end = len(d.Raw)
	}
	return strings.TrimRight(d.Raw[start:end], "\r")
}

var (
	fencedCode = regexp.MustCompile("(?s)```.*?```")
	inlineCode = regexp.MustCompile("`[^`\n]*`")
	wordRe     = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9'\x{2019}\-/]*`)
	headingRe  = regexp.MustCompile(`^\s*#{1,6}\s*`)
	listRe     = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+`)
	tableRowRe = regexp.MustCompile(`^\s*\|`)
)

// normalizeSpace collapses every run of whitespace to one space, so a finding
// that spans a line break still prints on one line.
func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// span returns the original text between two byte offsets, on one line.
func (d *Document) span(start, end int) string {
	if start < 0 || end > len(d.Raw) || start >= end {
		return ""
	}
	return normalizeSpace(d.Raw[start:end])
}

// codeWord stands in for one inline code span.
const codeWord = 'X'

// maskCode hides code from the rules. A fenced block becomes spaces, because it
// is not prose and must not count as words. An inline span becomes one
// placeholder word, because it is part of the sentence around it and can start
// a sentence. Both replacements keep the length of what they replace, so every
// byte offset in the masked text still lines up with the original. Findings can
// then point at real line and column numbers.
func maskCode(s string) string {
	b := []byte(s)
	fill := func(start, end int, c byte) {
		for i := start; i < end; i++ {
			if b[i] != '\n' {
				b[i] = c
			}
		}
	}
	for _, loc := range fencedCode.FindAllStringIndex(s, -1) {
		fill(loc[0], loc[1], ' ')
	}
	// Re-scan the masked text so backticks inside a fence are already gone.
	masked := string(b)
	for _, loc := range inlineCode.FindAllStringIndex(masked, -1) {
		fill(loc[0], loc[1], codeWord)
		b[loc[0]] = ' ' // the opening backtick
		b[loc[1]-1] = ' '
	}
	return string(b)
}

// Parse builds a Document from raw text.
func Parse(raw string) *Document {
	d := &Document{Raw: raw, Masked: maskCode(raw)}
	d.lineStarts = []int{0}
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\n' {
			d.lineStarts = append(d.lineStarts, i+1)
		}
	}
	d.Sentences = sentencesIn(d.Masked, 0)
	d.Paragraphs = paragraphsIn(d.Masked)
	return d
}

// paragraphsIn splits masked text on blank lines and parses each block.
func paragraphsIn(masked string) []Paragraph {
	var out []Paragraph
	for _, b := range splitBlankLines(masked) {
		if strings.TrimSpace(b.text) == "" {
			continue
		}
		out = append(out, Paragraph{
			Start:     b.start,
			Sentences: sentencesIn(b.text, b.start),
		})
	}
	return out
}

type block struct {
	text  string
	start int
}

var blankLine = regexp.MustCompile(`\n[ \t]*\n`)

func splitBlankLines(s string) []block {
	var out []block
	prev := 0
	for _, loc := range blankLine.FindAllStringIndex(s, -1) {
		out = append(out, block{text: s[prev:loc[0]], start: prev})
		prev = loc[1]
	}
	out = append(out, block{text: s[prev:], start: prev})
	return out
}

// chunk is a run of lines that belong to one block of prose: a heading, one
// list item and its continuation lines, or a group of wrapped plain lines.
// Sentences are found within a chunk, not within a line, so a hard-wrapped
// paragraph does not read as one sentence per line.
type chunk struct {
	start   int // byte offset in text of the first line
	end     int // byte offset one past the last line
	inList  bool
	inTable bool
	heading bool
	blanks  [][2]int // marker ranges to blank out, relative to text
}

// sentencesIn splits text into sentences. base is the byte offset of text
// within the document, so sentence offsets stay absolute.
func sentencesIn(text string, base int) []Sentence {
	var out []Sentence
	var cur *chunk

	flush := func() {
		if cur == nil {
			return
		}
		// Copy the byte range and blank the newlines and markers. Blanking
		// rather than deleting keeps every offset aligned with the document.
		buf := []byte(text[cur.start:cur.end])
		for i := range buf {
			if buf[i] == '\n' || buf[i] == '\r' {
				buf[i] = ' '
			}
		}
		for _, r := range cur.blanks {
			for i := r[0] - cur.start; i < r[1]-cur.start && i < len(buf); i++ {
				buf[i] = ' '
			}
		}
		for _, s := range splitSentence(string(buf), base+cur.start) {
			s.InList = cur.inList
			out = append(out, s)
		}
		cur = nil
	}

	pos := 0
	for _, raw := range strings.SplitAfter(text, "\n") {
		if raw == "" {
			break
		}
		start := pos
		pos += len(raw)
		body := strings.TrimRight(raw, "\r\n")

		if strings.TrimSpace(body) == "" {
			flush()
			continue
		}

		isHeading := headingRe.MatchString(body)
		listMarker := listRe.FindString(body)
		isList := listMarker != ""
		// A table row is one block of its own. Joining rows would read as one
		// very long sentence.
		isTableRow := tableRowRe.MatchString(body)

		// A heading, a list item, or a table row always starts a chunk. So does
		// a plain line that follows a heading or a table row.
		if isHeading || isList || isTableRow || (cur != nil && (cur.heading || cur.inTable)) {
			flush()
		}
		if cur == nil {
			cur = &chunk{start: start, inList: isList || isTableRow, inTable: isTableRow, heading: isHeading}
		}
		cur.end = start + len(body)

		// Record the marker so it does not become part of a sentence.
		lead := len(body) - len(strings.TrimLeft(body, " \t"))
		if lead > 0 {
			cur.blanks = append(cur.blanks, [2]int{start, start + lead})
		}
		if isHeading {
			cur.blanks = append(cur.blanks, [2]int{start, start + len(headingRe.FindString(body))})
		}
		if isList {
			cur.blanks = append(cur.blanks, [2]int{start, start + len(listMarker)})
		}
	}
	flush()
	return out
}

// closingMarkup holds the characters that can sit between the end of a sentence
// and the space after it, as in a bold run-in heading: "**Do this.** Then...".
const closingMarkup = "*_`\")]}'"

// openingMarkup holds the characters a sentence can start with.
const openingMarkup = "*_`\"'([-"

// splitSentence breaks one line at sentence-ending punctuation followed by
// whitespace and a capital letter, digit, or markup. Go's regexp package has no
// lookbehind, so the scan is explicit.
func splitSentence(line string, base int) []Sentence {
	var out []Sentence
	start := 0
	emit := func(end int) {
		raw := line[start:end]
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return
		}
		lead := strings.Index(raw, trimmed)
		out = append(out, newSentence(trimmed, base+start+lead))
	}
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c != '.' && c != '!' && c != '?' && c != ':' {
			continue
		}
		// Step over any closing markup that follows the punctuation.
		j := i + 1
		for j < len(line) && strings.IndexByte(closingMarkup, line[j]) >= 0 {
			j++
		}
		markup := j
		for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			j++
		}
		if j == markup || j >= len(line) {
			continue // punctuation must be followed by whitespace, then more text
		}
		// Step over any opening markup on the next sentence.
		k := j
		for k < len(line) && strings.IndexByte(openingMarkup, line[k]) >= 0 {
			k++
		}
		if k >= len(line) {
			continue
		}
		r := rune(line[k])
		if !(unicode.IsUpper(r) || unicode.IsDigit(r)) && k == j {
			continue
		}
		emit(i + 1)
		start = j
		i = j - 1
	}
	emit(len(line))
	return out
}

func newSentence(text string, start int) Sentence {
	s := Sentence{Text: text, Start: start}
	for _, loc := range wordRe.FindAllStringIndex(text, -1) {
		w := text[loc[0]:loc[1]]
		s.Tokens = append(s.Tokens, Token{
			Text:  w,
			Lower: strings.ToLower(w),
			Start: start + loc[0],
			End:   start + loc[1],
		})
	}
	return s
}

// tokenizePhrase splits a rule phrase the same way prose is tokenized, so
// phrase matching and word matching agree on what a word is.
func tokenizePhrase(p string) []string {
	var out []string
	for _, w := range wordRe.FindAllString(strings.ToLower(p), -1) {
		out = append(out, w)
	}
	return out
}
