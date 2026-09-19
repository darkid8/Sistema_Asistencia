package http

import (
	"net/http"
	"time"
)

// sessionCookieName is the cookie that carries the session token once a user
// signs in. HttpOnly keeps it out of reach of any script running on the page,
// so an XSS payload cannot read or exfiltrate it -- unlike the token that used
// to sit in window.localStorage.
const sessionCookieName = "workshop_session"

// setSessionCookie writes the session token as an HttpOnly, SameSite=Lax
// cookie. SameSite=Lax (rather than Strict) still blocks the cookie being
// attached to a cross-site POST -- the CSRF-relevant case -- while allowing it
// on a normal top-level navigation into the app. secure should be true in
// every deployment that serves the app over HTTPS; it only needs to be false
// for a local plain-HTTP run, since a browser never sends a Secure cookie
// over an insecure connection.
func setSessionCookie(writer http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie deletes the session cookie. A script cannot delete an
// HttpOnly cookie itself, so sign out must go through the server.
func clearSessionCookie(writer http.ResponseWriter, secure bool) {
	http.SetCookie(writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// tokenFromRequest reads the session token, preferring the HttpOnly cookie
// the browser sends automatically. The Authorization header is still
// accepted so a non-browser API client (a script, a test, an integration)
// keeps working the same way it always has; the frontend itself no longer
// reads or sends a token it can put in a header.
func tokenFromRequest(request *http.Request) (string, bool) {
	if cookie, err := request.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, true
	}
	header := request.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) > len(prefix) && header[:len(prefix)] == prefix {
		return header[len(prefix):], true
	}
	return "", false
}
