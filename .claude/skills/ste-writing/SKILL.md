---
name: ste-writing
description: Write or rewrite prose (docs, READMEs, PR descriptions, commit messages, error messages, release notes, comments - never code) in ASD-STE100 Simplified Technical English, to remove "AI slop". Use when asked to make writing not sound like AI, to make docs clear or plain, to enforce a controlled writing style, or to write technical documentation that reads human. Two modes - strict (procedures, error messages) and flavored (general prose).
---

# ste-writing

Write prose in ASD-STE100 Simplified Technical English. This applies to
documentation, READMEs, pull-request text, commit messages, error messages,
release notes, and comments. It does not apply to code, identifiers, or command
syntax.

Do not use this skill for marketing copy, an essay, or anything that needs a
voice. STE removes voice on purpose.

## Modes

- **strict** - procedures, runbooks, safety text, error messages. Apply every
  rule. Sentences stop at 20 words.
- **flavored** - general prose such as a README or a pull-request description.
  Apply the sentence, paragraph, active-voice, and plain-verb rules. Sentences
  stop at 25 words. Do not lock the text to the 900-word STE dictionary, so the
  prose keeps enough range to read well.

Use flavored unless the text tells a person what to do.

## Rules

### Words

- Use one name for one thing. Do not give the same item two names.
- Use the short common word: start (not begin, commence, or initiate), use (not
  utilize or leverage), help (not facilitate), make sure (not ensure), before
  (not prior to), after (not subsequent to), about (not regarding), get (not
  obtain or acquire), show (not demonstrate), also (not additionally,
  furthermore, or moreover), many (not numerous, myriad, or plethora), to (not in
  order to), if (not in the event that), because (not due to the fact that).
- Give each word one meaning. "fall" means to move down. It does not mean to
  decrease.
- Use no marketing adjectives: seamless, robust, powerful, cutting-edge,
  effortless, world-class, next-generation, revolutionary, elegant, turnkey,
  best-in-class, enterprise-grade, battle-tested.
- Use American spelling.

### Verbs

- Use the active voice. Write "the parser reads the file", not "the file is read
  by the parser".
- Use a verb for an action. Write "analyze the log", not "perform an analysis of
  the log".
- Do not stack auxiliaries. Write "this improves X", not "it is important to note
  that this may help to improve X".
- Do not use an "-ing" main verb where a simple tense works.
- Do not use a phrasal verb. Write start, not spin up. Write contact, not reach
  out. Write examine, not dive into. Write release, not roll out.

### Sentences

- Give one instruction in one sentence.
- Keep a sentence to 20 words in strict mode, 25 words in flavored mode.
- Do not use contractions. Write "do not", not "don't".
- Use articles: a, an, the, this, these.

### Punctuation

- Do not use a semicolon. Write two sentences.
- The standard does not ban the em dash. This repository still asks you to avoid
  it, because heavy use of it marks generated text.

### Structure

- Give one topic to one paragraph, six sentences at most.
- For steps, use a numbered vertical list. Give one action to one item, in the
  imperative form.
- Put a condition before its command. Write "If the light is on, press the
  button".

Write only the text that was asked for. Add no preamble, no summary, and no
closing remark.

## Check your work

This repository has the linter that checks these rules. Run it on the file you
wrote:

```
ste lint --format=agent <file>
```

In strict mode:

```
ste lint --mode=strict --format=agent <file>
```

Fix every violation it reports, then run it again. If `ste` is not on the path,
build it with `make build` and use `./bin/ste`.

To measure a rewrite, keep the draft and compare:

```
ste diff draft.md rewrite.md
```

The score is violations per 100 words. The change between two scores is the
signal, not the score by itself.

If you cannot run the tool, do this check by hand:

1. Is any sentence longer than the limit? Split it.
2. Is there a semicolon? Replace it with a period.
3. Is there a contraction? Write it in full.
4. Is there passive voice with a known actor? Make it active.
5. Is there an "-ing" main verb, a nominalization ("perform an analysis"), or a
   phrasal verb ("spin up")? Use one plain verb.
6. Does one thing have two names? Choose one name.

## Limits

These rules are mechanical, and they remove the form of slop. Full ASD-STE100
also needs human judgment: the right technical noun, and whether a sentence makes
good sense. No checker can certify that. These rules cannot make an empty
paragraph true.

The standard is free at https://asd-ste100.org. It is under copyright. Do not
copy it into this repository.
