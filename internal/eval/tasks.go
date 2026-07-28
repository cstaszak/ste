// Package eval runs the cross-model experiment: several writing tasks, each
// under several conditions, on one or more models. Each output is scored with
// the linter. The change between the baseline score and the STE score is the
// result.
package eval

// Task is one writing job given to the model.
type Task struct {
	ID     string
	Name   string
	Prompt string
	// Mode selects how the output is scored. A procedure or an error message
	// is scored in strict mode.
	Strict bool
}

// Tasks are the writing jobs of the experiment. They cover the text that a
// software project produces, which is where the STE rules apply.
var Tasks = []Task{
	{
		ID:   "readme",
		Name: "README introduction",
		Prompt: "Write the introduction section of a README for \"fluxcache\", " +
			"a semantic cache for large language model applications. It stores a " +
			"response and returns it again when a later prompt has a similar meaning, " +
			"even when the words differ. Write about 180 words. Return the text only.",
	},
	{
		ID:     "error",
		Name:   "Error message",
		Strict: true,
		Prompt: "Write the user-facing text for a rate-limit error. The API allows " +
			"100 requests each minute for each account. The response carries a " +
			"Retry-After header with the wait time. Tell the user what happened and " +
			"what to do. Write about 90 words. Return the text only.",
	},
	{
		ID:   "pr",
		Name: "Pull-request description",
		Prompt: "Write a pull-request description for a change that adds retries with " +
			"exponential backoff to an HTTP client. Before the change, one failed " +
			"call went straight to the caller. The change adds three attempts, a " +
			"delay that doubles, and a limit on the total wait. Write about 250 " +
			"words. Return the text only.",
	},
	{
		ID:     "runbook",
		Name:   "Runbook procedure",
		Strict: true,
		Prompt: "Write a runbook procedure for an on-call engineer to fail over a " +
			"PostgreSQL primary to its replica. Cover the checks before the change, " +
			"the steps, and how to confirm the result. Write about 200 words. " +
			"Return the text only.",
	},
	{
		ID:   "release",
		Name: "Release notes",
		Prompt: "Write release notes for version 2.4 of a command-line tool. It adds " +
			"a JSON output format, makes directory walks about twice as fast, fixes " +
			"a crash on an empty configuration file, and removes the --legacy flag. " +
			"Write about 150 words. Return the text only.",
	},
	{
		ID:   "doc",
		Name: "Documentation page",
		Prompt: "Write a documentation page that explains connection pooling to a " +
			"developer who has not used it. Cover what it does, why it helps, and " +
			"the two settings that matter most. Write about 220 words. " +
			"Return the text only.",
	},
}

// TaskByID returns a task by its ID.
func TaskByID(id string) (Task, bool) {
	for _, t := range Tasks {
		if t.ID == id {
			return t, true
		}
	}
	return Task{}, false
}
