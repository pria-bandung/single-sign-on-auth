package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// googleUserinfoURL is the OpenID Connect userinfo endpoint. Using the OIDC
// endpoint (rather than the older v2 one) returns the standard claims sub /
// email / email_verified / name and keeps us aligned for a future switch to
// verifying the id_token directly.
const googleUserinfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

// GoogleIdentity is the subset of Google profile data the app cares about.
type GoogleIdentity struct {
	Sub           string
	Email         string
	EmailVerified bool
	Name          string
}

// GoogleOAuth encapsulates the Google sign-in flow: building the consent URL,
// exchanging the authorization code for a token, and fetching the user's
// identity. Each step is exposed so the flow remains visible to the caller.
type GoogleOAuth struct {
	config      *oauth2.Config
	userinfoURL string
}

// NewGoogleOAuth builds a GoogleOAuth for the "Sign in with Google" flow with
// the openid/email/profile scopes.
func NewGoogleOAuth(clientID, clientSecret, redirectURL string) *GoogleOAuth {
	return &GoogleOAuth{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
		userinfoURL: googleUserinfoURL,
	}
}

// AuthCodeURL returns the Google consent-screen URL to redirect the user to,
// embedding the given CSRF state value.
func (g *GoogleOAuth) AuthCodeURL(state string) string {
	return g.config.AuthCodeURL(state)
}

// Exchange swaps an authorization code for an OAuth token.
func (g *GoogleOAuth) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.config.Exchange(ctx, code)
}

// FetchIdentity calls Google's userinfo endpoint with the token and returns the
// caller's identity.
func (g *GoogleOAuth) FetchIdentity(ctx context.Context, token *oauth2.Token) (*GoogleIdentity, error) {
	client := g.config.Client(ctx, token)
	resp, err := client.Get(g.userinfoURL)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read userinfo: %w", err)
	}
	return parseGoogleIdentity(body)
}

// parseGoogleIdentity decodes a userinfo JSON payload into a GoogleIdentity.
func parseGoogleIdentity(body []byte) (*GoogleIdentity, error) {
	var raw struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}
	return &GoogleIdentity{
		Sub:           raw.Sub,
		Email:         raw.Email,
		EmailVerified: raw.EmailVerified,
		Name:          raw.Name,
	}, nil
}
