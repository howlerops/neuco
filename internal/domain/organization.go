package domain

import (
	"time"

	"github.com/google/uuid"
)

// OrgPlan represents the billing tier of an organisation.
type OrgPlan string

const (
	OrgPlanStarter    OrgPlan = "starter"
	OrgPlanPro        OrgPlan = "pro"
	OrgPlanEnterprise OrgPlan = "enterprise"
)

// OrgRole represents a member's permission level within an organisation.
type OrgRole string

const (
	OrgRoleOwner  OrgRole = "owner"
	OrgRoleAdmin  OrgRole = "admin"
	OrgRoleMember OrgRole = "member"
	OrgRoleViewer OrgRole = "viewer"
)

// Organization is the top-level tenant boundary. Every project and signal is
// scoped to an org.
type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Plan      OrgPlan   `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OrgMember links a User to an Organization with a role assignment.
type OrgMember struct {
	OrgID     uuid.UUID  `json:"org_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Role      OrgRole    `json:"role"`
	InvitedAt time.Time  `json:"invited_at"`
	JoinedAt  *time.Time `json:"joined_at,omitempty"`
}
