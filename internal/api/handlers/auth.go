package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

// githubUser is the response from the GitHub user API endpoint.
type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// tokenPair holds the access and refresh JWTs returned to the client.
// The refresh token is set as an httpOnly cookie, not in the JSON body.
type tokenPair struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // seconds
}

const (
	refreshCookieName   = "neuco_refresh"
	refreshCookieMaxAge = 30 * 24 * 60 * 60 // 30 days in seconds
)

// meResponse is the response body for GET /api/v1/auth/me.
type meResponse struct {
	User domain.User            `json:"user"`
	Orgs []domain.Organization  `json:"orgs"`
}

func githubOAuthConfig(d *Deps) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     d.Config.GitHubClientID,
		ClientSecret: d.Config.GitHubClientSecret,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     githuboauth.Endpoint,
	}
}

func fetchGitHubUser(ctx context.Context, accessToken string) (*githubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var gu githubUser
	if err := json.NewDecoder(resp.Body).Decode(&gu); err != nil {
		return nil, err
	}
	return &gu, nil
}

// setRefreshCookie writes the refresh token as a Secure, HttpOnly, SameSite=Strict cookie.
func setRefreshCookie(w http.ResponseWriter, d *Deps, refreshToken string) {
	secure := d.Config.FrontendURL != "" && d.Config.FrontendURL != "http://localhost:5173"
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     "/api/v1/auth",
		MaxAge:   refreshCookieMaxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearRefreshCookie removes the refresh cookie.
func clearRefreshCookie(w http.ResponseWriter, d *Deps) {
	secure := d.Config.FrontendURL != "" && d.Config.FrontendURL != "http://localhost:5173"
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// issuedTokens holds both access and refresh JWTs. The refresh token is sent as
// an httpOnly cookie, while the access token is in the JSON response.
type issuedTokens struct {
	pair         tokenPair
	refreshToken string
}

func issueTokenPair(d *Deps, user domain.User, orgID uuid.UUID, role domain.OrgRole) (*issuedTokens, error) {
	secret := []byte(d.Config.JWTSecret)
	now := time.Now()

	accessClaims := mw.NeuClaims{
		UserID: user.ID.String(),
		OrgID:  orgID.String(),
		Email:  user.Email,
		Role:   string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(secret)
	if err != nil {
		return nil, err
	}

	refreshClaims := jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
		Audience:  jwt.ClaimStrings{"refresh"},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(secret)
	if err != nil {
		return nil, err
	}

	return &issuedTokens{
		pair: tokenPair{
			AccessToken: accessToken,
			ExpiresIn:   int((24 * time.Hour).Seconds()),
		},
		refreshToken: refreshToken,
	}, nil
}

// GitHubCallback handles POST /api/v1/auth/github/callback.
// Exchanges the OAuth code for a GitHub access token, fetches the user profile,
// upserts the user, auto-creates a personal org on first login, and issues a JWT pair.
func GitHubCallback(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Code string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
			respondErr(w, r, http.StatusBadRequest, "missing or invalid code")
			return
		}

		cfg := githubOAuthConfig(d)
		ghToken, err := cfg.Exchange(r.Context(), req.Code)
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "failed to exchange OAuth code: "+err.Error())
			return
		}

		ghUser, err := fetchGitHubUser(r.Context(), ghToken.AccessToken)
		if err != nil {
			respondErr(w, r, http.StatusBadGateway, "failed to fetch GitHub user: "+err.Error())
			return
		}

		// UpsertUser takes explicit fields (matching the store signature).
		user, err := d.Store.UpsertUser(r.Context(), ghUser.ID, ghUser.Login, ghUser.Email, ghUser.AvatarURL)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to upsert user")
			return
		}

		// Persist the GitHub OAuth token so user-scoped API calls can use it later.
		if err := d.Store.SetUserGitHubToken(r.Context(), user.ID, ghToken.AccessToken); err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to store github token")
			return
		}

		// Check if this user already has orgs; if not, create a personal org.
		orgs, err := d.Store.ListUserOrgs(r.Context(), user.ID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to fetch user orgs")
			return
		}

		var primaryOrg domain.Organization
		if len(orgs) == 0 {
			// First login — create personal org and add user as owner.
			slug := fmt.Sprintf("%s-personal", ghUser.Login)
			primaryOrg, err = d.Store.CreateOrg(r.Context(), fmt.Sprintf("%s's workspace", ghUser.Login), slug, domain.OrgPlanStarter)
			if err != nil {
				respondErr(w, r, http.StatusInternalServerError, "failed to create personal org")
				return
			}
			if _, err := d.Store.AddMember(r.Context(), primaryOrg.ID, user.ID, domain.OrgRoleOwner); err != nil {
				respondErr(w, r, http.StatusInternalServerError, "failed to add owner to personal org")
				return
			}
		} else {
			primaryOrg = orgs[0]
		}

		role, err := d.Store.GetMemberRole(r.Context(), primaryOrg.ID, user.ID)
		if err != nil {
			role = domain.OrgRoleOwner // safe default for the org owner
		}

		tokens, err := issueTokenPair(d, user, primaryOrg.ID, role)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to issue tokens")
			return
		}

		setRefreshCookie(w, d, tokens.refreshToken)
		respondOK(w, r, tokens.pair)
	}
}

// Me handles GET /api/v1/auth/me.
// Returns the current user and all their organisations.
func Me(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mw.UserIDFromCtx(r.Context())

		user, err := d.Store.GetUserByID(r.Context(), userID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to fetch user")
			return
		}

		orgs, err := d.Store.ListUserOrgs(r.Context(), userID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to fetch orgs")
			return
		}

		respondOK(w, r, meResponse{User: user, Orgs: orgs})
	}
}

// RefreshToken handles POST /api/v1/auth/refresh.
// Reads the refresh token from the httpOnly cookie and issues a new token pair.
func RefreshToken(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(refreshCookieName)
		if err != nil || cookie.Value == "" {
			respondErr(w, r, http.StatusUnauthorized, "missing refresh token")
			return
		}

		secret := []byte(d.Config.JWTSecret)
		claims := &jwt.RegisteredClaims{}

		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secret, nil
		}, jwt.WithValidMethods([]string{"HS256"}),
			jwt.WithAudience("refresh"))
		if err != nil || !token.Valid {
			clearRefreshCookie(w, d)
			respondErr(w, r, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			respondErr(w, r, http.StatusUnauthorized, "invalid subject in refresh token")
			return
		}

		user, err := d.Store.GetUserByID(r.Context(), userID)
		if err != nil {
			respondErr(w, r, http.StatusUnauthorized, "user not found")
			return
		}

		orgs, err := d.Store.ListUserOrgs(r.Context(), userID)
		if err != nil || len(orgs) == 0 {
			respondErr(w, r, http.StatusInternalServerError, "failed to fetch orgs")
			return
		}

		org := orgs[0]
		role, err := d.Store.GetMemberRole(r.Context(), org.ID, userID)
		if err != nil {
			role = domain.OrgRoleMember
		}

		tokens, err := issueTokenPair(d, user, org.ID, role)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to issue tokens")
			return
		}

		setRefreshCookie(w, d, tokens.refreshToken)
		respondOK(w, r, tokens.pair)
	}
}

// Logout handles POST /api/v1/auth/logout.
// Clears the refresh token cookie.
func Logout(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clearRefreshCookie(w, d)
		respondNoContent(w, r)
	}
}
