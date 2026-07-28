// Package report writes lint results in the formats the CLI supports.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/cstaszak/ste/internal/lint"
)

// Format names one output style.
type Format string

const (
	// Text is the default human-readable output.
	Text Format = "text"
	// Summary is one line per file.
	Summary Format = "summary"
	// JSON is the machine-readable form.
	JSON Format = "json"
	// GitHub emits workflow annotations.
	GitHub Format = "github"
	// SARIF is the static-analysis interchange format code scanners read.
	SARIF Format = "sarif"
	// Agent is a compact form for a coding agent to act on.
	Agent Format = "agent"
)

// ParseFormat converts a format name to a Format.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case Text, Summary, JSON, GitHub, SARIF, Agent:
		return Format(s), nil
	default:
		return "", fmt.Errorf("unknown format %q (use text, summary, json, github, sarif, or agent)", s)
	}
}

// Options controls what the writers print.
type Options struct {
	// MaxFindings caps how many findings the text and agent formats list.
	// Zero means no cap.
	MaxFindings int
	// Threshold is the score gate, shown in the text summary.
	Threshold float64
	// HasThreshold reports whether Threshold is set.
	HasThreshold bool
	// Quiet drops the per-finding detail and prints only the summary.
	Quiet bool
}

// Write prints every report in the chosen format.
func Write(w io.Writer, format Format, reports []*lint.Report, opt Options) error {
	switch format {
	case Text:
		return writeText(w, reports, opt)
	case Summary:
		return writeSummary(w, reports)
	case JSON:
		return writeJSON(w, reports)
	case GitHub:
		return writeGitHub(w, reports)
	case SARIF:
		return writeSARIF(w, reports)
	case Agent:
		return writeAgent(w, reports, opt)
	}
	return fmt.Errorf("unknown format %q", format)
}

func writeText(w io.Writer, reports []*lint.Report, opt Options) error {
	for i, r := range reports {
		if i > 0 {
			fmt.Fprintln(w)
		}
		name := r.Path
		if name == "" {
			name = "(stdin)"
		}
		fmt.Fprintf(w, "%s\n", name)
		if !opt.Quiet {
			shown := r.Findings
			if opt.MaxFindings > 0 && len(shown) > opt.MaxFindings {
				shown = shown[:opt.MaxFindings]
			}
			for _, f := range shown {
				line := fmt.Sprintf("  %d:%d  %-20s %s", f.Position.Line, f.Position.Col, f.Rule, f.Message)
				if f.Suggest != "" {
					line += "  -> " + f.Suggest
				}
				fmt.Fprintln(w, line)
			}
			if n := len(r.Findings) - len(shown); n > 0 {
				fmt.Fprintf(w, "  ... and %d more\n", n)
			}
		}
		fmt.Fprintf(w, "  %s, %s, %.2f per 100 words",
			count(r.Words, "word"), count(r.Total, "violation"), r.Per100W)
		if opt.HasThreshold {
			fmt.Fprintf(w, " (limit %.2f)", opt.Threshold)
		}
		fmt.Fprintln(w)
		if len(r.Counts) > 0 && !opt.Quiet {
			fmt.Fprintf(w, "  by rule: %s\n", ruleBreakdown(r))
		}
		if r.EmDashes > 0 {
			fmt.Fprintf(w, "  em dashes: %d\n", r.EmDashes)
		}
	}
	if len(reports) > 1 {
		words, total := 0, 0
		for _, r := range reports {
			words += r.Words
			total += r.Total
		}
		per := 0.0
		if words > 0 {
			per = float64(total) * 100 / float64(words)
		}
		fmt.Fprintf(w, "\ntotal: %s, %s, %s, %.2f per 100 words\n",
			count(len(reports), "file"), count(words, "word"),
			count(total, "violation"), per)
	}
	return nil
}

func writeSummary(w io.Writer, reports []*lint.Report) error {
	for _, r := range reports {
		name := r.Path
		if name == "" {
			name = "(stdin)"
		}
		fmt.Fprintf(w, "%-40s words=%5d total=%4d per100w=%6.2f em_dash=%3d\n",
			name, r.Words, r.Total, r.Per100W, r.EmDashes)
	}
	return nil
}

func writeJSON(w io.Writer, reports []*lint.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if len(reports) == 1 {
		return enc.Encode(reports[0])
	}
	return enc.Encode(reports)
}

func writeGitHub(w io.Writer, reports []*lint.Report) error {
	for _, r := range reports {
		for _, f := range r.Findings {
			msg := f.Message
			if f.Suggest != "" {
				msg += " -> " + f.Suggest
			}
			fmt.Fprintf(w, "::warning file=%s,line=%d,col=%d,title=ste/%s::%s\n",
				r.Path, f.Position.Line, f.Position.Col, f.Rule, msg)
		}
	}
	return nil
}

// writeAgent prints a compact list a coding agent can act on directly.
func writeAgent(w io.Writer, reports []*lint.Report, opt Options) error {
	total := 0
	for _, r := range reports {
		total += r.Total
	}
	if total == 0 {
		fmt.Fprintln(w, "ste: clean.")
		return nil
	}
	for _, r := range reports {
		if r.Total == 0 {
			continue
		}
		fmt.Fprintf(w, "%s: %s, %.2f per 100 words\n",
			r.Path, count(r.Total, "violation"), r.Per100W)
		shown := r.Findings
		if opt.MaxFindings > 0 && len(shown) > opt.MaxFindings {
			shown = shown[:opt.MaxFindings]
		}
		for _, f := range shown {
			line := fmt.Sprintf("  L%d %s", f.Position.Line, f.Message)
			if f.Suggest != "" {
				line += " -> " + f.Suggest
			}
			fmt.Fprintln(w, line)
		}
		if n := len(r.Findings) - len(shown); n > 0 {
			fmt.Fprintf(w, "  ... and %d more\n", n)
		}
	}
	return nil
}

// count writes a number and its noun, with the plural "s" when the number is
// not one.
func count(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func ruleBreakdown(r *lint.Report) string {
	ids := make([]string, 0, len(r.Counts))
	for id := range r.Counts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if r.Counts[ids[i]] != r.Counts[ids[j]] {
			return r.Counts[ids[i]] > r.Counts[ids[j]]
		}
		return ids[i] < ids[j]
	})
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%s=%d", id, r.Counts[id]))
	}
	return strings.Join(parts, " ")
}
