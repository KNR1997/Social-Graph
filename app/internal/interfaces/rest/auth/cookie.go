package auth

import (
	"net/http"
	"time"
)

// CookieConfig is the transport-level session cookie policy, built from
// configuration in the composition root so this package never reads the
// environment.
type CookieConfig struct {
	// Name is the cookie the session token travels in.
	Name string
	// Domain scopes the cookie. Empty means host-only.
	Domain string
	// Secure restricts the cookie to HTTPS.
	Secure bool
}

// set writes the session cookie for a freshly issued session.
//
// The flags are the whole security model of a cookie session, so they are worth
// stating plainly:
//
//   - HttpOnly keeps the token out of document.cookie, so an XSS bug can act as
//     the user for as long as it runs but cannot exfiltrate a credential that
//     outlives the page. This is the reason to prefer a cookie over
//     localStorage for a session token, not a stylistic one.
//   - SameSite=Lax is the CSRF defence: the browser withholds the cookie from
//     cross-site POST, PATCH and DELETE, which is every mutating route here.
//     It still sends it on a top-level GET navigation, which is why no GET in
//     this API changes state.
//   - Secure is forced on in production by config.SecureCookies.
//   - Path=/ because the SPA and the API share an origin behind the dev proxy.
//
// Expires matches the session's own expiry, so the browser stops sending a
// token the server would reject anyway.
func (c CookieConfig) set(w http.ResponseWriter, token string, expiresAt time.Time) {
	//nolint:gosec // G124: Secure is set from config.SecureCookies, which forces
	// it on in production; the linter cannot see through the field to prove it.
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name,
		Value:    token,
		Path:     "/",
		Domain:   c.Domain,
		Expires:  expiresAt.UTC(),
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clear expires the session cookie.
//
// The attributes have to match the ones used to set it -- name, path and domain
// identify a cookie, so a mismatch leaves the original in place and the browser
// keeps presenting a token the server has already deleted.
func (c CookieConfig) clear(w http.ResponseWriter) {
	//nolint:gosec // G124: same as set -- Secure comes from config.SecureCookies.
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name,
		Value:    "",
		Path:     "/",
		Domain:   c.Domain,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// token reads the session token out of the request, returning "" when there is
// no session cookie at all.
func (c CookieConfig) token(r *http.Request) string {
	cookie, err := r.Cookie(c.Name)
	if err != nil {
		return ""
	}

	return cookie.Value
}
