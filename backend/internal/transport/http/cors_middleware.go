package http

import "net/http"

// corsMiddleware answers cross origin requests for exactly one configured
// origin. A wildcard is never sent: the interface origin is known and comes
// from the environment.
func corsMiddleware(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			origin := request.Header.Get("Origin")
			if origin != "" && origin == allowedOrigin {
				header := writer.Header()
				header.Set("Access-Control-Allow-Origin", allowedOrigin)
				header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				// Credentials are allowed so the browser attaches the
				// HttpOnly session cookie to a cross-origin request (the SPA
				// is served from its own origin in every deployment, but
				// local development runs the Vite dev server on a different
				// port than the API). This is only safe because the origin
				// above is never a wildcard.
				header.Set("Access-Control-Allow-Credentials", "true")
				header.Set("Access-Control-Max-Age", "600")
				header.Add("Vary", "Origin")
			}
			if request.Method == http.MethodOptions {
				writer.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

// chain applies middleware in the order it is declared, so the first one in
// the list is the outermost.
func chain(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for index := len(middleware) - 1; index >= 0; index-- {
		handler = middleware[index](handler)
	}
	return handler
}
