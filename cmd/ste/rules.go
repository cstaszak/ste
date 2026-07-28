package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/cstaszak/ste/internal/lint"
)

func runRules(args []string) int {
	fset := flag.NewFlagSet("rules", flag.ExitOnError)
	verbose := fset.Bool("v", false, "show the word list size of each rule")
	fset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ste rules [flags]\n\nFlags:\n")
		fset.PrintDefaults()
	}
	if err := fset.Parse(args); err != nil {
		return exitError
	}
	if fset.NArg() > 0 {
		return fail("rules takes no arguments, but found %s: %s",
			count(fset.NArg(), "argument"), strings.Join(fset.Args(), " "))
	}

	flavored := lint.DefaultOptions(lint.ModeFlavored)
	strict := lint.DefaultOptions(lint.ModeStrict)
	for _, r := range lint.Rules() {
		state := "off"
		switch {
		case flavored.Enabled[r.ID]:
			state = "on"
		case strict.Enabled[r.ID]:
			state = "strict"
		}
		score := "scored"
		if !r.Scored {
			score = "signal"
		}
		fmt.Printf("%-22s %-12s %-8s %-7s %s\n", r.ID, r.Category, state, score, r.Doc)
		if *verbose {
			if l, ok := flavored.Lists[r.ID]; ok {
				fmt.Printf("%-22s   %d phrases\n", "", l.Len())
			}
		}
	}
	return exitOK
}
