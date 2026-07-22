// Package rbac centralises permission codes and the role→permission matrix from
// the implementation spec (§8.7.7). Handlers gate on permission codes, never on
// raw role checks, so the matrix stays in one place.
package rbac

import "github.com/inwardclub/server/internal/platform/authn"

// Permission is a stable capability code.
type Permission string

const (
	// Global templates (catalog / coupon / activity).
	PermGlobalTemplateWrite Permission = "global_template:write"
	PermGlobalTemplateRead  Permission = "global_template:read"

	// Store-owned catalog / coupon / activity resources.
	PermStoreResourceWrite Permission = "store_resource:write"
	PermStoreResourceRead  Permission = "store_resource:read"

	// Orders, payments, refunds.
	PermOrderRead     Permission = "order:read"
	PermOrderWrite    Permission = "order:write"
	PermRefundApprove Permission = "refund:approve"
	PermRefundRequest Permission = "refund:request"

	// Offline aggregated collection.
	PermOfflineCollectionManage Permission = "offline_collection:manage"
	PermOfflineCollectionConfig Permission = "offline_collection:config"

	// Coupon / ticket verification.
	PermVerify Permission = "verify"

	// Members and wallet.
	PermMemberRead          Permission = "member:read"
	PermWalletAdjustApprove Permission = "wallet_adjust:approve"
	PermWalletAdjustRequest Permission = "wallet_adjust:request"

	// Rule definitions.
	PermRulePublish Permission = "rule:publish"
	PermRuleDraft   Permission = "rule:draft"

	// Audit logs.
	PermAuditRead Permission = "audit:read"

	// Account / store administration.
	PermAdminAccountManage Permission = "admin_account:manage"
	PermStoreManage        Permission = "store:manage"
	PermStaffManage        Permission = "staff:manage"
)

// grants encodes the minimal permission matrix. super_admin is global; the
// store roles are already store-scoped by the store middleware, so these grants
// only decide capability, not data range.
var grants = map[authn.Role]map[Permission]bool{
	authn.RoleSuperAdmin: {
		PermGlobalTemplateWrite: true, PermGlobalTemplateRead: true,
		PermStoreResourceWrite: true, PermStoreResourceRead: true,
		PermOrderRead: true, PermOrderWrite: true,
		PermRefundApprove: true, PermRefundRequest: true,
		PermOfflineCollectionManage: true, PermOfflineCollectionConfig: true,
		PermVerify:     true,
		PermMemberRead: true, PermWalletAdjustApprove: true, PermWalletAdjustRequest: true,
		PermRulePublish: true, PermRuleDraft: true,
		PermAuditRead:          true,
		PermAdminAccountManage: true, PermStoreManage: true, PermStaffManage: true,
	},
	authn.RoleStoreAdmin: {
		PermGlobalTemplateRead: true,
		PermStoreResourceWrite: true, PermStoreResourceRead: true,
		PermOrderRead: true, PermOrderWrite: true,
		PermRefundRequest:           true,
		PermOfflineCollectionManage: true,
		PermVerify:                  true,
		PermMemberRead:              true, PermWalletAdjustRequest: true,
		PermRuleDraft:   true,
		PermAuditRead:   true,
		PermStaffManage: true,
	},
	authn.RoleCashier: {
		PermStoreResourceRead:       true,
		PermOrderRead:               true,
		PermRefundRequest:           true,
		PermOfflineCollectionManage: true,
		PermVerify:                  true,
		PermMemberRead:              true,
	},
	authn.RoleStaff: {
		PermStoreResourceRead: true,
		PermVerify:            true,
	},
	authn.RoleMember: {},
}

// Allowed reports whether a role holds a permission.
func Allowed(role authn.Role, perm Permission) bool {
	perms, ok := grants[role]
	if !ok {
		return false
	}
	return perms[perm]
}
