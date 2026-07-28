package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/cstaszak/ste/internal/eval"
)

func runEval(args []string) int {
	fset := flag.NewFlagSet("eval", flag.ExitOnError)
	var (
		models     = fset.String("models", "claude-opus-5", "comma-separated model IDs")
		taskIDs    = fset.String("tasks", "", "comma-separated task IDs (default: all)")
		condIDs    = fset.String("conditions", "", "comma-separated condition IDs (default: all)")
		outDir     = fset.String("out", "results", "directory for the raw outputs and the table")
		maxTokens  = fset.Int64("max-tokens", 8000, "cap on each response, thinking and text together")
		concurrent = fset.Int("concurrency", 4, "how many requests run at one time")
		yes        = fset.Bool("yes", false, "run without asking; this command spends money")
		dryRun     = fset.Bool("dry-run", false, "print the plan and the cost estimate, then stop")
		asJSON     = fset.Bool("json", false, "also write results.json with every output")
		listOnly   = fset.Bool("list", false, "list the tasks, conditions, and models, then stop")
	)
	fset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ste eval [flags]\n\n"+
			"Runs each writing task under each condition on each model, scores every\n"+
			"output with the linter, and writes a results table. The change between\n"+
			"the baseline score and the STE score is the result.\n\n"+
			"Needs ANTHROPIC_API_KEY. This command spends money.\n\nFlags:\n")
		fset.PrintDefaults()
	}
	if err := fset.Parse(args); err != nil {
		return exitError
	}
	// eval takes no positional arguments. A list flag given with spaces instead
	// of commas leaves the extra values here, where they would be dropped
	// without a word. The Go flag package also stops at the first argument, so
	// every flag after it is lost too. Report both, and show the command that
	// works.
	if fset.NArg() > 0 {
		var values, lost []string
		for _, a := range fset.Args() {
			if strings.HasPrefix(a, "-") {
				lost = append(lost, a)
			} else {
				values = append(values, a)
			}
		}
		fmt.Fprintf(os.Stderr, "ste: eval takes no arguments, but found %s: %s\n",
			count(fset.NArg(), "argument"), strings.Join(fset.Args(), " "))
		if len(lost) > 0 {
			fmt.Fprintf(os.Stderr, "The flags after the first argument were not read: %s\n",
				strings.Join(lost, " "))
		}
		if len(values) > 0 {
			cmd := "ste eval --models " + strings.Join(append([]string{*models}, values...), ",")
			if len(lost) > 0 {
				cmd += " " + strings.Join(lost, " ")
			}
			fmt.Fprintf(os.Stderr, "A list flag needs commas and no spaces. Did you mean:\n  %s\n", cmd)
		}
		return exitError
	}

	if *listOnly {
		listEval()
		return exitOK
	}

	cfg := eval.DefaultConfig()
	cfg.Models = splitList(*models)
	cfg.MaxTokens = *maxTokens
	cfg.Concurrency = *concurrent
	cfg.OutDir = *outDir

	if *taskIDs != "" {
		cfg.Tasks = nil
		for _, id := range splitList(*taskIDs) {
			t, ok := eval.TaskByID(id)
			if !ok {
				return fail("unknown task %q (run \"ste eval --list\")", id)
			}
			cfg.Tasks = append(cfg.Tasks, t)
		}
	}
	if *condIDs != "" {
		cfg.Conditions = nil
		for _, id := range splitList(*condIDs) {
			c, ok := eval.ConditionByID(id)
			if !ok {
				return fail("unknown condition %q (run \"ste eval --list\")", id)
			}
			cfg.Conditions = append(cfg.Conditions, c)
		}
	}
	if len(cfg.Models) == 0 || len(cfg.Tasks) == 0 || len(cfg.Conditions) == 0 {
		return fail("nothing to run")
	}

	fmt.Fprintf(os.Stderr, "Plan: %s x %s x %s = %s\n",
		count(len(cfg.Models), "model"),
		count(len(cfg.Tasks), "task"),
		count(len(cfg.Conditions), "condition"),
		count(cfg.Calls(), "request"))
	for _, id := range cfg.Models {
		fmt.Fprintf(os.Stderr, "  %s\n", id)
	}
	fmt.Fprintf(os.Stderr, "Estimated cost: about $%.2f\n", cfg.Estimate())

	if *dryRun {
		return exitOK
	}
	if !*yes {
		fmt.Fprintf(os.Stderr, "\nThis command spends money. Add --yes to run it.\n")
		return exitError
	}
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		fmt.Fprintf(os.Stderr,
			"\nNote: ANTHROPIC_API_KEY is not set. The client will try the other\n"+
				"credential sources, such as a profile from \"ant auth login\".\n\n")
	}

	cfg.Progress = func(line string) { fmt.Fprintln(os.Stderr, line) }

	client := anthropic.NewClient()
	results := eval.Run(context.Background(), client, cfg)

	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return fail("%v", err)
	}
	tablePath := filepath.Join(cfg.OutDir, "results.md")
	f, err := os.Create(tablePath)
	if err != nil {
		return fail("%v", err)
	}
	eval.WriteMarkdown(f, results, cfg)
	f.Close()

	if *asJSON {
		b, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fail("%v", err)
		}
		if err := os.WriteFile(filepath.Join(cfg.OutDir, "results.json"), b, 0o644); err != nil {
			return fail("%v", err)
		}
	}

	// Print the table to standard output as well.
	eval.WriteMarkdown(os.Stdout, results, cfg)
	fmt.Fprintf(os.Stderr, "\nWrote %s and the raw outputs under %s/\n", tablePath, cfg.OutDir)

	for _, r := range results {
		if r.Err != "" {
			return exitFound
		}
	}
	return exitOK
}

func listEval() {
	fmt.Println("Tasks:")
	for _, t := range eval.Tasks {
		mode := "flavored"
		if t.Strict {
			mode = "strict"
		}
		fmt.Printf("  %-10s %-26s scored in %s mode\n", t.ID, t.Name, mode)
	}
	fmt.Println("\nConditions:")
	for _, c := range eval.Conditions {
		size := "no system prompt"
		if c.System != "" {
			size = fmt.Sprintf("%d characters", len(c.System))
		}
		fmt.Printf("  %-14s %-22s %s\n", c.ID, c.Name, size)
	}
	fmt.Println("\nModels with a known price:")
	for _, m := range eval.Models {
		fmt.Printf("  %-20s %-18s $%.0f in, $%.0f out for each million tokens\n",
			m.ID, m.Name, m.InPerMTok, m.OutPerMTok)
	}
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
