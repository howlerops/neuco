package domain

import (
	"time"

	"github.com/google/uuid"
)

// ProjectFramework represents the frontend framework used by the project's codebase.
type ProjectFramework string

const (
	ProjectFrameworkReact   ProjectFramework = "react"
	ProjectFrameworkNextJS  ProjectFramework = "nextjs"
	ProjectFrameworkVue     ProjectFramework = "vue"
	ProjectFrameworkSvelte  ProjectFramework = "svelte"
	ProjectFrameworkAngular ProjectFramework = "angular"
)

// ProjectStyling represents the CSS/styling approach used by the project.
type ProjectStyling string

const (
	ProjectStylingTailwind  ProjectStyling = "tailwind"
	ProjectStylingCSS       ProjectStyling = "css"
	ProjectStylingModules   ProjectStyling = "css_modules"
	ProjectStylingStyled    ProjectStyling = "styled_components"
	ProjectStylingEmotion   ProjectStyling = "emotion"
)

// Project belongs to an Organization and represents a connected codebase that
// Neuco will generate code for.
type Project struct {
	ID         uuid.UUID        `json:"id"`
	OrgID      uuid.UUID        `json:"org_id"`
	Name       string           `json:"name"`
	GitHubRepo string           `json:"github_repo"`
	Framework  ProjectFramework `json:"framework"`
	Styling    ProjectStyling   `json:"styling"`
	CreatedBy  uuid.UUID        `json:"created_by"`
	CreatedAt  time.Time        `json:"created_at"`
}
