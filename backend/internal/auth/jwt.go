package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid or expired token")

// Claims is the access-token payload. Subject holds the user id; StoreID and
// Role drive multi-tenant scoping and RBAC without another DB round-trip.
type Claims struct {
	StoreID string `json:"store_id"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

// IssueAccessToken signs a short-lived HS256 access token.
func IssueAccessToken(secret string, userID, storeID uuid.UUID, role string, ttl time.Duration, now time.Time) (string, error) {
	claims := Claims{
		StoreID: storeID.String(),
		Role:    role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.NewString(),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseAccessToken validates the signature and expiry and returns the claims.
func ParseAccessToken(secret, token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
