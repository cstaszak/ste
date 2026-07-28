# ste

`ste` checks prose against the machine-checkable subset of ASD-STE100 Simplified
Technical English. It is one Go binary with no runtime dependencies.

ASD-STE100 is a controlled writing standard from 1986. Aircraft makers wrote it so
that a maintenance manual reads the same way to every engineer. The same rules
remove most of what people call "AI slop": marketing adjectives, hedges, passive
voice, nominalizations, and long stacked sentences.

This project comes from
[the-cure-for-ai-slop](https://github.com/woosal1337/blog/tree/main/videos/ep01-the-cure-for-ai-slop)
by woosal1337. The video is
[The cure for AI slop is a 1986 aircraft manual](https://www.youtube.com/watch?v=uJblcC4lKYw).

That project shows the effect with numbers. A model given the STE rules cut its
violation rate by 74% on Claude and 50% on GPT-5.5. `ste` is a Go rewrite of the
Python linter in that kit. It adds locations, configuration, more output formats,
and a dictionary check.

## Install

```
go install github.com/cstaszak/ste/cmd/ste@latest
```

Or build from a clone:

```
make build      # writes ./bin/ste
```

## Use

```
ste lint docs/                  # walk a directory
ste lint README.md              # one file
cat draft.md | ste lint         # standard input
ste lint --mode=strict runbook.md
ste diff before.md after.md     # the score change between two drafts
ste rules                       # the list of rules
```

The score is violations per 100 words. A lower score is cleaner. One score on its
own means little. Lint a draft, apply the writing rules, then lint it again. The
change between the two scores is the signal.

### Modes

`ste` has the same two modes as the writing skill.

- `flavored` is the default. It applies the structure, voice, and word rules to
  general prose. Sentences can go to 25 words.
- `strict` is for procedures, runbooks, and error messages. Sentences stop at 20
  words, and the dictionary check runs if an index is present.

### Output formats

| Format | Use |
|---|---|
| `text` | The default. One line for each violation, with the line and column. |
| `summary` | One line for each file. |
| `json` | Full detail for a script. |
| `github` | Workflow annotations for a pull request. |
| `sarif` | For a code scanner. |
| `agent` | A short form for a coding agent to act on. |

### The dictionary check

ASD-STE100 approves about 900 words for general use. `ste` can check your text
against that list, but it needs the standard, and the standard is under
copyright. Get your own copy, then build the index:

```
ste dict build --pdf ASD-STE100-ISSUE-9.pdf
ste dict stats
ste dict lookup utilize commence run
```

The index goes in your user cache directory, never in the repository. Ask for
the check with `--dict`:

```
ste lint --mode=strict --dict runbook.md
```

Use it for a procedure, a runbook, or an error message. Do not use it for a
README. General prose needs words that STE does not approve, which is why
flavored mode leaves the dictionary out.

Two limits are worth knowing:

- The standard permits a **technical name**, and it does not list them. `ste`
  reads a word as a noun when a determiner comes shortly before it. It then
  allows a word that the standard rejects for its verb sense only. So "remove
  the filter" passes and "filter the fuel" does not. This is a guess, not a
  parser. Put the names your project uses in `allow` in `.ste.yml`.
- The parser reads a four-column table out of a PDF. Run `ste dict stats` after
  a build. An approved count near 800 to 900 is right. A count far from that
  means the parse failed.

### In a build

```
ste lint --max-per100w 2.0 docs/
```

The command returns one of three codes:

- `0` when every file is at or below the limit.
- `1` when a file is above the limit.
- `2` when the tool cannot run.

## Configuration

Put `.ste.yml` in the repository root. Every field is optional. `ste` searches for
the file from the first input path up to the root of the filesystem.

```yaml
mode: flavored
max_per_100w: 2.0
max_sentence_words: 25
max_paragraph_sentences: 6

rules:
  em-dash: true          # off by default
  long-paragraph: false

words:
  banned-word:
    add:
      "going forward": "from now on"
    remove: [begin, begins]

allow: [Kubernetes, goroutine]   # technical names the dictionary check accepts
```

### The experiment

`ste eval` reproduces the cross-model test. It gives each writing task to each
model under four system prompts, scores every output with the linter, and writes
a table.

```
ste eval --list                        # the tasks, conditions, and models
ste eval --dry-run                     # the plan and the cost estimate
ste eval --models claude-opus-5 --yes  # run it
```

The four conditions are: no system prompt, a list of banned words, Orwell's six
rules, and the STE rules. The change from the baseline is the result.

The command needs `ANTHROPIC_API_KEY`, or another credential the Anthropic SDK
accepts. **It spends money.** The command prints an estimate and stops unless you
add `--yes`. Raw outputs go under `results/`, which `.gitignore` excludes.

## For Claude and other coding agents

`.claude/skills/ste-writing/SKILL.md` holds the writing rules as a skill. A
`PostToolUse` hook in `.claude/settings.json` runs `ste lint` on each markdown file
the agent writes and returns the violations. The agent then corrects its own draft.
The hook does not block a write. It skips without an error if `ste` is not on the
path.

## What this is not

This is not a certified ASD-STE100 checker. Some rules of the standard need a
human. A person must choose the right technical noun, keep one name for one thing
across a document, and judge whether a sentence makes good sense. `ste` covers the
mechanical subset, which is where the slop lives.

You can get the standard at no cost from
[asd-ste100.org](https://asd-ste100.org). The standard stays under copyright. This
repository holds no part of the specification, and no data derived from it.

## License

MIT. See [LICENSE](LICENSE).
