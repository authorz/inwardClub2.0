package rbac

import (
	"testing"

	"github.com/inwardclub/server/internal/platform/authn"
)

func TestSuperAdminHasGlobalTemplateWrite(t *testing.T) {
	if !Allowed(authn.RoleSuperAdmin, PermGlobalTemplateWrite) {
		t.Fatal("super_admin must write global templates")
	}
}

func TestStoreAdminCannotWriteGlobalTemplates(t *testing.T) {
	if Allowed(authn.RoleStoreAdmin, PermGlobalTemplateWrite) {
		t.Fatal("store_admin must not write global templates")
	}
	if !Allowed(authn.RoleStoreAdmin, PermGlobalTemplateRead) {
		t.Fatal("store_admin should read published global templates")
	}
}

func TestCashierCannotWriteStoreResources(t *testing.T) {
	if Allowed(authn.RoleCashier, PermStoreResourceWrite) {
		t.Fatal("cashier must not write store resources")
	}
	if !Allowed(authn.RoleCashier, PermVerify) {
		t.Fatal("cashier should be able to verify")
	}
}

func TestStaffOnlyVerifies(t *testing.T) {
	if Allowed(authn.RoleStaff, PermOrderWrite) {
		t.Fatal("staff must not write orders")
	}
	if !Allowed(authn.RoleStaff, PermVerify) {
		t.Fatal("staff should verify")
	}
}

func TestMemberHasNoBackOfficePermissions(t *testing.T) {
	if Allowed(authn.RoleMember, PermOrderRead) {
		t.Fatal("member has no back-office permissions")
	}
}
