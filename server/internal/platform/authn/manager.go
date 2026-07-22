package authn

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Manager issues and verifies JWTs with a shared HMAC signing key.
type Manager struct {
	signingKey []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

// Identity is the minimal input needed to mint a token pair.
type Identity struct {
	SubjectID    int64
	SubjectType  SubjectType
	Role         Role
	Audience     Audience
	StoreID      int64
	TokenVersion int64
}

// NewManager builds a token Manager.
func NewManager(signingKey, issuer string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		signingKey: []byte(signingKey),
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}
}

// TokenPair is the access + refresh result returned to clients.
type TokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	TokenType    string    `json:"tokenType"`
}

// Issue mints an access + refresh token pair for the identity.
func (m *Manager) Issue(id Identity) (TokenPair, error) {
	now := m.now().UTC()
	accessExp := now.Add(m.accessTTL)

	access, err := m.sign(id, TokenAccess, now, accessExp)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := m.sign(id, TokenRefresh, now, now.Add(m.refreshTTL))
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    accessExp,
		TokenType:    "Bearer",
	}, nil
}

func (m *Manager) sign(id Identity, kind TokenKind, now, exp time.Time) (string, error) {
	claims := Claims{
		SubjectType:  id.SubjectType,
		Role:         id.Role,
		StoreID:      id.StoreID,
		TokenVersion: id.TokenVersion,
		Kind:         kind,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(id.SubjectID, 10),
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{string(id.Audience)},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.signingKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Parse verifies the signature, expiry and audience of a token string.
func (m *Manager) Parse(tokenString string, expected Audience) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.signingKey, nil
	}, jwt.WithAudience(string(expected)), jwt.WithIssuer(m.issuer))
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func parseInt(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
