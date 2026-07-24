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

// registerTicketTTL bounds how long a first-time user has to complete the
// profile form after WeChat login before the ticket must be re-obtained.
const registerTicketTTL = 10 * time.Minute

// RegisterTicketClaims is the register ticket payload: the openID (JWT subject)
// plus an optional phone number. The phone is embedded once the user authorizes
// it (via the phone-mask endpoint) so the single-use WeChat phone code is never
// decrypted twice — registration reads the phone straight from the ticket.
type RegisterTicketClaims struct {
	Phone string `json:"phone,omitempty"`
	jwt.RegisteredClaims
}

// IssueRegisterTicket mints a short-lived ticket that authorizes a first-time
// WeChat user to complete registration. It carries the openID (as the JWT
// subject) and, once authorized, the phone number; no member exists yet, so it
// is NOT an access token and authorizes nothing but the register endpoint. Pass
// an empty phone at login time; re-issue with the phone after it is authorized.
func (m *Manager) IssueRegisterTicket(openID, phone string) (string, error) {
	now := m.now().UTC()
	claims := RegisterTicketClaims{
		Phone: phone,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   openID,
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{string(AudienceRegister)},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(registerTicketTTL)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.signingKey)
	if err != nil {
		return "", fmt.Errorf("sign register ticket: %w", err)
	}
	return signed, nil
}

// ParseRegisterTicket verifies a register ticket and returns its openID and the
// embedded phone (empty if the phone has not been authorized yet).
func (m *Manager) ParseRegisterTicket(ticket string) (openID, phone string, err error) {
	claims := &RegisterTicketClaims{}
	_, err = jwt.ParseWithClaims(ticket, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.signingKey, nil
	}, jwt.WithAudience(string(AudienceRegister)), jwt.WithIssuer(m.issuer))
	if err != nil {
		return "", "", err
	}
	return claims.Subject, claims.Phone, nil
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
