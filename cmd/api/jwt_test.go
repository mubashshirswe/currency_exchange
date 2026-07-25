package main

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssueTokenPairRoundTrip(t *testing.T) {
	const userID = int64(42)

	accessToken, refreshToken, err := issueTokenPair(userID)
	if err != nil {
		t.Fatalf("issueTokenPair: %v", err)
	}

	gotAccess, err := userIDFromToken(accessToken, accessTokenType)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if gotAccess != userID {
		t.Errorf("access token user id = %d, want %d", gotAccess, userID)
	}

	gotRefresh, err := userIDFromToken(refreshToken, refreshTokenType)
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}
	if gotRefresh != userID {
		t.Errorf("refresh token user id = %d, want %d", gotRefresh, userID)
	}
}

func TestTokenTypesAreNotInterchangeable(t *testing.T) {
	accessToken, refreshToken, err := issueTokenPair(1)
	if err != nil {
		t.Fatalf("issueTokenPair: %v", err)
	}

	if _, err := userIDFromToken(refreshToken, accessTokenType); !errors.Is(err, errWrongType) {
		t.Errorf("refresh token accepted as access token, err = %v", err)
	}

	if _, err := userIDFromToken(accessToken, refreshTokenType); !errors.Is(err, errWrongType) {
		t.Errorf("access token accepted as refresh token, err = %v", err)
	}
}

func TestRefreshTokenOutlivesAccessToken(t *testing.T) {
	cfg := tokens()
	if cfg.refreshTTL <= cfg.accessTTL {
		t.Fatalf("refresh TTL %v must be longer than access TTL %v", cfg.refreshTTL, cfg.accessTTL)
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	expired, err := signToken(tokens(), 7, accessTokenType, -time.Minute)
	if err != nil {
		t.Fatalf("signToken: %v", err)
	}

	if _, err := userIDFromToken(expired, accessTokenType); !errors.Is(err, jwt.ErrTokenExpired) {
		t.Errorf("expired token accepted, err = %v", err)
	}
}

// Tokens minted before refresh tokens existed carry "expiredAt" but no "exp",
// and must keep working until they run out.
func TestLegacyTokenWithoutExpClaim(t *testing.T) {
	sign := func(expiredAt int64) string {
		t.Helper()
		claims := jwt.MapClaims{
			"userID":    strconv.Itoa(9),
			"expiredAt": expiredAt,
		}
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(tokens().secret)
		if err != nil {
			t.Fatalf("sign legacy token: %v", err)
		}
		return token
	}

	valid := sign(time.Now().Add(time.Hour).Unix())
	userID, err := userIDFromToken(valid, accessTokenType)
	if err != nil {
		t.Fatalf("valid legacy token rejected: %v", err)
	}
	if userID != 9 {
		t.Errorf("legacy token user id = %d, want 9", userID)
	}

	expired := sign(time.Now().Add(-time.Hour).Unix())
	if _, err := userIDFromToken(expired, accessTokenType); !errors.Is(err, jwt.ErrTokenExpired) {
		t.Errorf("expired legacy token accepted, err = %v", err)
	}
}

func TestTokenSignedWithAnotherSecretIsRejected(t *testing.T) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims{
		Type:   accessTokenType,
		UserID: "1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}).SignedString([]byte("some-other-secret"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if _, err := userIDFromToken(token, accessTokenType); !errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		t.Errorf("token with foreign signature accepted, err = %v", err)
	}
}

func TestNoneAlgorithmIsRejected(t *testing.T) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, tokenClaims{
		Type:   accessTokenType,
		UserID: "1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if _, err := userIDFromToken(token, accessTokenType); err == nil {
		t.Error("token signed with alg=none was accepted")
	}
}

func TestGetTokenFromRequest(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"no header", "", ""},
		{"bearer prefix", "Bearer abc.def.ghi", "abc.def.ghi"},
		{"lowercase prefix", "bearer abc.def.ghi", "abc.def.ghi"},
		{"raw token", "abc.def.ghi", "abc.def.ghi"},
		{"prefix only", "Bearer ", ""},
		{"surrounding spaces", "Bearer   abc.def.ghi  ", "abc.def.ghi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodGet, "/", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			if tt.header != "" {
				r.Header.Set("Authorization", tt.header)
			}

			if got := GetTokenFromRequest(r); got != tt.want {
				t.Errorf("GetTokenFromRequest() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEmptyTokenIsRejected(t *testing.T) {
	if _, err := userIDFromToken("", accessTokenType); !errors.Is(err, errEmptyToken) {
		t.Errorf("empty token error = %v, want %v", err, errEmptyToken)
	}
}

func BenchmarkParseToken(b *testing.B) {
	token, _, err := issueTokenPair(1)
	if err != nil {
		b.Fatalf("issueTokenPair: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := userIDFromToken(token, accessTokenType); err != nil {
			b.Fatalf("userIDFromToken: %v", err)
		}
	}
}
