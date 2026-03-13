package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/store"
)

// ListNotifications handles GET /api/v1/orgs/{orgId}/notifications.
// Supports ?unread=true and standard limit/offset pagination.
func ListNotifications(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mw.UserIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		limit := 50
		offset := 0
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if n, err := strconv.Atoi(lStr); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}
		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			if n, err := strconv.Atoi(oStr); err == nil && n >= 0 {
				offset = n
			}
		}

		unreadOnly := r.URL.Query().Get("unread") == "true"

		notifs, total, err := d.Store.ListUserNotifications(
			r.Context(), userID, orgID, unreadOnly, store.Page(limit, offset),
		)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to list notifications")
			return
		}

		respondOK(w, r, map[string]any{
			"notifications": notifs,
			"total":         total,
		})
	}
}

// UnreadNotificationCount handles GET /api/v1/orgs/{orgId}/notifications/unread-count.
func UnreadNotificationCount(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mw.UserIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		count, err := d.Store.UnreadNotificationCount(r.Context(), userID, orgID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to count unread notifications")
			return
		}

		respondOK(w, r, map[string]any{"count": count})
	}
}

// MarkNotificationRead handles PATCH /api/v1/orgs/{orgId}/notifications/{notificationId}/read.
func MarkNotificationRead(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mw.UserIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		notifID, err := uuid.Parse(chi.URLParam(r, "notificationId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid notification_id")
			return
		}

		if err := d.Store.MarkNotificationRead(r.Context(), userID, orgID, notifID); err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to mark notification as read")
			return
		}

		respondNoContent(w, r)
	}
}

// MarkAllNotificationsRead handles POST /api/v1/orgs/{orgId}/notifications/read-all.
func MarkAllNotificationsRead(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mw.UserIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		if err := d.Store.MarkAllNotificationsRead(r.Context(), userID, orgID); err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to mark all notifications as read")
			return
		}

		respondNoContent(w, r)
	}
}
