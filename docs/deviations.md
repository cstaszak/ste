# Deviations from the Python reference linter

`ste` started as a port of `ste-lint.py` from
[the-cure-for-ai-slop](https://github.com/woosal1337/blog/tree/main/videos/ep01-the-cure-for-ai-slop).
The rules are the same. The scores are not always the same. This page records
each difference and the reason for it.

The reference numbers stay valid as a comparison between two texts scored by the
same tool. Do not compare a `ste` score with a `ste-lint.py` score.

## Matching

**Tokens, not regular expressions.** The Python version matches word lists with
`(?<![a-z])phrase(?![a-z])` on lowercased text. Go regular expressions have no
lookbehind, so a direct port is not possible. `ste` tokenizes the text once and
matches phrases over the token list. This finds the same words. It also gives an
exact byte offset for each match, which the Python version cannot give.

**Longest match wins.** In a list that holds both `prior` and `prior to`, `ste`
reports `prior to` once. The Python version can report an overlap twice.

## Counting

**No double count of the hedge phrase.** The Python lists hold the phrase
`it is important to note` in both `BANNED` and `MODAL_HEDGE`, so it scores
twice. In `ste` that phrase belongs to the `modal-hedge` rule only.

**The em dash does not score.** The Python version reports the em dash count
next to the total but leaves it out of the total. `ste` does the same. The
`em-dash` rule is off by default. Turn it on in `.ste.yml`. The count always
appears in the report.

**A list item does not count toward the paragraph limit.** STE asks for a
procedure as a numbered vertical list. The Python version counts each item as a
sentence in the paragraph. A correct nine-step procedure then fails the
six-sentence limit. `ste` counts only the sentences outside list items.

## Parsing

**Sentences cross line breaks.** The Python version splits sentences inside one
line at a time, so a hard-wrapped paragraph reads as one sentence for each line.
Long sentences then hide. `ste` joins wrapped lines into one chunk first. It finds
a 40-word sentence written over three lines.

**A heading, a list item, and a table row are each their own chunk.** Without
this, a markdown table joins into one very long sentence.

**Inline code is one word.** The Python version deletes code before it splits
sentences. A sentence that starts with inline code then joins to the sentence
before it. `ste` replaces an inline span with a placeholder word of the same
length, so the sentence boundary survives and the span counts as one word. A
fenced block still becomes blank and counts as no words.

## Rules

**A possessive is not a contraction.** The Python pattern counts `parser's` as a
contraction. `ste` counts `'s` only after a pronoun or a determiner, as in `it's`
and `that's`.

**A concrete noun is not a nominalization.** The Python rule fires on any word
of four or more letters that ends in `tion`, `ment`, `ance`, or `ence` before the
word `of`. That rule reports `the function of the parser`. `ste` holds a list of
nouns that name a thing rather than an action, and skips them.

**More irregular participles.** `ste` adds `chosen`, `driven`, `left`, `lost`,
`meant`, `paid`, and `told` to the passive-voice check.

**Each rule carries a replacement.** The word lists in `internal/lint/data` hold
a suggested replacement next to each phrase, so a report says what to write
instead.

## Sentence length

The Python version uses one limit of 20 words. `ste` follows the two limits in
the standard. An instruction stops at 20 words in strict mode. Descriptive text
stops at 25 words in flavored mode. You can change both limits in `.ste.yml`.
