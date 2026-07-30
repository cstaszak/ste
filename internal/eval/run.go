package eval

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/stazelabs/ste/internal/lint"
)

// Model is one model under test, with the price of a million tokens.
type Model struct {
	ID         string
	Name       string
	InPerMTok  float64
	OutPerMTok float64
}

// Models are the models the experiment knows the price of. Prices are in US
// dollars for one million tokens, as of July 2026.
var Models = []Model{
	{ID: "claude-opus-5", Name: "Claude Opus 5", InPerMTok: 5, OutPerMTok: 25},
	{ID: "claude-sonnet-5", Name: "Claude Sonnet 5", InPerMTok: 3, OutPerMTok: 15},
	{ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", InPerMTok: 1, OutPerMTok: 5},
}

// ModelByID returns a model by its ID. An unknown ID still runs, but the cost
// report shows zero.
func ModelByID(id string) Model {
	for _, m := range Models {
		if m.ID == id {
			return m
		}
	}
	return Model{ID: id, Name: id}
}

// Result is one output and its score.
type Result struct {
	Model     string       `json:"model"`
	Task      string       `json:"task"`
	Condition string       `json:"condition"`
	Text      string       `json:"text"`
	Report    *lint.Report `json:"report"`
	InTokens  int64        `json:"input_tokens"`
	OutTokens int64        `json:"output_tokens"`
	Err       string       `json:"error,omitempty"`
}

// Config controls one experiment.
type Config struct {
	Models     []string
	Tasks      []Task
	Conditions []Condition
	// MaxTokens caps each response. It covers thinking and text together.
	MaxTokens int64
	// Concurrency is how many requests run at one time.
	Concurrency int
	// OutDir, when set, receives the raw text of each output.
	OutDir string
	// Progress receives one line for each finished request.
	Progress func(string)
}

// DefaultConfig returns a configuration that runs every task and condition.
func DefaultConfig() Config {
	return Config{
		Models:      []string{"claude-opus-5"},
		Tasks:       Tasks,
		Conditions:  Conditions,
		MaxTokens:   8000,
		Concurrency: 4,
	}
}

// Calls returns how many API requests the configuration makes.
func (c Config) Calls() int {
	return len(c.Models) * len(c.Tasks) * len(c.Conditions)
}

// Estimate returns a rough cost in US dollars. It assumes 700 input tokens and
// 900 output tokens for each call, which matches the tasks in this package.
func (c Config) Estimate() float64 {
	const inTok, outTok = 700.0, 900.0
	perModel := float64(len(c.Tasks) * len(c.Conditions))
	total := 0.0
	for _, id := range c.Models {
		m := ModelByID(id)
		total += perModel * (inTok/1e6*m.InPerMTok + outTok/1e6*m.OutPerMTok)
	}
	return total
}

// Run makes every request and scores every output. It returns the results in a
// stable order, whatever order the requests finish in.
func Run(ctx context.Context, client anthropic.Client, cfg Config) []Result {
	type job struct {
		model string
		task  Task
		cond  Condition
		index int
	}
	var jobs []job
	for _, model := range cfg.Models {
		for _, t := range cfg.Tasks {
			for _, c := range cfg.Conditions {
				jobs = append(jobs, job{model: model, task: t, cond: c, index: len(jobs)})
			}
		}
	}

	results := make([]Result, len(jobs))
	limit := cfg.Concurrency
	if limit < 1 {
		limit = 1
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0

	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			r := one(ctx, client, cfg, j.model, j.task, j.cond)
			results[j.index] = r

			mu.Lock()
			done++
			status := fmt.Sprintf("%.2f per 100w", r.Report.Per100W)
			if r.Err != "" {
				status = "error: " + r.Err
			}
			if cfg.Progress != nil {
				cfg.Progress(fmt.Sprintf("[%d/%d] %s %s/%s: %s",
					done, len(jobs), j.model, j.task.ID, j.cond.ID, status))
			}
			mu.Unlock()
		}(j)
	}
	wg.Wait()
	return results
}

// one makes a single request and scores the output.
func one(ctx context.Context, client anthropic.Client, cfg Config, model string, t Task, c Condition) Result {
	r := Result{Model: model, Task: t.ID, Condition: c.ID, Report: &lint.Report{}}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: cfg.MaxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(t.Prompt)),
		},
	}
	if c.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: c.System}}
	}

	resp, err := client.Messages.New(ctx, params)
	if err != nil {
		r.Err = err.Error()
		return r
	}
	r.InTokens = resp.Usage.InputTokens
	r.OutTokens = resp.Usage.OutputTokens

	var sb strings.Builder
	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(tb.Text)
		}
	}
	r.Text = strings.TrimSpace(sb.String())
	if r.Text == "" {
		r.Err = fmt.Sprintf("no text in the response (stop reason %q)", resp.StopReason)
		return r
	}

	mode := lint.ModeFlavored
	if t.Strict {
		mode = lint.ModeStrict
	}
	r.Report = lint.RunText(r.Text, lint.DefaultOptions(mode))

	if cfg.OutDir != "" {
		if err := writeOutput(cfg.OutDir, r); err != nil {
			r.Err = err.Error()
		}
	}
	return r
}

// writeOutput saves the raw text, so a person can read what the model wrote.
func writeOutput(dir string, r Result) error {
	sub := filepath.Join(dir, safe(r.Model), r.Condition)
	if err := os.MkdirAll(sub, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(sub, r.Task+".md"), []byte(r.Text+"\n"), 0o644)
}

// safe makes a string usable as a directory name.
func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' {
			return '-'
		}
		return r
	}, s)
}

// Summary is the pooled score of one model under one condition.
type Summary struct {
	Model     string
	Condition string
	Per100W   float64
	Words     int
	Total     int
	Errors    int
	Cost      float64
}

// Summarize groups the results by model and condition. The score is pooled:
// total violations over total words, so one long output does not count more
// than its length deserves.
func Summarize(results []Result) []Summary {
	type key struct{ model, cond string }
	acc := map[key]*Summary{}
	var order []key
	for _, r := range results {
		k := key{r.Model, r.Condition}
		s, ok := acc[k]
		if !ok {
			s = &Summary{Model: r.Model, Condition: r.Condition}
			acc[k] = s
			order = append(order, k)
		}
		if r.Err != "" {
			s.Errors++
			continue
		}
		s.Words += r.Report.Words
		s.Total += r.Report.Total
		m := ModelByID(r.Model)
		s.Cost += float64(r.InTokens)/1e6*m.InPerMTok + float64(r.OutTokens)/1e6*m.OutPerMTok
	}

	out := make([]Summary, 0, len(acc))
	for _, k := range order {
		s := acc[k]
		if s.Words > 0 {
			s.Per100W = round2(float64(s.Total) * 100 / float64(s.Words))
		}
		out = append(out, *s)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Model != out[j].Model {
			return out[i].Model < out[j].Model
		}
		return conditionOrder(out[i].Condition) < conditionOrder(out[j].Condition)
	})
	return out
}

func conditionOrder(id string) int {
	for i, c := range Conditions {
		if c.ID == id {
			return i
		}
	}
	return len(Conditions)
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
