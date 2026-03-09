package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

var publicPaths = map[string]struct{}{
	"/healthz":    {},
	"/readyz":     {},
	"/debug/vars": {},
}

// Claims carries validated JWT claims placed in request context.
type Claims struct {
	Subject string
	Expiry  int64
	Raw     map[string]any
}

type claimsContextKey struct{}

// Auth validates Bearer JWTs signed with HS256.
func Auth(secret string, next http.Handler) http.Handler {
	secretBytes := []byte(secret)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if len(secretBytes) == 0 {
			writeJSONError(w, http.StatusInternalServerError, "auth secret not configured")
			return
		}

		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			writeJSONError(w, http.StatusUnauthorized, "invalid authorization header")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
		if token == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		claims, err := validateHS256Token(token, secretBytes, time.Now().Unix())
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFromContext returns validated claims if present.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(Claims)
	return claims, ok
}

func isPublicPath(path string) bool {
	_, ok := publicPaths[path]
	return ok
}

func validateHS256Token(token string, secret []byte, nowUnix int64) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("token must have three parts")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, errors.New("invalid token header encoding")
	}

	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return Claims{}, errors.New("invalid token header")
	}
	if header.Alg != "HS256" {
		return Claims{}, errors.New("unsupported jwt alg")
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, errors.New("invalid signature encoding")
	}

	signed := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signed))
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return Claims{}, errors.New("signature mismatch")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, errors.New("invalid payload encoding")
	}

	rawClaims := make(map[string]any)
	if err := json.Unmarshal(payloadBytes, &rawClaims); err != nil {
		return Claims{}, errors.New("invalid payload json")
	}

	expiry, err := numericClaim(rawClaims, "exp")
	if err != nil {
		return Claims{}, err
	}
	if nowUnix >= expiry {
		return Claims{}, errors.New("token expired")
	}

	if nbf, ok, err := optionalNumericClaim(rawClaims, "nbf"); err != nil {
		return Claims{}, err
	} else if ok && nowUnix < nbf {
		return Claims{}, errors.New("token not active")
	}

	if iat, ok, err := optionalNumericClaim(rawClaims, "iat"); err != nil {
		return Claims{}, err
	} else if ok && iat > nowUnix+60 {
		return Claims{}, errors.New("token iat in future")
	}

	sub := ""
	if v, ok := rawClaims["sub"].(string); ok {
		sub = v
	}

	return Claims{
		Subject: sub,
		Expiry:  expiry,
		Raw:     rawClaims,
	}, nil
}

func numericClaim(claims map[string]any, key string) (int64, error) {
	v, ok := claims[key]
	if !ok {
		return 0, errors.New("missing claim: " + key)
	}
	return anyToInt64(v)
}

func optionalNumericClaim(claims map[string]any, key string) (int64, bool, error) {
	v, ok := claims[key]
	if !ok {
		return 0, false, nil
	}
	n, err := anyToInt64(v)
	if err != nil {
		return 0, false, err
	}
	return n, true, nil
}

func anyToInt64(v any) (int64, error) {
	switch t := v.(type) {
	case float64:
		return int64(t), nil
	case json.Number:
		return t.Int64()
	case int64:
		return t, nil
	case int32:
		return int64(t), nil
	case int:
		return int64(t), nil
	default:
		return 0, errors.New("invalid numeric claim")
	}
}
