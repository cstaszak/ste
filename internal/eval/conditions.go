package eval

import (
	"embed"
	"strings"
)

//go:embed data/*.txt
var dataFS embed.FS

// Condition is one system prompt under test.
type Condition struct {
	ID   string
	Name string
	// System is the system prompt. An empty string means no system prompt.
	System string
}

// Conditions are the four system prompts of the experiment, in the order the
// results table shows them. The baseline comes first, so each later condition
// is measured against it.
var Conditions = []Condition{
	{ID: "baseline", Name: "baseline", System: ""},
	{ID: "banned-words", Name: "banned-words list", System: mustRead("banned-words.txt")},
	{ID: "orwell", Name: "Orwell's six rules", System: mustRead("orwell.txt")},
	{ID: "ste", Name: "STE rules", System: mustRead("ste.txt")},
}

// ConditionByID returns a condition by its ID.
func ConditionByID(id string) (Condition, bool) {
	for _, c := range Conditions {
		if c.ID == id {
			return c, true
		}
	}
	return Condition{}, false
}

func mustRead(name string) string {
	b, err := dataFS.ReadFile("data/" + name)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(b))
}
