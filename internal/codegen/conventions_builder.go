package codegen

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/neuco-ai/neuco/internal/generation"
)

// BuildConventions renders a practical convention guide using index signals and
// lightweight file-content sampling.
func BuildConventions(index generation.RepoIndex, sampleFiles map[string]string) (string, error) {
	data := conventionsTemplateData{
		Framework:         fallback(index.Framework, "unknown framework"),
		Styling:           fallback(index.Styling, "project-default styling"),
		ComponentPattern:  detectComponentPattern(index.Components),
		TestPattern:       detectTestPattern(index.TestSetup, sampleFiles),
		ComponentStructure: detectComponentStructure(index.Components, sampleFiles),
		TestPatterns:      detectTestPatterns(index.TestSetup, sampleFiles),
		ImportStyle:       detectImportStyle(sampleFiles),
		StateManagement:   detectStateManagement(sampleFiles),
		DoList:            detectDoPatterns(index, sampleFiles),
		DontList: []string{
			"Modify files outside the scope",
			"Add new dependencies without justification",
			"Change existing test assertions",
		},
	}

	tmpl, err := template.New("conventions").Parse(conventionsTemplate)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", err
	}

	return out.String(), nil
}

type conventionsTemplateData struct {
	Framework          string
	Styling            string
	ComponentPattern   string
	TestPattern        string
	ComponentStructure string
	TestPatterns       string
	ImportStyle        string
	StateManagement    string
	DoList             []string
	DontList           []string
}

const conventionsTemplate = `# Project Conventions

## Framework
{{.Framework}} project using {{.Styling}} for styling.

## File Naming
- Components: {{.ComponentPattern}}
- Tests: {{.TestPattern}}

## Code Patterns
### Component Structure
{{.ComponentStructure}}

### Test Patterns
{{.TestPatterns}}

### Import Style
{{.ImportStyle}}

### State Management
{{.StateManagement}}

## Do
{{- range .DoList}}
- {{.}}
{{- end}}

## Don't
{{- range .DontList}}
- {{.}}
{{- end}}
`

func detectComponentPattern(components []generation.ComponentInfo) string {
	if len(components) == 0 {
		return "No component index available; follow nearby file naming"
	}

	tsxCount := 0
	componentSuffix := 0
	indexFiles := 0
	for _, c := range components {
		lower := strings.ToLower(c.Path)
		if strings.HasSuffix(lower, ".tsx") || strings.HasSuffix(lower, ".jsx") || strings.HasSuffix(lower, ".vue") || strings.HasSuffix(lower, ".svelte") {
			tsxCount++
		}
		if strings.Contains(lower, ".component.") {
			componentSuffix++
		}
		if strings.HasSuffix(lower, "/index.tsx") || strings.HasSuffix(lower, "/index.jsx") || strings.HasSuffix(lower, "/index.ts") {
			indexFiles++
		}
	}

	switch {
	case componentSuffix > 0:
		return "Use <feature>.component.ts style for framework components"
	case indexFiles > len(components)/3:
		return "Directory-per-component with index.* entry files"
	case tsxCount > 0:
		return "PascalCase component files (e.g. UserCard.tsx)"
	default:
		return "Follow existing component file names from indexed components"
	}
}

func detectTestPattern(testSetup string, sampleFiles map[string]string) string {
	goTest := 0
	jsTest := 0
	specTest := 0
	for path := range sampleFiles {
		lower := strings.ToLower(path)
		switch {
		case strings.HasSuffix(lower, "_test.go"):
			goTest++
		case strings.Contains(lower, ".test."):
			jsTest++
		case strings.Contains(lower, ".spec."):
			specTest++
		}
	}

	switch {
	case goTest > 0:
		return "Use *_test.go files colocated with production packages"
	case jsTest >= specTest && jsTest > 0:
		return "Use *.test.* files near source files"
	case specTest > 0:
		return "Use *.spec.* files near source files"
	case testSetup != "":
		return fmt.Sprintf("Follow %s conventions from existing test directories", testSetup)
	default:
		return "Follow existing test file naming found near modified code"
	}
}

func detectComponentStructure(components []generation.ComponentInfo, _ map[string]string) string {
	if len(components) == 0 {
		return "No indexed component files; mirror structure of neighboring files in target module."
	}

	withProps := 0
	withImports := 0
	for _, c := range components {
		if len(c.Props) > 0 {
			withProps++
		}
		if len(c.Imports) > 0 {
			withImports++
		}
	}

	parts := []string{}
	if withProps > 0 {
		parts = append(parts, "Components commonly define explicit Props/ComponentProps types")
	}
	if withImports > 0 {
		parts = append(parts, "Components rely on explicit imports instead of implicit globals")
	}
	if len(parts) == 0 {
		parts = append(parts, "Keep components focused and follow existing folder-level composition")
	}

	return strings.Join(parts, "; ") + "."
}

func detectTestPatterns(testSetup string, sampleFiles map[string]string) string {
	var cues []string
	if testSetup != "" {
		cues = append(cues, fmt.Sprintf("Primary runner appears to be %s", testSetup))
	}

	assertions := 0
	for _, content := range sampleFiles {
		lower := strings.ToLower(content)
		if strings.Contains(lower, "expect(") || strings.Contains(lower, "require.") || strings.Contains(lower, "assert.") {
			assertions++
		}
	}
	if assertions > 0 {
		cues = append(cues, "Tests use explicit assertions (expect/assert/require) and should preserve assertion intent")
	}
	if len(cues) == 0 {
		cues = append(cues, "No clear test samples detected; preserve patterns from closest existing tests")
	}
	return strings.Join(cues, "; ") + "."
}

func detectImportStyle(sampleFiles map[string]string) string {
	relative := 0
	absolute := 0
	barrel := 0

	for _, content := range sampleFiles {
		for _, line := range strings.Split(content, "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "import ") {
				continue
			}
			if strings.Contains(trimmed, " from './") || strings.Contains(trimmed, " from '../") {
				relative++
			} else {
				absolute++
			}
			if strings.Contains(trimmed, " from './index'") || strings.Contains(trimmed, " from '../index'") {
				barrel++
			}
		}
	}

	switch {
	case relative == 0 && absolute == 0:
		return "No import samples detected; follow local module conventions"
	case relative >= absolute:
		if barrel > 0 {
			return "Primarily relative imports with occasional barrel (index) exports"
		}
		return "Primarily relative imports between nearby modules"
	default:
		if barrel > 0 {
			return "Leans toward absolute/aliased imports with some barrel exports"
		}
		return "Leans toward absolute/aliased imports for cross-module references"
	}
}

func detectStateManagement(sampleFiles map[string]string) string {
	joined := strings.ToLower(joinSampleContents(sampleFiles))
	patterns := []string{}

	if strings.Contains(joined, "createcontext(") || strings.Contains(joined, "usecontext(") {
		patterns = append(patterns, "React Context")
	}
	if strings.Contains(joined, "usestate(") || strings.Contains(joined, "usereducer(") {
		patterns = append(patterns, "React hook-local state")
	}
	if strings.Contains(joined, "zustand") || strings.Contains(joined, "create(") && strings.Contains(joined, "store") {
		patterns = append(patterns, "store-based state (possibly Zustand/custom stores)")
	}
	if strings.Contains(joined, "redux") || strings.Contains(joined, "configurestore") {
		patterns = append(patterns, "Redux")
	}
	if strings.Contains(joined, "pinia") || strings.Contains(joined, "vuex") {
		patterns = append(patterns, "Vue store pattern")
	}

	if len(patterns) == 0 {
		return "No explicit global state pattern detected; keep state management consistent with nearby modules"
	}

	return "Detected patterns: " + strings.Join(patterns, ", ")
}

func detectDoPatterns(index generation.RepoIndex, sampleFiles map[string]string) []string {
	do := []string{
		"Match existing naming and folder layout in the touched area",
		"Keep changes minimal and scoped to the requested behavior",
	}

	if index.TestSetup != "" {
		do = append(do, fmt.Sprintf("Run and fix %s tests when modifying behavior", index.TestSetup))
	}
	if len(index.Components) > 0 {
		do = append(do, "Reuse existing component props and composition patterns")
	}

	if hasConfigFile(sampleFiles, "go.mod") {
		do = append(do, "Preserve Go package boundaries and existing error-handling style")
	}
	if hasTSFiles(sampleFiles) {
		do = append(do, "Respect existing TypeScript types and avoid weakening type safety")
	}

	return do
}

func hasConfigFile(sampleFiles map[string]string, name string) bool {
	for path := range sampleFiles {
		if strings.EqualFold(filepath.Base(path), name) {
			return true
		}
	}
	return false
}

func hasTSFiles(sampleFiles map[string]string) bool {
	for path := range sampleFiles {
		lower := strings.ToLower(path)
		if strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx") {
			return true
		}
	}
	return false
}

func joinSampleContents(sampleFiles map[string]string) string {
	var b strings.Builder
	for _, content := range sampleFiles {
		b.WriteString(content)
		b.WriteString("\n")
	}
	return b.String()
}

func fallback(value string, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}
