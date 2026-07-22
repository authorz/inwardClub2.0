package authn

import (
	"testing"
	"time"
)

func newTestManager() *Manager {
	return NewManager("test-signing-key", "inwardclub", time.Hour, 24*time.Hour)
}

func TestIssueAndParseRoundTrip(t *testing.T) {
	m := newTestManager()
	pair, err := m.Issue(Identity{
		SubjectID:    42,
		SubjectType:  SubjectStoreAdmin,
		Role:         RoleStoreAdmin,
		Audience:     AudienceStore,
		StoreID:      7,
		TokenVersion: 3,
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := m.Parse(pair.AccessToken, AudienceStore)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.SubjectID() != 42 || claims.StoreID != 7 || claims.TokenVersion != 3 {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.Kind != TokenAccess {
		t.Fatalf("expected access token, got %s", claims.Kind)
	}
}

func TestAudienceIsolation(t *testing.T) {
	m := newTestManager()
	// A store token must be rejected when parsed for the admin audience.
	pair, err := m.Issue(Identity{
		SubjectID: 1, SubjectType: SubjectStoreAdmin, Role: RoleStoreAdmin,
		Audience: AudienceStore, StoreID: 1,
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := m.Parse(pair.AccessToken, AudienceAdmin); err == nil {
		t.Fatal("expected admin audience to reject store token")
	}
	if _, err := m.Parse(pair.AccessToken, AudienceMini); err == nil {
		t.Fatal("expected mini audience to reject store token")
	}
}

func TestWrongSigningKeyRejected(t *testing.T) {
	m := newTestManager()
	pair, _ := m.Issue(Identity{SubjectID: 1, SubjectType: SubjectMember, Role: RoleMember, Audience: AudienceMini})

	other := NewManager("different-key", "inwardclub", time.Hour, 24*time.Hour)
	if _, err := other.Parse(pair.AccessToken, AudienceMini); err == nil {
		t.Fatal("expected token signed with different key to be rejected")
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	m := NewManager("k", "inwardclub", -time.Minute, time.Hour) // already expired access
	pair, _ := m.Issue(Identity{SubjectID: 1, SubjectType: SubjectMember, Role: RoleMember, Audience: AudienceMini})
	if _, err := m.Parse(pair.AccessToken, AudienceMini); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}
