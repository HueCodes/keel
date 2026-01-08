package reporter

import (
	"encoding/json"

	"github.com/HueCodes/keel/internal/analyzer"
)

// SARIFReporter outputs results in SARIF format
type SARIFReporter struct {
	cfg *Config
}

// SARIF format structures
type SARIFLog struct {
	Schema  string      `json:"$schema"`
	Version string      `json:"version"`
	Runs    []SARIFRun  `json:"runs"`
}

type SARIFRun struct {
	Tool    SARIFTool    `json:"tool"`
	Results []SARIFResult `json:"results"`
}

type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

type SARIFDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationUri string      `json:"informationUri"`
	Rules          []SARIFRule `json:"rules"`
}

type SARIFRule struct {
	ID               string            `json:"id"`
	Name             string            `json:"name,omitempty"`
	ShortDescription SARIFMessage      `json:"shortDescription,omitempty"`
	DefaultConfig    SARIFRuleConfig   `json:"defaultConfiguration,omitempty"`
}

type SARIFRuleConfig struct {
	Level string `json:"level"`
}

type SARIFMessage struct {
	Text string `json:"text"`
}

type SARIFResult struct {
	RuleID    string           `json:"ruleId"`
	Level     string           `json:"level"`
	Message   SARIFMessage     `json:"message"`
	Locations []SARIFLocation  `json:"locations"`
}

type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
	Region           SARIFRegion           `json:"region"`
}

type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

type SARIFRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine,omitempty"`
	EndColumn   int `json:"endColumn,omitempty"`
}

func severityToSARIFLevel(s analyzer.Severity) string {
	switch s {
	case analyzer.SeverityError:
		return "error"
	case analyzer.SeverityWarning:
		return "warning"
	case analyzer.SeverityInfo:
		return "note"
	default:
		return "none"
	}
}

// Report outputs the analysis results in SARIF format
func (r *SARIFReporter) Report(result *analyzer.Result, source string) error {
	log := SARIFLog{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []SARIFRun{{
			Tool: SARIFTool{
				Driver: SARIFDriver{
					Name:           "keel",
					Version:        "0.1.0",
					InformationUri: "https://github.com/HueCodes/keel",
					Rules:          []SARIFRule{},
				},
			},
			Results: []SARIFResult{},
		}},
	}

	// Track rules we've seen
	rulesSeen := make(map[string]bool)

	for _, diag := range result.Diagnostics {
		// Add rule if not seen
		if !rulesSeen[diag.Rule] {
			rulesSeen[diag.Rule] = true
			log.Runs[0].Tool.Driver.Rules = append(log.Runs[0].Tool.Driver.Rules, SARIFRule{
				ID:               diag.Rule,
				ShortDescription: SARIFMessage{Text: diag.Message},
				DefaultConfig:    SARIFRuleConfig{Level: severityToSARIFLevel(diag.Severity)},
			})
		}

		// Add result
		log.Runs[0].Results = append(log.Runs[0].Results, SARIFResult{
			RuleID:  diag.Rule,
			Level:   severityToSARIFLevel(diag.Severity),
			Message: SARIFMessage{Text: diag.Message},
			Locations: []SARIFLocation{{
				PhysicalLocation: SARIFPhysicalLocation{
					ArtifactLocation: SARIFArtifactLocation{URI: result.Filename},
					Region: SARIFRegion{
						StartLine:   diag.Pos.Line,
						StartColumn: diag.Pos.Column,
						EndLine:     diag.EndPos.Line,
						EndColumn:   diag.EndPos.Column,
					},
				},
			}},
		})
	}

	encoder := json.NewEncoder(r.cfg.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(log)
}
