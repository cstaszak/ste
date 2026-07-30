package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/stazelabs/ste/internal/config"
	"github.com/stazelabs/ste/internal/lint"
)

// Delta is the score change between two drafts.
type Delta struct {
	Before   *lint.Report   `json:"before"`
	After    *lint.Report   `json:"after"`
	Change   float64        `json:"change_per_100w"`
	Percent  float64        `json:"percent_change"`
	ByRule   map[string]int `json:"by_rule_change"`
	Improved bool           `json:"improved"`
}

func runDiff(args []string) int {
	fset := flag.NewFlagSet("diff", flag.ExitOnError)
	var (
		mode    = fset.String("mode", "", "flavored or strict (default flavored)")
		asJSON  = fset.Bool("json", false, "print the result as JSON")
		require = fset.Float64("require-drop", 0, "fail unless the score falls by at least this many points")
	)
	fset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ste diff [flags] <before> <after>\n\n"+
			"Reports the change in violations per 100 words between two drafts.\n"+
			"The change is the signal: lint a draft, apply the writing rules, lint it again.\n\nFlags:\n")
		fset.PrintDefaults()
	}
	if err := fset.Parse(args); err != nil {
		return exitError
	}
	if fset.NArg() != 2 {
		fset.Usage()
		return exitError
	}

	cfg, err := config.Load(config.Find(filepath.Dir(fset.Arg(0))))
	if err != nil {
		return fail("%v", err)
	}
	m, err := cfg.ResolveMode(*mode)
	if err != nil {
		return fail("%v", err)
	}
	opt := cfg.Apply(lint.DefaultOptions(m))

	reports := make([]*lint.Report, 2)
	for i, p := range []string{fset.Arg(0), fset.Arg(1)} {
		b, err := os.ReadFile(p)
		if err != nil {
			return fail("%v", err)
		}
		reports[i] = lint.RunText(string(b), opt)
		reports[i].Path = p
	}

	d := newDelta(reports[0], reports[1])
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(d); err != nil {
			return fail("%v", err)
		}
	} else {
		printDelta(d)
	}

	if *require > 0 && -d.Change < *require {
		return exitFound
	}
	return exitOK
}

func newDelta(before, after *lint.Report) *Delta {
	d := &Delta{
		Before: before,
		After:  after,
		Change: round2(after.Per100W - before.Per100W),
		ByRule: map[string]int{},
	}
	if before.Per100W > 0 {
		d.Percent = round2((after.Per100W - before.Per100W) / before.Per100W * 100)
	}
	for id, n := range after.Counts {
		if diff := n - before.Counts[id]; diff != 0 {
			d.ByRule[id] = diff
		}
	}
	for id, n := range before.Counts {
		if _, ok := after.Counts[id]; !ok {
			d.ByRule[id] = -n
		}
	}
	d.Improved = d.Change < 0
	return d
}

func printDelta(d *Delta) {
	fmt.Printf("before  %-40s %6.2f per 100 words (%d violations, %d words)\n",
		d.Before.Path, d.Before.Per100W, d.Before.Total, d.Before.Words)
	fmt.Printf("after   %-40s %6.2f per 100 words (%d violations, %d words)\n",
		d.After.Path, d.After.Per100W, d.After.Total, d.After.Words)

	verdict := "no change"
	switch {
	case d.Change < 0:
		verdict = "cleaner"
	case d.Change > 0:
		verdict = "worse"
	}
	fmt.Printf("change  %+.2f per 100 words (%+.0f%%) - %s\n", d.Change, d.Percent, verdict)

	if len(d.ByRule) == 0 {
		return
	}
	ids := make([]string, 0, len(d.ByRule))
	for id := range d.ByRule {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return d.ByRule[ids[i]] < d.ByRule[ids[j]] })
	fmt.Println("\nby rule:")
	for _, id := range ids {
		fmt.Printf("  %-22s %+d\n", id, d.ByRule[id])
	}
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
