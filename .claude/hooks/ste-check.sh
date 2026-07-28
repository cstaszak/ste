#!/bin/sh
# PostToolUse hook: lint markdown that the agent just wrote.
#
# The hook is advisory. It never blocks a write. It exits 0 with no output when
# there is nothing to say, and exits 2 with the violations on standard error
# when there are some, which is how Claude Code feeds text back to the model.
#
# It does nothing if the ste binary is missing, so a fresh clone still works.

set -u

payload=$(cat)

# The hook receives JSON on standard input. Pull the file path out of it
# without a JSON parser, so the hook has no dependencies.
file=$(printf '%s' "$payload" |
	sed -n 's/.*"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
	head -1)

case "$file" in
*.md | *.markdown | *.mdx | *.txt) ;;
*) exit 0 ;;
esac

[ -f "$file" ] || exit 0

# Prefer a build in this repository, then the path.
root=${CLAUDE_PROJECT_DIR:-.}
if [ -x "$root/bin/ste" ]; then
	ste="$root/bin/ste"
elif command -v ste >/dev/null 2>&1; then
	ste=ste
else
	exit 0
fi

out=$("$ste" lint --format=agent --max-findings 12 "$file" 2>/dev/null) || true

case "$out" in
"" | "ste: clean.") exit 0 ;;
esac

printf 'ASD-STE100 writing check on %s:\n%s\n' "$file" "$out" >&2
printf 'Rewrite the lines above to remove these violations. See .claude/skills/ste-writing/SKILL.md.\n' >&2
exit 2
