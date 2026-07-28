package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cstaszak/ste/internal/config"
	"github.com/cstaszak/ste/internal/lint"
	"github.com/cstaszak/ste/internal/report"
)

// proseExtensions are the files a directory walk picks up.
var proseExtensions = map[string]bool{
	".md": true, ".markdown": true, ".txt": true, ".rst": true, ".mdx": true,
}

func runLint(args []string) int {
	fset := flag.NewFlagSet("lint", flag.ExitOnError)
	var (
		mode     = fset.String("mode", "", "flavored or strict (default flavored)")
		format   = fset.String("format", "text", "text, summary, json, github, sarif, or agent")
		maxScore = fset.Float64("max-per100w", -1, "fail when the score is above this limit")
		enable   = fset.String("enable", "", "comma-separated rules to turn on")
		disable  = fset.String("disable", "", "comma-separated rules to turn off")
		only     = fset.String("only", "", "comma-separated rules to run, to the exclusion of all others")
		maxFind  = fset.Int("max-findings", 0, "list at most this many findings per file (0 means all)")
		quiet    = fset.Bool("quiet", false, "print the summary only")
		confPath = fset.String("config", "", "path to .ste.yml (default: search up from the file)")
		noConf   = fset.Bool("no-config", false, "ignore .ste.yml")
		useDict  = fset.Bool("dict", false, "check words against the STE dictionary (needs \"ste dict build\")")
		dictPath = fset.String("dict-index", "", "path to the dictionary index")
	)
	fset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ste lint [flags] [files or directories...]\n\n"+
			"With no paths, ste reads standard input.\n\nFlags:\n")
		fset.PrintDefaults()
	}
	if err := fset.Parse(args); err != nil {
		return exitError
	}

	outFormat, err := report.ParseFormat(*format)
	if err != nil {
		return fail("%v", err)
	}

	paths, err := expand(fset.Args())
	if err != nil {
		return fail("%v", err)
	}

	// Load the configuration relative to the first input, or the working
	// directory when reading standard input.
	confDir := "."
	if len(paths) > 0 {
		confDir = filepath.Dir(paths[0])
	}
	cfgFile := *confPath
	if cfgFile == "" && !*noConf {
		cfgFile = config.Find(confDir)
	}
	if *noConf {
		cfgFile = ""
	}
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fail("%v", err)
	}

	m, err := cfg.ResolveMode(*mode)
	if err != nil {
		return fail("%v", err)
	}
	opt := cfg.Apply(lint.DefaultOptions(m))
	if *useDict {
		ix, _, err := loadIndex(*dictPath)
		if err != nil {
			return fail("%v", err)
		}
		opt.Dict = ix
		opt.Enabled["non-approved-word"] = true
	}
	if err := applyRuleFlags(&opt, *only, *enable, *disable); err != nil {
		return fail("%v", err)
	}
	// A dictionary rule cannot run without an index.
	if opt.Dict == nil {
		opt.Enabled["non-approved-word"] = false
		opt.Enabled["unknown-word"] = false
	}

	threshold, hasThreshold := cfg.Threshold()
	if *maxScore >= 0 {
		threshold, hasThreshold = *maxScore, true
	}

	var reports []*lint.Report
	if len(paths) == 0 {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fail("read standard input: %v", err)
		}
		reports = append(reports, lint.RunText(string(b), opt))
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return fail("%v", err)
		}
		r := lint.RunText(string(b), opt)
		r.Path = p
		reports = append(reports, r)
	}

	err = report.Write(os.Stdout, outFormat, reports, report.Options{
		MaxFindings:  *maxFind,
		Threshold:    threshold,
		HasThreshold: hasThreshold,
		Quiet:        *quiet,
	})
	if err != nil {
		return fail("%v", err)
	}

	if hasThreshold {
		for _, r := range reports {
			if r.Per100W > threshold {
				return exitFound
			}
		}
	}
	return exitOK
}

// applyRuleFlags turns rules on and off from the command line.
func applyRuleFlags(opt *lint.Options, only, enable, disable string) error {
	if only != "" {
		ids, err := ruleIDs(only)
		if err != nil {
			return err
		}
		for id := range opt.Enabled {
			opt.Enabled[id] = false
		}
		for _, id := range ids {
			opt.Enabled[id] = true
		}
	}
	ids, err := ruleIDs(enable)
	if err != nil {
		return err
	}
	for _, id := range ids {
		opt.Enabled[id] = true
	}
	ids, err = ruleIDs(disable)
	if err != nil {
		return err
	}
	for _, id := range ids {
		opt.Enabled[id] = false
	}
	return nil
}

func ruleIDs(csv string) ([]string, error) {
	var out []string
	for _, id := range strings.Split(csv, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := lint.Lookup(id); !ok {
			return nil, fmt.Errorf("unknown rule %q (run \"ste rules\" for the list)", id)
		}
		out = append(out, id)
	}
	return out, nil
}

// expand turns the arguments into a list of files. A directory is walked for
// prose files. A file named on the command line is always read, whatever its
// extension.
func expand(args []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	for _, arg := range args {
		st, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		if !st.IsDir() {
			add(arg)
			continue
		}
		err = filepath.WalkDir(arg, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := d.Name()
			if d.IsDir() {
				if name != "." && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor") {
					return filepath.SkipDir
				}
				return nil
			}
			if proseExtensions[strings.ToLower(filepath.Ext(name))] {
				add(p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}
