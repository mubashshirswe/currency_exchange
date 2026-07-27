package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mubashshir3767/currencyExchange/internal/env"
	"github.com/mubashshir3767/currencyExchange/internal/store"
)

type contextkey string

const UserKey contextkey = "UserID"

// Token types live in the "typ" claim so a refresh token can never be used as
// an access token, or the other way around.
const (
	accessTokenType  = "access"
	refreshTokenType = "refresh"
)

const (
	defaultAccessTTL  = 24 * time.Hour
	defaultRefreshTTL = 90 * 24 * time.Hour
)

var (
	errEmptyToken   = errors.New("token is empty")
	errNoExpiry     = errors.New("token has no expiry claim")
	errWrongType    = errors.New("unexpected token type")
	errMissingClaim = errors.New("token is missing the user id claim")
)

// tokenConfig is resolved from the environment once: the values cannot change
// while the process runs and the auth middleware reads them on every request.
type tokenConfig struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

var (
	tokenCfg     tokenConfig
	tokenCfgOnce sync.Once

	// A parser is safe for concurrent use, so one instance serves all requests.
	tokenParser = jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
)

func tokens() *tokenConfig {
	tokenCfgOnce.Do(func() {
		tokenCfg = tokenConfig{
			secret:     []byte(env.GetString("JWTSECRET", "secret")),
			accessTTL:  envDuration("JWTExpirationInSeconds", defaultAccessTTL),
			refreshTTL: envDuration("JWTRefreshExpirationInSeconds", defaultRefreshTTL),
		}
	})
	return &tokenCfg
}

func envDuration(key string, fallback time.Duration) time.Duration {
	seconds := env.GetInt(key, int(fallback.Seconds()))
	if seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

// tokenClaims is the payload of every token this API issues. UserID stays a
// string and ExpiredAt mirrors the standard "exp" claim because tokens issued
// by earlier versions of this API were shaped that way.
type tokenClaims struct {
	Type      string `json:"typ,omitempty"`
	UserID    string `json:"userID"`
	ExpiredAt int64  `json:"expiredAt,omitempty"`

	// Tenant ma'lumoti mijoz uchun (business => company => user). Server
	// avtorizatsiyada bularga tayanmaydi — DB'dagi user manba hisoblanadi.
	BusinessID int64 `json:"businessID,omitempty"`
	CompanyID  int64 `json:"companyID,omitempty"`
	Role       int64 `json:"role,omitempty"`

	jwt.RegisteredClaims
}

// Validate runs after the library's own checks. Tokens issued before "exp" was
// added carry only "expiredAt", so their expiry is enforced here.
func (c tokenClaims) Validate() error {
	if c.ExpiresAt != nil {
		return nil
	}
	if c.ExpiredAt == 0 {
		return errNoExpiry
	}
	if time.Now().Unix() >= c.ExpiredAt {
		return jwt.ErrTokenExpired
	}
	return nil
}

// tokenType reports the token type, defaulting to access for legacy tokens that
// predate the "typ" claim.
func (c tokenClaims) tokenType() string {
	if c.Type == "" {
		return accessTokenType
	}
	return c.Type
}

func (c tokenClaims) userID() (int64, error) {
	if c.UserID == "" {
		return 0, errMissingClaim
	}
	return strconv.ParseInt(c.UserID, 10, 64)
}

func signToken(cfg *tokenConfig, userID int64, tokenType string, ttl time.Duration) (string, error) {
	return signTenantToken(cfg, &store.User{ID: userID}, tokenType, ttl)
}

// signTenantToken — tokenga tenant (business/company/role) claim'larini ham yozadi.
func signTenantToken(cfg *tokenConfig, user *store.User, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := tokenClaims{
		Type:       tokenType,
		UserID:     strconv.FormatInt(user.ID, 10),
		ExpiredAt:  expiresAt.Unix(),
		BusinessID: user.BusinessId,
		CompanyID:  user.CompanyId,
		Role:       user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(cfg.secret)
}

// issueTokenPair mints a short lived access token plus the refresh token the
// client uses to renew it without asking for credentials again.
func issueTokenPair(userID int64) (accessToken string, refreshToken string, err error) {
	return issueUserTokenPair(&store.User{ID: userID})
}

// issueUserTokenPair — tenant claim'lari bilan token juftligi.
func issueUserTokenPair(user *store.User) (accessToken string, refreshToken string, err error) {
	cfg := tokens()

	accessToken, err = signTenantToken(cfg, user, accessTokenType, cfg.accessTTL)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = signTenantToken(cfg, user, refreshTokenType, cfg.refreshTTL)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func tokenKeyFunc(*jwt.Token) (any, error) {
	return tokens().secret, nil
}

// parseToken verifies the signature and expiry of a token and asserts it is of
// the expected type.
func parseToken(tokenString string, wantType string) (*tokenClaims, error) {
	if tokenString == "" {
		return nil, errEmptyToken
	}

	claims := &tokenClaims{}
	if _, err := tokenParser.ParseWithClaims(tokenString, claims, tokenKeyFunc); err != nil {
		return nil, err
	}

	if claims.tokenType() != wantType {
		return nil, errWrongType
	}

	return claims, nil
}

// userIDFromToken verifies an access token and returns the user it belongs to.
func userIDFromToken(tokenString string, wantType string) (int64, error) {
	claims, err := parseToken(tokenString, wantType)
	if err != nil {
		return 0, err
	}
	return claims.userID()
}

// GetTokenFromRequest extracts the bearer token from the Authorization header.
func GetTokenFromRequest(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}

	const prefix = "Bearer "
	if len(header) >= len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}

	return strings.TrimSpace(header)
}
