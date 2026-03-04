package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

// RepoIndex is the full codebase index produced by IndexRepo. It captures
// components, stories, type files, design tokens, and detected tooling.
type RepoIndex struct {
	Components   []ComponentInfo   `json:"components"`
	Stories      []StoryInfo       `json:"stories"`
	TypeFiles    []TypeFileInfo    `json:"type_files"`
	DesignTokens []DesignTokenInfo `json:"design_tokens"`
	Framework    string            `json:"framework"`
	Styling      string            `json:"styling"`
	TestSetup    string            `json:"test_setup"`
}

// ComponentInfo describes a single UI component file.
type ComponentInfo struct {
	Path     string   `json:"path"`
	Name     string   `json:"name"`
	Imports  []string `json:"imports"`
	Props    []string `json:"props"`
	FileSize int      `json:"file_size"`
}

// StoryInfo describes a Storybook story file.
type StoryInfo struct {
	Path          string `json:"path"`
	ComponentName string `json:"component_name"`
	FileSize      int    `json:"file_size"`
}

// TypeFileInfo describes a TypeScript type definition file.
type TypeFileInfo struct {
	Path     string `json:"path"`
	FileSize int    `json:"file_size"`
}

// DesignTokenInfo describes a design token or theming file.
type DesignTokenInfo struct {
	Path     string `json:"path"`
	FileSize int    `json:"file_size"`
}

// Indexer walks a GitHub repository and builds a RepoIndex from its file tree.
type Indexer struct {
	gh *GitHubService
}

// NewIndexer constructs an Indexer backed by the given GitHubService.
func NewIndexer(ghService *GitHubService) *Indexer {
	return &Indexer{gh: ghService}
}

// IndexRepo walks the repository tree at ref (or the default branch when ref
// is empty), classifies files by their role, fetches content for identified
// files, extracts metadata, and returns a populated RepoIndex.
func (idx *Indexer) IndexRepo(
	ctx context.Context,
	installationID int64,
	owner, repo, ref string,
) (*RepoIndex, error) {
	client, err := idx.gh.GetInstallationClient(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("IndexRepo: get client: %w", err)
	}

	// Fetch the full recursive tree in one API call to avoid per-directory
	// round trips (the recursive tree endpoint returns all paths at once).
	if ref == "" {
		// Get the default branch SHA.
		repoInfo, _, err := client.Repositories.Get(ctx, owner, repo)
		if err != nil {
			return nil, fmt.Errorf("IndexRepo: get repo info: %w", err)
		}
		ref = repoInfo.GetDefaultBranch()
	}

	tree, _, err := client.Git.GetTree(ctx, owner, repo, ref, true)
	if err != nil {
		return nil, fmt.Errorf("IndexRepo: get tree: %w", err)
	}

	index := &RepoIndex{}

	// Classify every entry.
	var componentPaths []string
	var storyPaths []string
	var typePaths []string
	var tokenPaths []string
	var packageJSONPath string

	for _, entry := range tree.Entries {
		if entry.GetType() != "blob" {
			continue
		}
		path := entry.GetPath()

		switch {
		case path == "package.json":
			packageJSONPath = path

		case isComponentFile(path):
			componentPaths = append(componentPaths, path)

		case isStoryFile(path):
			storyPaths = append(storyPaths, path)

		case isTypeFile(path):
			typePaths = append(typePaths, path)

		case isDesignTokenFile(path):
			tokenPaths = append(tokenPaths, path)
		}
	}

	// Detect framework and styling from package.json.
	if packageJSONPath != "" {
		content, err := idx.gh.GetFileContent(ctx, client, owner, repo, packageJSONPath, ref)
		if err == nil {
			index.Framework, index.Styling, index.TestSetup = detectToolingFromPackageJSON(content)
		} else {
			slog.Warn("IndexRepo: could not fetch package.json", "error", err)
		}
	}

	// Fetch and parse component files (cap at 40 to stay within rate limits).
	maxComponents := 40
	if len(componentPaths) < maxComponents {
		maxComponents = len(componentPaths)
	}
	for _, path := range componentPaths[:maxComponents] {
		content, err := idx.gh.GetFileContent(ctx, client, owner, repo, path, ref)
		if err != nil {
			slog.Warn("IndexRepo: skipping component", "path", path, "error", err)
			continue
		}
		info := ComponentInfo{
			Path:     path,
			Name:     componentNameFromPath(path),
			Imports:  extractImports(content),
			Props:    extractProps(content),
			FileSize: len(content),
		}
		index.Components = append(index.Components, info)
	}

	// Fetch and index story files (cap at 20).
	maxStories := 20
	if len(storyPaths) < maxStories {
		maxStories = len(storyPaths)
	}
	for _, path := range storyPaths[:maxStories] {
		content, err := idx.gh.GetFileContent(ctx, client, owner, repo, path, ref)
		if err != nil {
			slog.Warn("IndexRepo: skipping story", "path", path, "error", err)
			continue
		}
		index.Stories = append(index.Stories, StoryInfo{
			Path:          path,
			ComponentName: storyComponentName(path),
			FileSize:      len(content),
		})
	}

	// Type files (cap at 20).
	maxTypes := 20
	if len(typePaths) < maxTypes {
		maxTypes = len(typePaths)
	}
	for _, path := range typePaths[:maxTypes] {
		content, err := idx.gh.GetFileContent(ctx, client, owner, repo, path, ref)
		if err != nil {
			slog.Warn("IndexRepo: skipping type file", "path", path, "error", err)
			continue
		}
		index.TypeFiles = append(index.TypeFiles, TypeFileInfo{
			Path:     path,
			FileSize: len(content),
		})
	}

	// Design token files.
	for _, path := range tokenPaths {
		content, err := idx.gh.GetFileContent(ctx, client, owner, repo, path, ref)
		if err != nil {
			slog.Warn("IndexRepo: skipping token file", "path", path, "error", err)
			continue
		}
		index.DesignTokens = append(index.DesignTokens, DesignTokenInfo{
			Path:     path,
			FileSize: len(content),
		})
	}

	slog.Info("IndexRepo complete",
		"owner", owner,
		"repo", repo,
		"components", len(index.Components),
		"stories", len(index.Stories),
		"type_files", len(index.TypeFiles),
		"design_tokens", len(index.DesignTokens),
		"framework", index.Framework,
		"styling", index.Styling,
	)
	return index, nil
}

// ---------------------------------------------------------------------------
// Classification helpers
// ---------------------------------------------------------------------------

// isComponentFile returns true for React/Vue/Svelte/Angular component files
// that are likely to contain reusable UI components.
func isComponentFile(path string) bool {
	if isStoryFile(path) {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".tsx", ".jsx":
		// Accept files under common component directories.
		lower := strings.ToLower(path)
		return inComponentDir(lower)
	case ".vue":
		return true
	case ".svelte":
		return true
	case ".ts":
		// Angular components end with .component.ts
		return strings.HasSuffix(strings.ToLower(path), ".component.ts")
	}
	return false
}

// isStoryFile returns true for Storybook story files.
func isStoryFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, ".stories.tsx") ||
		strings.Contains(lower, ".stories.ts") ||
		strings.Contains(lower, ".stories.jsx") ||
		strings.Contains(lower, ".stories.js")
}

// isTypeFile returns true for TypeScript type definition files.
func isTypeFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return base == "types.ts" ||
		base == "interfaces.ts" ||
		strings.HasSuffix(base, ".d.ts") ||
		base == "types.d.ts"
}

// isDesignTokenFile returns true for design token or theme configuration files.
func isDesignTokenFile(path string) bool {
	lower := strings.ToLower(path)
	base := strings.ToLower(filepath.Base(path))

	// tailwind.config.* at any depth
	if strings.HasPrefix(base, "tailwind.config.") {
		return true
	}
	// theme.* files
	if strings.HasPrefix(base, "theme.") {
		return true
	}
	// tokens.* files
	if strings.HasPrefix(base, "tokens.") {
		return true
	}
	// CSS files in root or styles/ directory
	if filepath.Ext(base) == ".css" {
		dir := filepath.Dir(lower)
		if dir == "." || dir == "styles" || strings.HasSuffix(dir, "/styles") {
			return true
		}
	}
	return false
}

// inComponentDir returns true if the lowercased path sits inside a typical
// component directory.
func inComponentDir(lowerPath string) bool {
	prefixes := []string{
		"src/components/",
		"components/",
		"app/components/",
		"src/app/",
		"app/",
		"src/ui/",
		"ui/",
		"src/features/",
		"features/",
		"src/pages/",
		"pages/",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lowerPath, p) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Metadata extraction (heuristic, regex-free, line-scanning approach)
// ---------------------------------------------------------------------------

// extractImports returns the list of module specifiers imported by a JS/TS file.
func extractImports(content string) []string {
	var imports []string
	seen := map[string]struct{}{}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "import ") {
			continue
		}
		// Extract the quoted module specifier.
		for _, sep := range []string{"from '", `from "`} {
			idx := strings.Index(trimmed, sep)
			if idx == -1 {
				continue
			}
			rest := trimmed[idx+len(sep):]
			end := strings.IndexAny(rest, `'"`)
			if end == -1 {
				continue
			}
			mod := rest[:end]
			if _, ok := seen[mod]; !ok {
				seen[mod] = struct{}{}
				imports = append(imports, mod)
			}
		}
	}
	return imports
}

// extractProps returns prop names found in TypeScript interface or type Props
// declarations, covering both interface Props { … } and
// type Props = { … } patterns.
func extractProps(content string) []string {
	var props []string
	inPropsBlock := false
	braceDepth := 0

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inPropsBlock {
			// Detect the start of a Props type/interface.
			lower := strings.ToLower(trimmed)
			if strings.Contains(lower, "interface props") ||
				strings.Contains(lower, "type props") ||
				strings.Contains(lower, "interface componentprops") ||
				strings.Contains(lower, "type componentprops") {
				inPropsBlock = true
				braceDepth = strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
				continue
			}
		} else {
			braceDepth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			if braceDepth <= 0 {
				inPropsBlock = false
				continue
			}
			// Extract prop name: everything before "?", ":", or whitespace.
			if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue
			}
			name := trimmed
			for _, sep := range []string{"?", ":", " ", "\t"} {
				if idx := strings.Index(name, sep); idx != -1 {
					name = name[:idx]
				}
			}
			name = strings.TrimSpace(name)
			if name != "" && name != "{" && name != "}" {
				props = append(props, name)
			}
		}
	}
	return props
}

// componentNameFromPath derives a PascalCase component name from its file path.
// e.g. "src/components/ui/Button.tsx" -> "Button"
func componentNameFromPath(path string) string {
	base := filepath.Base(path)
	// Strip extension(s): "Button.stories.tsx" -> "Button.stories" -> "Button"
	name := base
	for ext := filepath.Ext(name); ext != ""; ext = filepath.Ext(name) {
		name = strings.TrimSuffix(name, ext)
	}
	// Handle "index" files: use the parent directory name.
	if strings.ToLower(name) == "index" {
		name = filepath.Base(filepath.Dir(path))
	}
	return name
}

// storyComponentName derives the component name from a story file path.
// e.g. "src/stories/Button.stories.tsx" -> "Button"
func storyComponentName(path string) string {
	base := filepath.Base(path)
	// Strip ".stories.*" suffix.
	for _, sep := range []string{".stories.tsx", ".stories.ts", ".stories.jsx", ".stories.js"} {
		if strings.HasSuffix(strings.ToLower(base), sep) {
			return base[:len(base)-len(sep)]
		}
	}
	return componentNameFromPath(path)
}

// ---------------------------------------------------------------------------
// Tooling detection
// ---------------------------------------------------------------------------

// detectToolingFromPackageJSON parses a package.json blob to identify the
// primary frontend framework, CSS approach, and test runner in use.
func detectToolingFromPackageJSON(content string) (framework, styling, testSetup string) {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return "", "", ""
	}

	all := make(map[string]struct{})
	for k := range pkg.Dependencies {
		all[k] = struct{}{}
	}
	for k := range pkg.DevDependencies {
		all[k] = struct{}{}
	}

	has := func(pkg string) bool {
		_, ok := all[pkg]
		return ok
	}

	// Framework detection (ordered from most specific to least).
	switch {
	case has("next"):
		framework = "nextjs"
	case has("@nuxt/core") || has("nuxt"):
		framework = "vue"
	case has("@angular/core"):
		framework = "angular"
	case has("svelte"):
		framework = "svelte"
	case has("vue"):
		framework = "vue"
	case has("react"):
		framework = "react"
	}

	// Styling detection.
	switch {
	case has("tailwindcss"):
		styling = "tailwind"
	case has("styled-components"):
		styling = "styled_components"
	case has("@emotion/react") || has("@emotion/styled"):
		styling = "emotion"
	default:
		styling = "css"
	}

	// Test runner detection.
	switch {
	case has("vitest"):
		testSetup = "vitest"
	case has("jest"):
		testSetup = "jest"
	case has("@playwright/test"):
		testSetup = "playwright"
	case has("cypress"):
		testSetup = "cypress"
	}

	return framework, styling, testSetup
}
