package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"

	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/generation"
)

// installGitHubAppRequest is the body for POST …/github/installations.
// When GitHub redirects after an App installation, the client receives an
// installation_id query parameter. The frontend passes it here.
type installGitHubAppRequest struct {
	InstallationID int64 `json:"installation_id"`
}

// GitHubInstallCallback handles POST /api/v1/orgs/{orgId}/github/installations.
//
// When an org owner installs the Neuco GitHub App via the GitHub App settings
// page, GitHub appends ?installation_id=<n>&setup_action=install to the
// "Setup URL" configured in the App. The frontend receives that redirect,
// extracts the installation_id, and calls this endpoint to persist it.
//
// Storing the installation_id allows the code generation workers to request
// short-lived installation access tokens for any repository accessible to
// the installation.
func GitHubInstallCallback(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := mw.ResolvedOrgIDFromCtx(r.Context())

		// Accept installation_id either from the JSON body or as a query param
		// (GET-style redirects from GitHub pass it as ?installation_id=…).
		var installationID int64
		if r.Header.Get("Content-Type") == "application/json" {
			var req installGitHubAppRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InstallationID == 0 {
				respondErr(w, r, http.StatusBadRequest, "installation_id is required")
				return
			}
			installationID = req.InstallationID
		} else {
			raw := r.URL.Query().Get("installation_id")
			if raw == "" {
				respondErr(w, r, http.StatusBadRequest, "installation_id is required")
				return
			}
			var err error
			installationID, err = strconv.ParseInt(raw, 10, 64)
			if err != nil || installationID <= 0 {
				respondErr(w, r, http.StatusBadRequest, "installation_id must be a positive integer")
				return
			}
		}

		if err := d.Store.SetOrgGitHubInstallation(r.Context(), orgID, installationID); err != nil {
			slog.Error("failed to store github installation",
				"org_id", orgID,
				"installation_id", installationID,
				"error", err,
			)
			respondErr(w, r, http.StatusInternalServerError, "failed to store github installation")
			return
		}

		userID := mw.UserIDFromCtx(r.Context())
		recordAudit(r.Context(), d, orgID, "github.installation.created", "org", orgID.String(),
			map[string]any{
				"installation_id": installationID,
				"user_id":         userID.String(),
			},
		)

		slog.Info("github app installation stored",
			"org_id", orgID,
			"installation_id", installationID,
		)

		respondCreated(w, r, map[string]any{
			"org_id":          orgID,
			"installation_id": installationID,
		})
	}
}

// GitHubListRepos handles GET /api/v1/orgs/{orgId}/github/repos.
//
// Returns the list of repositories accessible to the GitHub App installation
// for the org. Requires the GitHub App to be configured (GITHUB_APP_ID and
// GITHUB_APP_PRIVATE_KEY_PATH must be set) and the App to be installed on the
// org's GitHub account.
func GitHubListRepos(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := mw.ResolvedOrgIDFromCtx(r.Context())

		if d.Config.GitHubAppID == "" || d.Config.GitHubAppPrivateKeyPath == "" {
			respondErr(w, r, http.StatusServiceUnavailable, "github app not configured")
			return
		}

		installationID, err := d.Store.GetOrgGitHubInstallation(r.Context(), orgID)
		if err != nil {
			respondErr(w, r, http.StatusNotFound, "github app not installed for this org")
			return
		}

		ghSvc, err := generation.NewGitHubService(d.Config.GitHubAppID, d.Config.GitHubAppPrivateKeyPath)
		if err != nil {
			slog.Error("failed to initialise github service", "error", err)
			respondErr(w, r, http.StatusInternalServerError, "github service unavailable")
			return
		}

		token, err := ghSvc.GetInstallationToken(r.Context(), installationID)
		if err != nil {
			slog.Error("failed to get installation token",
				"org_id", orgID,
				"installation_id", installationID,
				"error", err,
			)
			respondErr(w, r, http.StatusBadGateway, "could not authenticate with github")
			return
		}

		// Use the installation token directly to list repos accessible to it.
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tc := oauth2.NewClient(r.Context(), ts)
		ghClient := github.NewClient(tc)

		var allRepos []*github.Repository
		opts := &github.ListOptions{PerPage: 100}
		for {
			repos, resp, err := ghClient.Apps.ListRepos(r.Context(), &github.ListOptions{
				Page:    opts.Page,
				PerPage: opts.PerPage,
			})
			if err != nil {
				slog.Error("failed to list installation repos",
					"org_id", orgID,
					"error", err,
				)
				respondErr(w, r, http.StatusBadGateway, "could not list github repositories")
				return
			}
			allRepos = append(allRepos, repos.Repositories...)
			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}

		type repoSummary struct {
			ID          int64  `json:"id"`
			FullName    string `json:"full_name"`
			Name        string `json:"name"`
			Private     bool   `json:"private"`
			HTMLURL     string `json:"html_url"`
			Description string `json:"description,omitempty"`
			Language    string `json:"language,omitempty"`
		}

		result := make([]repoSummary, 0, len(allRepos))
		for _, r := range allRepos {
			result = append(result, repoSummary{
				ID:          r.GetID(),
				FullName:    r.GetFullName(),
				Name:        r.GetName(),
				Private:     r.GetPrivate(),
				HTMLURL:     r.GetHTMLURL(),
				Description: r.GetDescription(),
				Language:    r.GetLanguage(),
			})
		}

		respondOK(w, r, map[string]any{
			"repositories": result,
			"total":        len(result),
		})
	}
}
