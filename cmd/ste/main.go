// Command ste checks prose against the machine-checkable subset of
// ASD-STE100 Simplified Technical English.
package main

import (
	"fmt"
	"os"

	"github.com/cstaszak/ste/internal/report"
)

// version is set at build time with -ldflags.
var version = "dev"

const usage = `ste - ASD-STE100 Simplified Technical English tools

Usage:
  ste <command> [flags] [paths...]

Commands:
  lint     Check prose and report violations
  diff     Compare two drafts and report the score change
  dict     Build and read the ASD-STE100 approved-word index
  eval     Run the cross-model writing experiment
  rules    List the rules
  version  Print the version

Run "ste <command> -h" for the flags of one command.
`

// exit codes
const (
	exitOK    = 0
	exitFound = 1 // violations over the threshold
	exitError = 2 // the tool could not run
)

func main() {
	report.Version = version
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(exitError)
	}
	args := os.Args[2:]
	var code int
	switch os.Args[1] {
	case "lint":
		code = runLint(args)
	case "diff":
		code = runDiff(args)
	case "dict":
		code = runDict(args)
	case "eval":
		code = runEval(args)
	case "rules":
		code = runRules(args)
	case "version", "--version", "-v":
		fmt.Println("ste " + version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "ste: unknown command %q\n\n%s", os.Args[1], usage)
		code = exitError
	}
	os.Exit(code)
}

func fail(format string, a ...any) int {
	fmt.Fprintf(os.Stderr, "ste: "+format+"\n", a...)
	return exitError
}
