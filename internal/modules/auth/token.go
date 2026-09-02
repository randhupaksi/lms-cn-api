package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"lms-cn-api/internal/authz"

	"github.com/golang-jwt/jwt/v5"
)

type accessClaims struct {
	Role               string `json:"role"`
	SessionID          string `json:"sid"`
	MustChangePassword bool   `json:"pwd"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
}

func NewTokenManager(secret, issuer string, accessTTL time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), issuer: issuer, accessTTL: accessTTL}
}

func (m *TokenManager) CreateAccessToken(principal authz.Principal, now time.Time) (string, error) {
	claims := accessClaims{
		Role: principal.Role, SessionID: principal.SessionID, MustChangePassword: principal.MustChangePassword,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: principal.UserID, Issuer: m.issuer,
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *TokenManager) VerifyAccessToken(rawToken string) (authz.Principal, error) {
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.Subject == "" || claims.SessionID == "" {
		return authz.Principal{}, errors.New("invalid access token")
	}
	return authz.Principal{UserID: claims.Subject, SessionID: claims.SessionID, Role: claims.Role, MustChangePassword: claims.MustChangePassword}, nil
}

func generateRefreshToken() (raw, hash string, err error) {
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(bytes)
	return raw, hashToken(raw), nil
}

func hashToken(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}
