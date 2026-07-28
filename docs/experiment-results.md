# Does a writing system reduce AI slop?

This is the first-party test of the claim behind this project. Three models
wrote six documents under four system prompts. The linter scored every output.

Run on 2026-07-28. 72 requests, $0.55.

```
ste eval --models claude-opus-5,claude-sonnet-5,claude-haiku-4-5 --yes
```

The score is heuristic violations per 100 words. A lower score is cleaner.

## Result

| Condition | Claude Haiku 4.5 | Claude Opus 5 | Claude Sonnet 5 |
|---|---|---|---|
| baseline | 1.73 | 3.07 | 2.82 |
| banned-words list | 1.04 (-40%) | 2.07 (-33%) | 2.11 (-25%) |
| Orwell's six rules | 0.82 (-53%) | 1.07 (-65%) | 1.33 (-53%) |
| STE rules | 0.20 (-88%) | 0.09 (-97%) | 0.08 (-97%) |

The order is the same on all three models. Every writing system beats the
baseline. The banned-words list helps least. The STE rules help most.

The reference project measured a 74% drop on Claude and a 50% drop on GPT-5.5.
This run shows a larger effect.

## Read this before you trust the headline

The 97% carries a bias. The STE system prompt and the linter state the
same rules, and the same author wrote both. The prompt says "no contractions",
"no semicolons", "sentences of 25 words at most", and "use the active voice".
Those are four of the rules the linter checks.

A prompt that names the rules will beat a prompt that does not, on a checker
for those rules. The reference project has the same problem. Treat the order of
the conditions as the result. Do not treat the size of the drop as one.

The next section holds a result that does not have this problem.

## Slop is structural, not lexical

This table pools every violation across all 72 outputs.

| Condition | Total | passive-voice | long-sentence | contraction | semicolon | banned-word |
|---|---|---|---|---|---|---|
| baseline | 81 | 29 | 21 | 12 | 6 | 4 |
| banned-words list | 56 | 23 | 11 | 14 | 5 | 0 |
| Orwell's six rules | 34 | 9 | 8 | 10 | 6 | 0 |
| STE rules | 4 | 0 | 2 | 0 | 0 | 0 |

Banned words account for only 4 of the 81 baseline violations. The other 77 are
structural: passive voice, long sentences, contractions, and semicolons.

This explains why a banned-words list is the weakest fix. It targets the
smallest category. It cannot reach the other 95% of the problem. Orwell's rules
do better because two of the six are structural, and passive voice fell from 29
to 9.

The author of the word list does not control this result. The models chose what
to write.

## The banned-words list can make the score worse

Claude Sonnet 5 wrote a worse README under the banned-words list than under no
system prompt at all. The score went from 3.30 to 4.57.

The eight violations were five counts of passive voice, two long sentences, and
one contraction. The output contained no banned word. The list did its job, and
the score still went up.

## The STE outputs are shorter in violations, not in words

| Condition | Words | Violations |
|---|---|---|
| baseline | 3260 | 81 |
| banned-words list | 3354 | 56 |
| Orwell's six rules | 3216 | 34 |
| STE rules | 3354 | 4 |

The STE outputs are the same length as the baseline outputs. The rules do not
win by saying less.

## The writing gets better, not flatter

The largest change was the error message from Claude Opus 5, which went from
6.32 to 0.00.

Baseline:

```
**Too many requests**

You've reached the rate limit for your account. Our API accepts up to 100
requests per minute per account, and that limit has been exceeded, so this
request wasn't processed.

Check the `Retry-After` header in this response - it tells you how many seconds
to wait before trying again. Once that time has passed, your request should go
through normally.
```

STE rules:

```
**429 Too Many Requests**

Your account sent more than 100 requests in one minute. The API blocked this
request. The API did not change any data.

The response includes a Retry-After header. This header gives the wait time in
seconds.

To continue:

1. Read the Retry-After value from the response.
2. Wait for this number of seconds.
3. Send the request again.
```

The second version is the better error message. It opens with the status code.
It tells the reader that no data changed, which the first version never says.
It gives numbered steps. The rules did not flatten the writing here. They
removed the padding and left the facts.

## A smaller model starts cleaner

Claude Haiku 4.5 has the cleanest baseline of the three, at 1.73. Claude Opus 5
has the worst, at 3.07.

The larger models write longer sentences and reach for more decoration, so they
have further to fall. Haiku also gains least from the STE rules, because it
starts closer to them.

## By task

### Claude Haiku 4.5

| Task | baseline | banned-words | Orwell | STE |
|---|---|---|---|---|
| README introduction | 2.44 | 2.25 | 1.15 | 0.59 |
| Error message | 2.50 | 2.44 | 3.85 | 0.00 |
| Pull-request description | 2.36 | 0.41 | 0.42 | 0.00 |
| Runbook procedure | 0.00 | 0.00 | 0.69 | 0.60 |
| Release notes | 1.54 | 2.68 | 0.00 | 0.00 |
| Documentation page | 1.36 | 0.00 | 0.47 | 0.00 |

### Claude Opus 5

| Task | baseline | banned-words | Orwell | STE |
|---|---|---|---|---|
| README introduction | 5.35 | 2.78 | 1.59 | 0.53 |
| Error message | 6.32 | 3.23 | 2.20 | 0.00 |
| Pull-request description | 2.81 | 3.26 | 1.13 | 0.00 |
| Runbook procedure | 1.09 | 0.58 | 0.00 | 0.00 |
| Release notes | 3.27 | 2.52 | 0.64 | 0.00 |
| Documentation page | 1.68 | 0.43 | 1.32 | 0.00 |

### Claude Sonnet 5

| Task | baseline | banned-words | Orwell | STE |
|---|---|---|---|---|
| README introduction | 3.30 | 4.57 | 3.41 | 0.57 |
| Error message | 4.82 | 2.08 | 4.12 | 0.00 |
| Pull-request description | 4.12 | 2.50 | 1.43 | 0.00 |
| Runbook procedure | 1.69 | 1.44 | 0.46 | 0.00 |
| Release notes | 1.86 | 0.00 | 0.00 | 0.00 |
| Documentation page | 1.89 | 1.88 | 0.00 | 0.00 |

The linter scores the two procedural tasks, the error message and the runbook,
in strict mode. It scores the other four in flavored mode.

## Limits of this test

- One sample for each cell. There is no repeat run, so a single cell can move
  on noise. The pooled totals are firmer than any one number.
- The linter measures form. It cannot tell you whether a paragraph is true or
  whether it answers the question.
- Six tasks cover the text a software project writes. They say nothing about
  marketing copy, an essay, or fiction. STE removes voice on purpose.
- The tasks name a word count, so the models wrote to about the same length.
  A test without that instruction could give a different result.

## Reproduce it

```
make build
ste eval --list                        # the tasks, conditions, and models
ste eval --dry-run                     # the plan and the cost estimate
ste eval --models claude-opus-5,claude-sonnet-5,claude-haiku-4-5 --yes
```

The command writes `results/results.md` and every raw output under
`results/`. That directory is in `.gitignore`, so the outputs stay on your
machine.
