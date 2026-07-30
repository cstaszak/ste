package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/stazelabs/ste/internal/dict"
)

const dictUsage = `Usage: ste dict <command> [flags]

Commands:
  build --pdf <file>   Build the approved-word index from your copy of ASD-STE100
  stats                Show what the index holds
  lookup <word>...     Show what the standard says about a word
  path                 Print the index location

The standard is under copyright. The index is built on your machine from your
own copy, and it is never committed. Get the standard at https://asd-ste100.org.
`

func runDict(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, dictUsage)
		return exitError
	}
	switch args[0] {
	case "build":
		return runDictBuild(args[1:])
	case "stats":
		return runDictStats(args[1:])
	case "lookup":
		return runDictLookup(args[1:])
	case "path":
		p, err := dict.DefaultPath()
		if err != nil {
			return fail("%v", err)
		}
		fmt.Println(p)
		return exitOK
	default:
		fmt.Fprintf(os.Stderr, "ste dict: unknown command %q\n\n%s", args[0], dictUsage)
		return exitError
	}
}

func runDictBuild(args []string) int {
	fset := flag.NewFlagSet("dict build", flag.ExitOnError)
	pdfPath := fset.String("pdf", "", "path to the ASD-STE100 specification, as a PDF")
	out := fset.String("out", "", "where to write the index (default: the cache directory)")
	if err := fset.Parse(args); err != nil {
		return exitError
	}
	if *pdfPath == "" {
		fmt.Fprint(os.Stderr, "ste dict build: give the specification with --pdf\n")
		return exitError
	}

	fmt.Fprintf(os.Stderr, "Reading %s ...\n", *pdfPath)
	ix, err := dict.Build(*pdfPath)
	if err != nil {
		return fail("%v", err)
	}

	path := *out
	if path == "" {
		if path, err = dict.DefaultPath(); err != nil {
			return fail("%v", err)
		}
	}
	if err := ix.Save(path); err != nil {
		return fail("%v", err)
	}
	printStats(ix.Stats(), path)
	return exitOK
}

func runDictStats(args []string) int {
	fset := flag.NewFlagSet("dict stats", flag.ExitOnError)
	in := fset.String("index", "", "path to the index")
	if err := fset.Parse(args); err != nil {
		return exitError
	}
	ix, path, err := loadIndex(*in)
	if err != nil {
		return fail("%v", err)
	}
	printStats(ix.Stats(), path)
	return exitOK
}

func runDictLookup(args []string) int {
	fset := flag.NewFlagSet("dict lookup", flag.ExitOnError)
	in := fset.String("index", "", "path to the index")
	if err := fset.Parse(args); err != nil {
		return exitError
	}
	if fset.NArg() == 0 {
		fmt.Fprint(os.Stderr, "ste dict lookup: give one word or more\n")
		return exitError
	}
	ix, _, err := loadIndex(*in)
	if err != nil {
		return fail("%v", err)
	}
	for _, w := range fset.Args() {
		key := strings.ToLower(w)
		switch {
		case ix.Approved[key].Word != "":
			e := ix.Approved[key]
			fmt.Printf("%-18s approved  %s\n", w, strings.Join(e.POS, ", "))
		case ix.IsApproved(key):
			fmt.Printf("%-18s approved  (a regular form of an approved word)\n", w)
		default:
			alts, pos, listed := ix.Rejected(key)
			if !listed {
				fmt.Printf("%-18s not in the dictionary\n", w)
				continue
			}
			line := fmt.Sprintf("%-18s NOT approved", w)
			if len(pos) > 0 {
				line += "  (" + strings.Join(pos, ", ") + ")"
			}
			if len(alts) > 0 {
				line += "  use: " + strings.Join(alts, ", ")
			}
			fmt.Println(line)
		}
	}
	return exitOK
}

// loadIndex reads the index from the given path, or from the cache.
func loadIndex(path string) (*dict.Index, string, error) {
	var err error
	if path == "" {
		if path, err = dict.DefaultPath(); err != nil {
			return nil, "", err
		}
	}
	ix, err := dict.Load(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, path, fmt.Errorf("no index at %s; run \"ste dict build --pdf <file>\" first", path)
		}
		return nil, path, err
	}
	return ix, path, nil
}

func printStats(s dict.Stats, path string) {
	fmt.Printf("index      %s\n", path)
	if s.Source != "" {
		fmt.Printf("source     %s (%d pages)\n", s.Source, s.Pages)
	}
	fmt.Printf("approved   %d words\n", s.Approved)
	fmt.Printf("rejected   %d words, %d with a suggested replacement\n", s.Unapproved, s.WithAlts)
	if s.Approved < 500 || s.Approved > 2000 {
		fmt.Fprintf(os.Stderr,
			"\nWarning: the approved count is outside the range this parser expects (500 to 2000).\n"+
				"The PDF may not be ASD-STE100 Issue 9, or its layout may differ.\n")
	}
}
