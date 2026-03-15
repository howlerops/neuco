package codegen

import (
	"bytes"
	"text/template"

	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/generation"
)

// InstructionData is the input model for INSTRUCTIONS.md rendering.
type InstructionData struct {
	Spec               domain.Spec           `json:"spec"`
	RepoIndex          generation.RepoIndex  `json:"repo_index"`
	Context            ContextBundle         `json:"context"`
	ValidationCommands []string              `json:"validation_commands"`
	MaxIterations      int                   `json:"max_iterations"`
	AgentProvider      string                `json:"agent_provider"`
	AgentModel         string                `json:"agent_model"`
}

const instructionsTemplate = `# Neuco Code Generation Task

## Task
Implement the following specification in this repository.

## Problem Statement
{{.Spec.ProblemStatement}}

## Proposed Solution
{{.Spec.ProposedSolution}}

## User Stories
{{- if .Spec.UserStories}}
{{- range .Spec.UserStories}}
- As a {{.Role}}, I want {{.Want}}, so that {{.SoThat}}
{{- end}}
{{- else}}
- (none provided)
{{- end}}

## Acceptance Criteria
{{- if .Spec.AcceptanceCriteria}}
{{- range .Spec.AcceptanceCriteria}}
- {{.}}
{{- end}}
{{- else}}
- (none provided)
{{- end}}

## UI Changes
{{if .Spec.UIChanges}}{{.Spec.UIChanges}}{{else}}(none provided){{end}}

## Repository Info
- Framework: {{if .RepoIndex.Framework}}{{.RepoIndex.Framework}}{{else}}unknown{{end}}
- Styling: {{if .RepoIndex.Styling}}{{.RepoIndex.Styling}}{{else}}unknown{{end}}
- Test Setup: {{if .RepoIndex.TestSetup}}{{.RepoIndex.TestSetup}}{{else}}unknown{{end}}

## Agent Settings
- Provider: {{if .AgentProvider}}{{.AgentProvider}}{{else}}(not specified){{end}}
- Model: {{if .AgentModel}}{{.AgentModel}}{{else}}(provider default){{end}}
- Max Iterations: {{if gt .MaxIterations 0}}{{.MaxIterations}}{{else}}(not specified){{end}}

## Context Files Provided
The following files have been placed in .neuco/context/ for reference:
{{- if .Context.Manifest}}
{{- range .Context.Manifest}}
- {{.Path}} ({{.Lines}} lines, {{.Relevance}})
{{- end}}
{{- else}}
- (no context files provided)
{{- end}}

## Validation Commands
Before completing, run these commands and ensure they pass:
{{- if .ValidationCommands}}
{{- range .ValidationCommands}}
- {{.}}
{{- end}}
{{- else}}
- (none provided)
{{- end}}

## Rules
1. Read all context files in .neuco/context/ before starting.
2. Follow existing code patterns and conventions.
3. Create tests following the project's test patterns.
4. Only modify files within the scope of this specification.
5. Do not leave TODO or placeholder comments.
6. Run validation commands and fix any failures before finishing.
7. Provide a summary of all changes made.
`

// BuildInstructions renders an INSTRUCTIONS.md document from generation input.
func BuildInstructions(data InstructionData) (string, error) {
	tmpl, err := template.New("instructions").Parse(instructionsTemplate)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", err
	}

	return out.String(), nil
}
