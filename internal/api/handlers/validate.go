package handlers

import (
	"fmt"

	"github.com/neuco-ai/neuco/internal/domain"
)

// Field length limits.
const (
	MaxNameLen        = 255
	MaxSlugLen        = 100
	MaxDescriptionLen = 10_000
	MaxContentLen     = 50_000 // spec fields, context content
	MaxTitleLen       = 500
	MaxPaginationLimit = 100
)

// validateStringLen checks that s does not exceed maxLen. Returns an error
// message suitable for the API response, or "" if valid.
func validateStringLen(field, s string, maxLen int) string {
	if len(s) > maxLen {
		return fmt.Sprintf("%s exceeds maximum length of %d characters", field, maxLen)
	}
	return ""
}

// Valid OrgRole values.
var validOrgRoles = map[domain.OrgRole]bool{
	domain.OrgRoleOwner:  true,
	domain.OrgRoleAdmin:  true,
	domain.OrgRoleMember: true,
	domain.OrgRoleViewer: true,
}

func isValidOrgRole(r domain.OrgRole) bool {
	return validOrgRoles[r]
}

// Valid CandidateStatus values.
var validCandidateStatuses = map[domain.CandidateStatus]bool{
	domain.CandidateStatusNew:        true,
	domain.CandidateStatusSpecced:    true,
	domain.CandidateStatusInProgress: true,
	domain.CandidateStatusReviewing:  true,
	domain.CandidateStatusAccepted:   true,
	domain.CandidateStatusRejected:   true,
	domain.CandidateStatusBacklogged: true,
	domain.CandidateStatusShipped:    true,
}

func isValidCandidateStatus(s domain.CandidateStatus) bool {
	return validCandidateStatuses[s]
}

// Valid ProjectFramework values (empty is allowed — means "not set").
var validFrameworks = map[domain.ProjectFramework]bool{
	"":                             true,
	domain.ProjectFrameworkReact:   true,
	domain.ProjectFrameworkNextJS:  true,
	domain.ProjectFrameworkVue:     true,
	domain.ProjectFrameworkSvelte:  true,
	domain.ProjectFrameworkAngular: true,
}

func isValidFramework(f domain.ProjectFramework) bool {
	return validFrameworks[f]
}

// Valid ProjectStyling values (empty is allowed).
var validStylings = map[domain.ProjectStyling]bool{
	"":                            true,
	domain.ProjectStylingTailwind: true,
	domain.ProjectStylingCSS:      true,
	domain.ProjectStylingModules:  true,
	domain.ProjectStylingStyled:   true,
	domain.ProjectStylingEmotion:  true,
}

func isValidStyling(s domain.ProjectStyling) bool {
	return validStylings[s]
}

// Valid ContextCategory values (empty defaults to "insight" in handler).
var validContextCategories = map[domain.ContextCategory]bool{
	"":                                true,
	domain.ContextCategoryInsight:     true,
	domain.ContextCategoryTheme:       true,
	domain.ContextCategoryDecision:    true,
	domain.ContextCategoryRisk:        true,
	domain.ContextCategoryOpportunity: true,
}

func isValidContextCategory(c domain.ContextCategory) bool {
	return validContextCategories[c]
}

// ptrStr dereferences a *string, returning "" if nil.
func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// clampPagination ensures limit is within [1, MaxPaginationLimit].
func clampPagination(limit int) int {
	if limit < 1 {
		return 50
	}
	if limit > MaxPaginationLimit {
		return MaxPaginationLimit
	}
	return limit
}
