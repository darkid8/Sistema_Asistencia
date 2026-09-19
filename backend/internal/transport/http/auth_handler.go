package http

import (
	"net/http"
	"time"

	"workshop/internal/domain"
	"workshop/internal/usecase"
)

// signInRequest is the credential payload the login screen sends.
type signInRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// sessionResponse is what a successful sign in returns. It never carries the
// password hash.
type sessionResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	UserID    string `json:"userId"`
	Username  string `json:"username"`
	FullName  string `json:"fullName"`
	Role      string `json:"role"`
}

// AuthHandler exposes the sign in operation. It also throttles repeated
// failed attempts for the same username/IP pair so a script cannot try
// passwords without limit (OWASP A07 - identification and authentication
// failures).
type AuthHandler struct {
	authenticate usecase.AuthenticateUser
	limiter      *loginRateLimiter
	now          func() time.Time
	cookieSecure bool
}

// NewAuthHandler wires the authentication handler. cookieSecure marks the
// session cookie Secure (HTTPS only) and should be true in every deployment
// that serves the app over HTTPS.
func NewAuthHandler(authenticate usecase.AuthenticateUser, now func() time.Time, cookieSecure bool) AuthHandler {
	return AuthHandler{
		authenticate: authenticate,
		limiter:      newLoginRateLimiter(5, 15*time.Minute),
		now:          now,
		cookieSecure: cookieSecure,
	}
}

// SignIn verifies the credentials and returns a session token. It refuses to
// even check the password once the caller has piled up 5 failed attempts for
// the same username in the last 15 minutes.
//
// The token is set as an HttpOnly cookie so the browser sends it back
// automatically and no script on the page can read it. It is also still
// returned in the response body for a non-browser API client (a script, an
// integration) that has nowhere to keep a cookie; the frontend itself no
// longer stores or reads that field.
func (h AuthHandler) SignIn(writer http.ResponseWriter, request *http.Request) {
	var payload signInRequest
	if err := decode(writer, request, &payload); err != nil {
		failure(writer, err)
		return
	}
	now := h.now()
	key := loginRateLimitKey(request, payload.Username)
	if h.limiter.blocked(key, now) {
		failure(writer, domain.ErrTooManyRequests)
		return
	}
	session, err := h.authenticate.Execute(request.Context(), payload.Username, payload.Password)
	if err != nil {
		h.limiter.recordFailure(key, now)
		failure(writer, err)
		return
	}
	h.limiter.reset(key)
	setSessionCookie(writer, session.Token, session.ExpiresAt, h.cookieSecure)
	respond(writer, http.StatusOK, sessionResponse{
		Token:     session.Token,
		ExpiresAt: formatTime(session.ExpiresAt),
		UserID:    session.UserID,
		Username:  session.Username,
		FullName:  session.FullName,
		Role:      string(session.Role),
	})
}

// SignOut clears the session cookie. A script cannot delete an HttpOnly
// cookie on its own, so the frontend calls this instead of just forgetting
// its local state.
func (h AuthHandler) SignOut(writer http.ResponseWriter, _ *http.Request) {
	clearSessionCookie(writer, h.cookieSecure)
	writer.WriteHeader(http.StatusNoContent)
}
