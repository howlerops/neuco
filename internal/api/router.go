package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/neuco-ai/neuco/internal/api/handlers"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/domain"
)

// NewRouter constructs the full Chi router with all routes, middleware stacks,
// and handler registrations. It accepts a Deps bundle so handlers can access
// the store, River client, and configuration.
func NewRouter(d *Deps, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// ─── Global middleware ────────────────────────────────────────────────────
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.Recoverer)
	r.Use(mw.CORS(d.Config.FrontendURL))
	r.Use(mw.RequestLogger(logger))

	// ─── Public auth routes ───────────────────────────────────────────────────
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/github/callback", handlers.GitHubCallback(d))
		r.Post("/refresh", handlers.RefreshToken(d))
		r.Post("/logout", handlers.Logout(d))

		// Protected auth routes (require a valid JWT).
		r.Group(func(r chi.Router) {
			r.Use(mw.Authenticate(d.Config.JWTSecret))
			r.Get("/me", handlers.Me(d))
			r.Get("/github/repos", handlers.ListUserRepos(d))
			r.Post("/nango/connect-session", handlers.CreateNangoConnectSession(d))
		})
	})

	// ─── Webhook routes (no JWT, rate-limited) ───────────────────────────────
	r.Route("/api/v1/webhooks", func(r chi.Router) {
		r.Use(mw.WebhookRateLimit())
		r.Post("/{projectId}/{secret}", handlers.Webhook(d))
	})

	// ─── Authenticated API routes ─────────────────────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(mw.Authenticate(d.Config.JWTSecret))
		r.Use(mw.DefaultRateLimit())

		// Org-level routes.
		r.Route("/api/v1/orgs", func(r chi.Router) {
			r.Get("/", handlers.ListOrgs(d))
			r.Post("/", handlers.CreateOrg(d))

			r.Route("/{orgId}", func(r chi.Router) {
				r.Use(mw.ResolveOrg(d.Store))
				r.Get("/", handlers.GetOrg(d))
				r.With(mw.RequireRole(domain.OrgRoleAdmin)).Patch("/", handlers.UpdateOrg(d))

				// Member management.
				r.Route("/members", func(r chi.Router) {
					r.Get("/", handlers.ListMembers(d))
					r.With(mw.RequireRole(domain.OrgRoleAdmin)).Post("/invite", handlers.InviteMember(d))
					r.With(mw.RequireRole(domain.OrgRoleOwner)).Patch("/{userId}", handlers.UpdateMemberRole(d))
					r.With(mw.RequireRole(domain.OrgRoleAdmin)).Delete("/{userId}", handlers.RemoveMember(d))
				})

				// Projects under an org.
				r.Route("/projects", func(r chi.Router) {
					r.Get("/", handlers.ListOrgProjects(d))
					r.With(mw.RequireRole(domain.OrgRoleMember)).Post("/", handlers.CreateProject(d))
				})

				// GitHub App integration.
				// POST /installations — called by the frontend after the user
				//   completes the GitHub App install flow on github.com and GitHub
				//   appends ?installation_id=<n> to the configured callback URL.
				// GET  /repos — lists all repos accessible to the installation.
				r.Route("/github", func(r chi.Router) {
					r.With(mw.RequireRole(domain.OrgRoleAdmin)).
						Post("/installations", handlers.GitHubInstallCallback(d))
					r.Get("/repos", handlers.GitHubListRepos(d))
				})

				// Audit log.
				r.Get("/audit-log", handlers.AuditLog(d))
			})
		})

		// Project-scoped routes — tenant middleware verifies project belongs to org.
		r.Route("/api/v1/projects/{projectId}", func(r chi.Router) {
			r.Use(mw.ProjectTenant(d.Store))

			r.Get("/", handlers.GetProjectHandler(d))
			r.With(mw.RequireRole(domain.OrgRoleAdmin)).Patch("/", handlers.UpdateProject(d))
			r.With(mw.RequireRole(domain.OrgRoleAdmin)).Delete("/", handlers.DeleteProject(d))

			// Signals.
			r.Route("/signals", func(r chi.Router) {
				r.Get("/", handlers.ListSignals(d))
				r.Post("/upload", handlers.UploadSignals(d))
				r.Post("/query", handlers.QuerySignals(d))
				r.Delete("/{signalId}", handlers.DeleteSignal(d))
			})

			// Candidates.
			r.Route("/candidates", func(r chi.Router) {
				r.Get("/", handlers.ListCandidates(d))
				r.Post("/refresh", handlers.RefreshCandidates(d))
				r.Patch("/{cId}", handlers.UpdateCandidateStatus(d))

				// Specs (nested under candidates).
				r.Route("/{cId}/spec", func(r chi.Router) {
					r.Get("/", handlers.GetSpec(d))
					r.Patch("/", handlers.UpdateSpec(d))
					r.Post("/generate", handlers.GenerateSpec(d))
				})

				// Codegen (nested under candidates).
				r.With(mw.GenerationRateLimit()).
					Post("/{cId}/generate", handlers.EnqueueCodegen(d))
			})

			// Generations.
			r.Route("/generations", func(r chi.Router) {
				r.Get("/", handlers.ListGenerations(d))
				r.Get("/{gId}", handlers.GetGeneration(d))
				r.Get("/{gId}/stream", handlers.StreamGenerationProgress(d))
			})

			// Pipeline runs.
			r.Route("/pipelines", func(r chi.Router) {
				r.Get("/", handlers.ListPipelines(d))
				r.Get("/{runId}", handlers.GetPipeline(d))
				r.Post("/{runId}/retry", handlers.RetryPipeline(d))
			})

			// Project stats.
			r.Get("/stats", handlers.GetProjectStats(d))

			// Copilot notes.
			r.Route("/copilot/notes", func(r chi.Router) {
				r.Get("/", handlers.ListCopilotNotes(d))
				r.Patch("/{noteId}", handlers.DismissCopilotNote(d))
			})

			// Integrations.
			r.Route("/integrations", func(r chi.Router) {
				r.Get("/", handlers.ListIntegrations(d))
				r.With(mw.RequireRole(domain.OrgRoleAdmin)).Post("/", handlers.CreateIntegration(d))
				r.Get("/{integrationId}", handlers.GetIntegration(d))
				r.With(mw.RequireRole(domain.OrgRoleAdmin)).Delete("/{integrationId}", handlers.DeleteIntegration(d))
			})

			// Nango-managed integrations (OAuth via Nango frontend SDK).
			r.Route("/nango", func(r chi.Router) {
				r.Route("/connections", func(r chi.Router) {
					r.Get("/", handlers.ListNangoConnections(d))
					r.With(mw.RequireRole(domain.OrgRoleAdmin)).Post("/", handlers.CreateNangoConnection(d))
					r.With(mw.RequireRole(domain.OrgRoleAdmin)).Delete("/{connectionId}", handlers.DeleteNangoConnection(d))
				})
				r.Post("/sync/{connectionId}", handlers.TriggerNangoSync(d))
			})
		})
	})

	// ─── Internal operator routes ─────────────────────────────────────────────
	r.Route("/operator", func(r chi.Router) {
		r.Use(mw.InternalToken(d.Config.InternalAPIToken))

		r.Get("/orgs", handlers.OperatorListOrgs(d))
		r.Get("/orgs/{orgId}", handlers.OperatorGetOrg(d))
		r.Get("/users", handlers.OperatorListUsers(d))
		r.Get("/health", handlers.OperatorHealth(d))

		// Feature flags.
		r.Get("/flags", handlers.OperatorListFlags(d))
		r.Patch("/flags/{key}", handlers.OperatorUpdateFlag(d))
	})

	return r
}
