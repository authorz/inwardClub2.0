// Package authn owns JWT identity: audiences, subject types, roles, token
// issuance/verification and the auth middleware. The three consoles (mini,
// admin, store) use distinct audiences that must never be interchangeable.
package authn

import "github.com/golang-jwt/jwt/v5"

// Audience separates the three independent front-ends. A token minted for one
// audience is rejected by the others.
type Audience string

const (
	AudienceMini     Audience = "mini"     // WeChat mini program (member + staff)
	AudienceAdmin    Audience = "admin"    // headquarters console
	AudienceStore    Audience = "store"    // single-store console
	AudienceRegister Audience = "register" // pre-registration ticket (no member yet)
)

// SubjectType is the kind of identity behind the token.
type SubjectType string

const (
	SubjectMember     SubjectType = "member"
	SubjectStaff      SubjectType = "staff"
	SubjectStoreAdmin SubjectType = "store_admin"
	SubjectCashier    SubjectType = "cashier"
	SubjectSuperAdmin SubjectType = "super_admin"
)

// Role is the RBAC role carried in the token. Roles map 1:1 to subject types in
// v2 but are kept distinct so future sub-roles can be added without new subjects.
type Role string

const (
	RoleMember     Role = "member"
	RoleStaff      Role = "staff"
	RoleStoreAdmin Role = "store_admin"
	RoleCashier    Role = "cashier"
	RoleSuperAdmin Role = "super_admin"
)

// TokenKind distinguishes short-lived access tokens from refresh tokens.
type TokenKind string

const (
	TokenAccess  TokenKind = "access"
	TokenRefresh TokenKind = "refresh"
)

// Claims is the full JWT payload. StoreID is authoritative and server-derived;
// it must never be populated from a request parameter.
type Claims struct {
	SubjectType  SubjectType `json:"subject_type"`
	Role         Role        `json:"role"`
	StoreID      int64       `json:"store_id"`
	TokenVersion int64       `json:"token_version"`
	Kind         TokenKind   `json:"kind"`
	jwt.RegisteredClaims
}

// HasStore reports whether the token carries a concrete store scope.
func (c *Claims) HasStore() bool { return c.StoreID > 0 }

// SubjectID returns the numeric subject (member/account/staff id) or 0.
func (c *Claims) SubjectID() int64 { return parseInt(c.Subject) }
