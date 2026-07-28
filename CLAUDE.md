# CLAUDE.md

Guidance for Claude Code when it works in this repository.

## What this project is

`ste` is a Go command that checks prose against the machine-checkable subset of
ASD-STE100 Simplified Technical English. The score is violations per 100 words.

The project tests one claim. A model that follows a real writing system produces
much less "AI slop" than a model given a list of banned words. The reference work
is
[the-cure-for-ai-slop](https://github.com/woosal1337/blog/tree/main/videos/ep01-the-cure-for-ai-slop)
by woosal1337, which measured a 74% drop on Claude and a 50% drop on GPT-5.5.

## Commands

```
make build      # build ./bin/ste
make test       # go test ./...
make check      # gofmt check, go vet, and tests
make lint       # run ste on this repository's own documents
make all        # check and lint
```

## Layout

| Path | What it holds |
|---|---|
| `cmd/ste` | The command and its subcommands, one file for each |
| `internal/lint` | The parser, the rules, and the scoring engine |
| `internal/lint/data` | The word lists, embedded with `go:embed` |
| `internal/config` | The optional `.ste.yml` file |
| `internal/report` | The output formats |
| `internal/dict` | The approved-word index built from a local copy of the standard |
| `internal/eval` | The cross-model experiment |
| `testdata/corpus` | Documents with known scores |
| `docs/experiment-results.md` | The first-party run of `ste eval` and its analysis |
| `.claude/skills/ste-writing` | The writing skill for an agent |

## How the linter works

`Parse` does the work that every rule depends on. Read `internal/lint/tokenize.go`
before you change a rule.

1. `maskCode` hides code. A fenced block becomes spaces. An inline span becomes a
   run of the letter `X`, which reads as one word.
2. Both replacements keep the length of what they replace. Every byte offset in
   the masked text still points at the right place in the original file. This is
   what gives each finding a real line and column.
3. Lines group into chunks. A heading, a list item, and a table row are each their
   own chunk. Wrapped plain lines join into one chunk, so a hard-wrapped paragraph
   does not read as one sentence for each line.
4. Each chunk splits into sentences, and each sentence into tokens.

Rules match over tokens, not over raw text. Go regular expressions have no
lookbehind, so a direct port of the Python patterns would be wrong. Token matching
also gives exact offsets and correct handling of hyphenated terms such as
`cutting-edge`.

## How to add a rule

1. Add the rule to `rules_lexical.go`, `rules_syntax.go`, or `rules_structure.go`.
2. Call `register` in the `init` function of that file. Give the rule a kebab-case
   ID, a category, and a one-sentence `Doc`.
3. Set `Scored: true` if the rule counts toward the score. A rule that only marks a
   signal, such as `em-dash`, sets `Scored: false`.
4. Build each finding with the `finding` helper, and take quoted text with
   `d.span`. Both collapse whitespace.
5. Add a test to `internal/lint/lint_test.go`.
6. Run `make lint`. If the corpus scores move, update the numbers in
   `internal/lint/corpus_test.go` and say why in the commit message.

A word list needs no Go code. Add a line to the correct file in
`internal/lint/data`, in the form `phrase<TAB>replacement`.

## Rules for this repository

- **Every document here must pass `make lint`.** The tool has no value if its own
  documentation fails it. Run `./bin/ste lint <file>` after you write prose.
- **Never commit the ASD-STE100 specification, and never commit data derived from
  it.** The standard is under copyright and this repository is public. The PDF and
  the dictionary index are in `.gitignore`. Each machine builds its own dictionary
  index from a local copy of the standard.
- **Keep the lint core free of dependencies.** `internal/lint` uses the standard
  library only. A dependency belongs in `internal/config`, `internal/dict`, or
  `internal/eval`.
- **The score must stay deterministic.** The same input always gives the same
  score. Two runs that disagree make the before-and-after measurement worthless.

## Writing prose in this repository

Load the skill in `.claude/skills/ste-writing/SKILL.md` before you write
documentation, a commit message, or a pull-request description. Then check the
result:

```
./bin/ste lint --format=agent <file>
```

Fix what it reports and run it again.
