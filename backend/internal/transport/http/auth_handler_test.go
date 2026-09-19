package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"workshop/internal/domain"
	"workshop/internal/usecase"
)

type fakeUserStore struct {
	user map[string]domain.User
}

func newFakeUserStore(t *testing.T, username, password string) *fakeUserStore {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hashing the fixture password failed: %v", err)
	}
	user, err := domain.NewUser("user-1", username, string(hash), "Administrador del taller", domain.RoleAdministrator, testMoment)
	if err != nil {
		t.Fatalf("building the user fixture failed: %v", err)
	}
	return &fakeUserStore{user: map[string]domain.User{username: user}}
}

func (f *fakeUserStore) FindByUsername(_ context.Context, username string) (domain.User, error) {
	found, ok := f.user[username]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return found, nil
}

func (f *fakeUserStore) FindByID(_ context.Context, id string) (domain.User, error) {
	for _, item := range f.user {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func signIn(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	handler := NewAuthHandler(usecase.NewAuthenticateUser(
		newFakeUserStore(t, "admin", "Admin2026"), testIssuer(), testClock(),
	), testClock(), true)
	request := httptest.NewRequest(http.MethodPost, "/api/session", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.SignIn(recorder, request)
	return recorder
}

func TestSignInReturnsATokenForValidCredentials(t *testing.T) {
	recorder := signIn(t, `{"username":"admin","password":"Admin2026"}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("valid credentials must answer 200, got %d body %s", recorder.Code, recorder.Body.String())
	}
	var payload sessionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("the response must be a session payload: %v", err)
	}
	if payload.Token == "" || payload.Role != string(domain.RoleAdministrator) {
		t.Fatalf("the session must carry a token and the role, got %+v", payload)
	}
	if _, err := testIssuer().Verify(payload.Token, testMoment.Add(time.Minute)); err != nil {
		t.Fatalf("the issued token must verify: %v", err)
	}
}

func TestSignInRejectsAWrongPasswordInSpanishWithoutAToken(t *testing.T) {
	recorder := signIn(t, `{"username":"admin","password":"wrong"}`)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("a wrong password must answer 401, got %d", recorder.Code)
	}
	var payload errorPayload
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("the response must be an error payload: %v", err)
	}
	if payload.Message != "Usuario o contrasena incorrectos." {
		t.Fatalf("the message shown to the user must be in Spanish, got %q", payload.Message)
	}
	if strings.Contains(recorder.Body.String(), "token") {
		t.Fatal("a failed sign in must not return a token")
	}
}

func TestSignInRejectsAMalformedBody(t *testing.T) {
	recorder := signIn(t, "not-json")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("a malformed body must answer 400, got %d", recorder.Code)
	}
	var payload errorPayload
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("the response must be an error payload: %v", err)
	}
	if payload.Message == "" || strings.Contains(payload.Message, "json") {
		t.Fatalf("the message must be sanitized and in Spanish, got %q", payload.Message)
	}
}

func TestSignInNeverReturnsThePasswordHash(t *testing.T) {
	recorder := signIn(t, `{"username":"admin","password":"Admin2026"}`)
	if strings.Contains(recorder.Body.String(), "$2a$") {
		t.Fatal("the session response must never carry the password hash")
	}
}

func TestSignInSetsAnHttpOnlySecureSessionCookie(t *testing.T) {
	recorder := signIn(t, `{"username":"admin","password":"Admin2026"}`)

	cookies := recorder.Result().Cookies()
	var session *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			session = cookie
		}
	}
	if session == nil {
		t.Fatal("a successful sign in must set the session cookie")
	}
	if !session.HttpOnly {
		t.Fatal("the session cookie must be HttpOnly so a script cannot read it")
	}
	if !session.Secure {
		t.Fatal("the session cookie must be Secure in this deployment")
	}
	if session.SameSite != http.SameSiteLaxMode {
		t.Fatalf("the session cookie must use SameSite=Lax, got %v", session.SameSite)
	}
	if session.Value == "" {
		t.Fatal("the session cookie must carry the session token")
	}
}

func TestSignInDoesNotSetTheCookieOnAFailedAttempt(t *testing.T) {
	recorder := signIn(t, `{"username":"admin","password":"wrong"}`)
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			t.Fatal("a failed sign in must not set a session cookie")
		}
	}
}

func TestSignOutClearsTheSessionCookie(t *testing.T) {
	handler := NewAuthHandler(usecase.NewAuthenticateUser(
		newFakeUserStore(t, "admin", "Admin2026"), testIssuer(), testClock(),
	), testClock(), true)
	request := httptest.NewRequest(http.MethodPost, "/api/session/sign-out", nil)
	recorder := httptest.NewRecorder()

	handler.SignOut(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("sign out must answer 204, got %d", recorder.Code)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName {
		t.Fatalf("sign out must clear the session cookie, got %+v", cookies)
	}
	if cookies[0].Value != "" || cookies[0].MaxAge >= 0 {
		t.Fatalf("the cleared cookie must have an empty value and a negative MaxAge, got %+v", cookies[0])
	}
}

func TestSignInBlocksAfterFiveFailedAttemptsForTheSameUsername(t *testing.T) {
	handler := NewAuthHandler(usecase.NewAuthenticateUser(
		newFakeUserStore(t, "admin", "Admin2026"), testIssuer(), testClock(),
	), testClock(), true)

	var last *httptest.ResponseRecorder
	for i := 0; i < 5; i++ {
		request := httptest.NewRequest(
			http.MethodPost, "/api/session", strings.NewReader(`{"username":"admin","password":"wrong"}`),
		)
		last = httptest.NewRecorder()
		handler.SignIn(last, request)
		if last.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d with a wrong password must answer 401, got %d", i+1, last.Code)
		}
	}

	sixthWrong := httptest.NewRequest(
		http.MethodPost, "/api/session", strings.NewReader(`{"username":"admin","password":"wrong"}`),
	)
	blocked := httptest.NewRecorder()
	handler.SignIn(blocked, sixthWrong)
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("the 6th failed attempt in the window must answer 429, got %d", blocked.Code)
	}

	sixthCorrect := httptest.NewRequest(
		http.MethodPost, "/api/session", strings.NewReader(`{"username":"admin","password":"Admin2026"}`),
	)
	blockedEvenWithRightPassword := httptest.NewRecorder()
	handler.SignIn(blockedEvenWithRightPassword, sixthCorrect)
	if blockedEvenWithRightPassword.Code != http.StatusTooManyRequests {
		t.Fatalf("a blocked key must be refused even with the right password, got %d", blockedEvenWithRightPassword.Code)
	}
}

func TestSignInDoesNotBlockADifferentUsernameFromTheSameCaller(t *testing.T) {
	handler := NewAuthHandler(usecase.NewAuthenticateUser(
		newFakeUserStore(t, "admin", "Admin2026"), testIssuer(), testClock(),
	), testClock(), true)

	for i := 0; i < 5; i++ {
		request := httptest.NewRequest(
			http.MethodPost, "/api/session", strings.NewReader(`{"username":"jperez","password":"wrong"}`),
		)
		recorder := httptest.NewRecorder()
		handler.SignIn(recorder, request)
	}

	request := httptest.NewRequest(
		http.MethodPost, "/api/session", strings.NewReader(`{"username":"admin","password":"Admin2026"}`),
	)
	recorder := httptest.NewRecorder()
	handler.SignIn(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("exhausting another username's attempts must not block this one, got %d", recorder.Code)
	}
}

func TestLoginRateLimiterForgetsAKeyOnceItHasNoRecentFailures(t *testing.T) {
	handler := NewAuthHandler(usecase.NewAuthenticateUser(
		newFakeUserStore(t, "admin", "Admin2026"), testIssuer(), testClock(),
	), testClock(), true)

	// A wave of successful, unrelated sign-ins must not accumulate an entry
	// per caller forever: blocked() runs on every attempt, and each one here
	// has zero recent failures, so none of them should leave a trace behind.
	for i := 0; i < 50; i++ {
		request := httptest.NewRequest(
			http.MethodPost, "/api/session", strings.NewReader(`{"username":"admin","password":"Admin2026"}`),
		)
		recorder := httptest.NewRecorder()
		handler.SignIn(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("attempt %d with the right password must answer 200, got %d", i+1, recorder.Code)
		}
	}

	if len(handler.limiter.failure) != 0 {
		t.Fatalf("the limiter must not keep an entry for a key with no recent failures, got %d entries", len(handler.limiter.failure))
	}
}
