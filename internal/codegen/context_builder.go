package codegen

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/generation"
)

const (
	defaultContextTokenBudget = 50000
	defaultContextMaxFileSize = 100 * 1024
	largeFilePenaltyThreshold = 50 * 1024
)

// ContextBundle is the rich context payload supplied to a generation run.
type ContextBundle struct {
	Files       []ContextFile   `json:"files"`
	Manifest    []ManifestEntry `json:"manifest"`
	TotalTokens int             `json:"total_tokens"`
	Truncated   bool            `json:"truncated"`
}

// ContextFile is a selected repository file with content and relevance metadata.
type ContextFile struct {
	Path          string  `json:"path"`
	Content       string  `json:"content"`
	Score         float64 `json:"score"`
	TokenEstimate int     `json:"token_estimate"`
}

// ManifestEntry summarizes one selected file for lightweight display/logging.
type ManifestEntry struct {
	Path          string `json:"path"`
	Lines         int    `json:"lines"`
	TokenEstimate int    `json:"token_estimate"`
	Relevance     string `json:"relevance"`
}

// ContextBuilderConfig controls rich-context sizing and relevance filtering.
type ContextBuilderConfig struct {
	TokenBudget int     `json:"token_budget"`
	MaxFileSize int     `json:"max_file_size"`
	MinScore    float64 `json:"min_score"`
}

// BuildRichContext selects and packs high-signal repository files and content
// into a context bundle sized for agentic code generation.
func BuildRichContext(
	spec domain.Spec,
	index generation.RepoIndex,
	files map[string]string,
	cfg ContextBuilderConfig,
) (*ContextBundle, error) {
	cfg = normalizeContextConfig(cfg)

	keywords := extractKeywords(spec)
	anchors := buildAnchorPathSet(index)
	mustInclude := mustIncludePaths(files)

	type candidate struct {
		path    string
		content string
		score   float64
		tokens  int
		lines   int
		reason  string
		must    bool
	}

	candidates := make([]candidate, 0, len(files))
	for path, content := range files {
		if isGeneratedOrVendor(path) {
			continue
		}

		trimmedContent := content
		reasons := make([]string, 0, 6)
		score := scoreFile(path, content, keywords, index)

		if strings.Contains(strings.ToLower(path), "test") || isTestFile(path) {
			if isTestNearLikelyTarget(path, anchors) {
				reasons = append(reasons, "test near likely target")
			}
		}
		if isConfigOrConventionFile(path) {
			reasons = append(reasons, "project config/conventions")
		}
		if pathMatchesKeywords(path, keywords) {
			reasons = append(reasons, "path matches spec keywords")
		}
		if contentMatchesKeywords(content, keywords) {
			reasons = append(reasons, "content matches spec keywords")
		}
		if matchesAnchor(path, anchors) {
			reasons = append(reasons, "matches indexed component/story/type")
		}

		if len(content) > cfg.MaxFileSize {
			trimmedContent = content[:cfg.MaxFileSize]
			reasons = append(reasons, fmt.Sprintf("truncated to %d bytes", cfg.MaxFileSize))
		}
		if len(content) > largeFilePenaltyThreshold {
			reasons = append(reasons, "large file penalty")
		}

		if len(reasons) == 0 {
			reasons = append(reasons, "general contextual relevance")
		}

		entry := candidate{
			path:    path,
			content: trimmedContent,
			score:   score,
			tokens:  estimateTokens(trimmedContent),
			lines:   countLines(trimmedContent),
			reason:  strings.Join(reasons, "; "),
			must:    mustInclude[path],
		}
		candidates = append(candidates, entry)
	}

	if len(candidates) == 0 {
		return &ContextBundle{}, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].must != candidates[j].must {
			return candidates[i].must
		}
		if candidates[i].score == candidates[j].score {
			if candidates[i].tokens == candidates[j].tokens {
				return candidates[i].path < candidates[j].path
			}
			return candidates[i].tokens < candidates[j].tokens
		}
		return candidates[i].score > candidates[j].score
	})

	bundle := &ContextBundle{}
	usedTokens := 0
	selected := map[string]struct{}{}

	appendCandidate := func(c candidate, force bool) {
		if _, exists := selected[c.path]; exists {
			return
		}
		if !force && c.score < cfg.MinScore {
			return
		}
		if !force && usedTokens+c.tokens > cfg.TokenBudget {
			bundle.Truncated = true
			return
		}
		if force && usedTokens+c.tokens > cfg.TokenBudget {
			bundle.Truncated = true
		}

		bundle.Files = append(bundle.Files, ContextFile{
			Path:          c.path,
			Content:       c.content,
			Score:         c.score,
			TokenEstimate: c.tokens,
		})
		bundle.Manifest = append(bundle.Manifest, ManifestEntry{
			Path:          c.path,
			Lines:         c.lines,
			TokenEstimate: c.tokens,
			Relevance:     c.reason,
		})
		selected[c.path] = struct{}{}
		usedTokens += c.tokens
	}

	for _, c := range candidates {
		if c.must {
			appendCandidate(c, true)
		}
	}
	for _, c := range candidates {
		if !c.must {
			appendCandidate(c, false)
		}
	}

	bundle.TotalTokens = usedTokens
	return bundle, nil
}

func normalizeContextConfig(cfg ContextBuilderConfig) ContextBuilderConfig {
	if cfg.TokenBudget <= 0 {
		cfg.TokenBudget = defaultContextTokenBudget
	}
	if cfg.MaxFileSize <= 0 {
		cfg.MaxFileSize = defaultContextMaxFileSize
	}
	if cfg.MinScore == 0 {
		cfg.MinScore = 1
	}
	return cfg
}

// extractKeywords generates a de-duplicated keyword list from spec content.
func extractKeywords(spec domain.Spec) []string {
	var builder strings.Builder
	builder.WriteString(spec.ProblemStatement)
	builder.WriteString("\n")
	builder.WriteString(spec.ProposedSolution)
	builder.WriteString("\n")
	builder.WriteString(spec.UIChanges)
	builder.WriteString("\n")
	builder.WriteString(spec.DataModelChanges)
	builder.WriteString("\n")
	for _, story := range spec.UserStories {
		builder.WriteString(story.Role)
		builder.WriteString(" ")
		builder.WriteString(story.Want)
		builder.WriteString(" ")
		builder.WriteString(story.SoThat)
		builder.WriteString("\n")
	}
	for _, ac := range spec.AcceptanceCriteria {
		builder.WriteString(ac)
		builder.WriteString("\n")
	}

	stopwords := map[string]struct{}{
		"the": {}, "and": {}, "for": {}, "that": {}, "with": {}, "from": {},
		"this": {}, "into": {}, "your": {}, "have": {}, "will": {}, "when": {},
		"want": {}, "so": {}, "are": {}, "but": {}, "not": {}, "use": {},
		"as": {}, "a": {}, "an": {}, "to": {}, "of": {}, "in": {}, "on": {},
	}

	parts := strings.FieldsFunc(strings.ToLower(builder.String()), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '_' || r == '/')
	})

	keywords := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, "-_/ ")
		if len(part) < 3 {
			continue
		}
		if _, blocked := stopwords[part]; blocked {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		keywords = append(keywords, part)
	}
	return keywords
}

// estimateTokens approximates token count using 1 token ~= 4 characters.
func estimateTokens(content string) int {
	return (len(content) + 3) / 4
}

// isGeneratedOrVendor excludes third-party or build output files.
func isGeneratedOrVendor(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	blocked := []string{
		"/node_modules/", "node_modules/",
		"/dist/", "dist/",
		"/.git/", ".git/",
		"/vendor/", "vendor/",
		"/build/", "build/",
		"/.next/", ".next/",
	}
	for _, marker := range blocked {
		if strings.Contains(lower, marker) || strings.HasPrefix(lower, marker) {
			return true
		}
	}
	return strings.HasSuffix(lower, ".min.js") || strings.HasSuffix(lower, ".map")
}

// scoreFile computes the weighted relevance score for one file.
func scoreFile(path string, content string, keywords []string, index generation.RepoIndex) float64 {
	if isGeneratedOrVendor(path) {
		return math.Inf(-1)
	}

	score := 0.0
	anchors := buildAnchorPathSet(index)

	if matchesAnchor(path, anchors) {
		score += 50
	}
	if pathMatchesKeywords(path, keywords) {
		score += 25
	}
	if content != "" && contentMatchesKeywords(content, keywords) {
		score += 20
	}
	if isTestFile(path) && isTestNearLikelyTarget(path, anchors) {
		score += 15
	}
	if isConfigOrConventionFile(path) {
		score += 10
	}
	if len(content) > largeFilePenaltyThreshold {
		score -= 30
	}

	return score
}

func buildAnchorPathSet(index generation.RepoIndex) map[string]struct{} {
	anchors := make(map[string]struct{})
	for _, c := range index.Components {
		anchors[normalizePath(c.Path)] = struct{}{}
	}
	for _, s := range index.Stories {
		anchors[normalizePath(s.Path)] = struct{}{}
	}
	for _, t := range index.TypeFiles {
		anchors[normalizePath(t.Path)] = struct{}{}
	}
	for _, d := range index.DesignTokens {
		anchors[normalizePath(d.Path)] = struct{}{}
	}
	return anchors
}

func mustIncludePaths(files map[string]string) map[string]bool {
	must := make(map[string]bool)
	for path := range files {
		norm := normalizePath(path)
		base := strings.ToLower(filepath.Base(norm))
		switch {
		case strings.HasPrefix(base, "readme"):
			must[path] = true
		case base == "package.json":
			must[path] = true
		case base == "tsconfig.json":
			must[path] = true
		case base == "go.mod":
			must[path] = true
		}
	}
	return must
}

func normalizePath(path string) string {
	return strings.ToLower(filepath.ToSlash(path))
}

func matchesAnchor(path string, anchors map[string]struct{}) bool {
	norm := normalizePath(path)
	if _, ok := anchors[norm]; ok {
		return true
	}
	base := strings.ToLower(filepath.Base(norm))
	for anchor := range anchors {
		if strings.HasSuffix(anchor, "/"+base) {
			return true
		}
	}
	return false
}

func pathMatchesKeywords(path string, keywords []string) bool {
	lower := normalizePath(path)
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func contentMatchesKeywords(content string, keywords []string) bool {
	lower := strings.ToLower(content)
	matches := 0
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			matches++
			if matches >= 2 {
				return true
			}
		}
	}
	return matches > 0
}

func isConfigOrConventionFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	known := map[string]struct{}{
		"package.json": {},
		"tsconfig.json": {},
		"tsconfig.base.json": {},
		"go.mod": {},
		"go.sum": {},
		".eslintrc": {},
		".eslintrc.js": {},
		".eslintrc.cjs": {},
		".eslintrc.json": {},
		".prettierrc": {},
		".prettierrc.js": {},
		".prettierrc.json": {},
		"biome.json": {},
		"golangci.yml": {},
		"golangci.yaml": {},
		"readme.md": {},
		"contributing.md": {},
	}
	if _, ok := known[base]; ok {
		return true
	}
	return strings.HasPrefix(base, "tsconfig") || strings.HasPrefix(base, "eslint")
}

func isTestFile(path string) bool {
	lower := normalizePath(path)
	return strings.HasSuffix(lower, "_test.go") ||
		strings.Contains(lower, ".test.") ||
		strings.Contains(lower, ".spec.")
}

func isTestNearLikelyTarget(path string, anchors map[string]struct{}) bool {
	norm := normalizePath(path)
	base := strings.ToLower(filepath.Base(norm))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.TrimSuffix(base, ".test")
	base = strings.TrimSuffix(base, ".spec")
	base = strings.TrimSuffix(base, "_test")

	dir := filepath.Dir(norm)
	for anchor := range anchors {
		if strings.HasPrefix(anchor, dir) {
			return true
		}
		anchorBase := strings.ToLower(filepath.Base(anchor))
		anchorBase = strings.TrimSuffix(anchorBase, filepath.Ext(anchorBase))
		if anchorBase == base || strings.Contains(anchorBase, base) || strings.Contains(base, anchorBase) {
			return true
		}
	}
	return false
}

func countLines(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}
