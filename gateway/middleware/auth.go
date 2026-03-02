package middleware

import (
	"net/http"
	"strings"
)

// Auth returns middleware that validates JWT tokens from the Authorization header.
//
// TODO: Implement actual JWT validation. Current implementation is a
// pass-through skeleton that extracts the Bearer token but does not verify it.
// Production requirements:
//   - Validate JWT signature against known keys (JWKS endpoint or shared secret)
//   - Check exp, iat, nbf claims
//   - Verify issuer and audience
//   - Inject authenticated user/claims into request context
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// TODO: Enforce authentication once JWT validation is implemented.
		// For now, allow unauthenticated requests to pass through.
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract Bearer token.
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		_ = strings.TrimPrefix(authHeader, "Bearer ")
		// TODO: Validate the token and inject claims into r.Context().
		// token := strings.TrimPrefix(authHeader, "Bearer ")
		// claims, err := validateJWT(token)
		// if err != nil {
		//     http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		//     return
		// }
		// ctx := context.WithValue(r.Context(), claimsKey, claims)
		// r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
