package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
)

// inviteMemberRequest is the request body for POST /api/v1/orgs/{orgId}/members/invite.
type inviteMemberRequest struct {
	UserID string         `json:"user_id"` // direct user ID — email invite is a follow-up
	Role   domain.OrgRole `json:"role"`
}

// updateMemberRoleRequest is the request body for PATCH /api/v1/orgs/{orgId}/members/{userId}.
type updateMemberRoleRequest struct {
	Role domain.OrgRole `json:"role"`
}

// ListMembers handles GET /api/v1/orgs/{orgId}/members.
func ListMembers(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := mw.ResolvedOrgIDFromCtx(r.Context())

		members, err := d.Store.ListOrgMembers(r.Context(), orgID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to list members")
			return
		}

		respondOK(w, r, members)
	}
}

// InviteMember handles POST /api/v1/orgs/{orgId}/members/invite. Requires admin+ role.
// For now, the user is added directly without an email invite.
func InviteMember(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := mw.ResolvedOrgIDFromCtx(r.Context())

		var req inviteMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
			respondErr(w, r, http.StatusBadRequest, "user_id is required")
			return
		}

		targetUserID, err := uuid.Parse(req.UserID)
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid user_id")
			return
		}

		role := req.Role
		if role == "" {
			role = domain.OrgRoleMember
		}

		member, err := d.Store.AddMember(r.Context(), orgID, targetUserID, role)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to add member")
			return
		}

		recordAudit(r.Context(), d, orgID, "member.invite", "org_member", targetUserID.String(),
			map[string]any{"role": role})
		respondCreated(w, r, member)
	}
}

// UpdateMemberRole handles PATCH /api/v1/orgs/{orgId}/members/{userId}. Requires owner role.
func UpdateMemberRole(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := mw.ResolvedOrgIDFromCtx(r.Context())

		targetUserID, err := uuid.Parse(chi.URLParam(r, "userId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid user_id")
			return
		}

		var req updateMemberRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Role == "" {
			respondErr(w, r, http.StatusBadRequest, "role is required")
			return
		}

		member, err := d.Store.UpdateMemberRole(r.Context(), orgID, targetUserID, req.Role)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to update member role")
			return
		}

		recordAudit(r.Context(), d, orgID, "member.role_change", "org_member", targetUserID.String(),
			map[string]any{"new_role": req.Role})
		respondOK(w, r, member)
	}
}

// RemoveMember handles DELETE /api/v1/orgs/{orgId}/members/{userId}. Requires admin+ role.
// Cannot remove the org owner.
func RemoveMember(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := mw.ResolvedOrgIDFromCtx(r.Context())

		targetUserID, err := uuid.Parse(chi.URLParam(r, "userId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid user_id")
			return
		}

		// Prevent removing the org owner.
		targetRole, err := d.Store.GetMemberRole(r.Context(), orgID, targetUserID)
		if err != nil {
			respondErr(w, r, http.StatusNotFound, "member not found")
			return
		}
		if targetRole == domain.OrgRoleOwner {
			respondErr(w, r, http.StatusConflict, "cannot remove the org owner")
			return
		}

		callerID := mw.UserIDFromCtx(r.Context())
		if callerID == targetUserID {
			respondErr(w, r, http.StatusBadRequest, "cannot remove yourself")
			return
		}

		if err := d.Store.RemoveMember(r.Context(), orgID, targetUserID); err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to remove member")
			return
		}

		recordAudit(r.Context(), d, orgID, "member.remove", "org_member", targetUserID.String(), nil)
		respondNoContent(w, r)
	}
}
