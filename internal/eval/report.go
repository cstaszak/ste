package eval

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// WriteMarkdown prints the results table. The headline table gives one row for
// each condition and one column for each model, with the change from the
// baseline. A lower score is cleaner.
func WriteMarkdown(w io.Writer, results []Result, cfg Config) {
	sums := Summarize(results)

	// index[model][condition] = summary
	index := map[string]map[string]Summary{}
	var models []string
	for _, s := range sums {
		if _, ok := index[s.Model]; !ok {
			index[s.Model] = map[string]Summary{}
			models = append(models, s.Model)
		}
		index[s.Model][s.Condition] = s
	}
	sort.Strings(models)

	fmt.Fprintln(w, "# ASD-STE100 writing experiment")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%d writing tasks, %d conditions, %d models. Score is heuristic\n",
		len(cfg.Tasks), len(cfg.Conditions), len(models))
	fmt.Fprintln(w, "violations per 100 words. A lower score is cleaner.")
	fmt.Fprintln(w)

	// Headline table.
	fmt.Fprint(w, "| Condition |")
	for _, m := range models {
		fmt.Fprintf(w, " %s |", ModelByID(m).Name)
	}
	fmt.Fprintln(w)
	fmt.Fprint(w, "|---|")
	for range models {
		fmt.Fprint(w, "---|")
	}
	fmt.Fprintln(w)

	for _, c := range cfg.Conditions {
		fmt.Fprintf(w, "| %s |", c.Name)
		for _, m := range models {
			s, ok := index[m][c.ID]
			if !ok {
				fmt.Fprint(w, " - |")
				continue
			}
			cell := fmt.Sprintf("%.2f", s.Per100W)
			if base, ok := index[m]["baseline"]; ok && c.ID != "baseline" && base.Per100W > 0 {
				pct := (s.Per100W - base.Per100W) / base.Per100W * 100
				cell += fmt.Sprintf(" (%+.0f%%)", pct)
			}
			if s.Errors > 0 {
				cell += fmt.Sprintf(" [%d errors]", s.Errors)
			}
			fmt.Fprintf(w, " %s |", cell)
		}
		fmt.Fprintln(w)
	}

	// Per-task detail.
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## By task")
	fmt.Fprintln(w)
	for _, m := range models {
		if len(models) > 1 {
			fmt.Fprintf(w, "### %s\n\n", ModelByID(m).Name)
		}
		fmt.Fprint(w, "| Task |")
		for _, c := range cfg.Conditions {
			fmt.Fprintf(w, " %s |", c.Name)
		}
		fmt.Fprintln(w)
		fmt.Fprint(w, "|---|")
		for range cfg.Conditions {
			fmt.Fprint(w, "---|")
		}
		fmt.Fprintln(w)

		for _, t := range cfg.Tasks {
			fmt.Fprintf(w, "| %s |", t.Name)
			for _, c := range cfg.Conditions {
				cell := "-"
				for _, r := range results {
					if r.Model == m && r.Task == t.ID && r.Condition == c.ID {
						if r.Err != "" {
							cell = "error"
						} else {
							cell = fmt.Sprintf("%.2f", r.Report.Per100W)
						}
						break
					}
				}
				fmt.Fprintf(w, " %s |", cell)
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)
	}

	// Cost.
	total := 0.0
	for _, s := range sums {
		total += s.Cost
	}
	fmt.Fprintf(w, "Total cost: $%.2f over %d requests.\n", total, len(results))

	if errs := errorLines(results); len(errs) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## Errors")
		fmt.Fprintln(w)
		for _, e := range errs {
			fmt.Fprintf(w, "- %s\n", e)
		}
	}
}

func errorLines(results []Result) []string {
	var out []string
	for _, r := range results {
		if r.Err != "" {
			out = append(out, fmt.Sprintf("`%s` %s/%s: %s",
				r.Model, r.Task, r.Condition, strings.SplitN(r.Err, "\n", 2)[0]))
		}
	}
	return out
}
