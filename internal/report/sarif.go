package report

import (
	"encoding/json"
	"io"

	"github.com/cstaszak/ste/internal/lint"
)

// Version is set by the build and named in the SARIF output.
var Version = "dev"

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Version        string      `json:"version"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	ShortDescription sarifText       `json:"shortDescription"`
	Properties       sarifRuleProps  `json:"properties"`
	DefaultConfig    sarifRuleConfig `json:"defaultConfiguration"`
}

type sarifRuleProps struct {
	Tags []string `json:"tags"`
}

type sarifRuleConfig struct {
	Level string `json:"level"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifText       `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           sarifRegion   `json:"region"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}

func writeSARIF(w io.Writer, reports []*lint.Report) error {
	run := sarifRun{
		Tool: sarifTool{Driver: sarifDriver{
			Name:           "ste",
			InformationURI: "https://github.com/cstaszak/ste",
			Version:        Version,
		}},
		Results: []sarifResult{},
	}
	for _, r := range lint.Rules() {
		run.Tool.Driver.Rules = append(run.Tool.Driver.Rules, sarifRule{
			ID:               r.ID,
			Name:             r.ID,
			ShortDescription: sarifText{Text: r.Doc},
			Properties:       sarifRuleProps{Tags: []string{"ste", string(r.Category)}},
			DefaultConfig:    sarifRuleConfig{Level: "warning"},
		})
	}
	for _, rep := range reports {
		for _, f := range rep.Findings {
			msg := f.Message
			if f.Suggest != "" {
				msg += " -> " + f.Suggest
			}
			run.Results = append(run.Results, sarifResult{
				RuleID:  f.Rule,
				Level:   "warning",
				Message: sarifText{Text: msg},
				Locations: []sarifLocation{{PhysicalLocation: sarifPhysical{
					ArtifactLocation: sarifArtifact{URI: rep.Path},
					Region:           sarifRegion{StartLine: f.Position.Line, StartColumn: f.Position.Col},
				}}},
			})
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs:    []sarifRun{run},
	})
}
