/**
 * 总后台固定角色（集中定义）。
 * 依据 docs/ADMIN_CONSOLE_ARCHITECTURE.md 第 5 节。
 * 第一期使用固定角色；服务端 /admin/auth/me 返回的 permissions 才是权限真值来源，
 * 角色仅用于展示与降级默认权限。
 */
export const ADMIN_ROLE = {
  SUPER_ADMIN: 'super_admin',
  FINANCE_ADMIN: 'finance_admin',
  OPS_ADMIN: 'ops_admin',
  AUDIT_ADMIN: 'audit_admin',
  SUPPORT_ADMIN: 'support_admin',
} as const

export type AdminRole = (typeof ADMIN_ROLE)[keyof typeof ADMIN_ROLE]

export const ADMIN_ROLE_LABELS: Record<AdminRole, string> = {
  [ADMIN_ROLE.SUPER_ADMIN]: '超级管理员',
  [ADMIN_ROLE.FINANCE_ADMIN]: '财务管理员',
  [ADMIN_ROLE.OPS_ADMIN]: '运营管理员',
  [ADMIN_ROLE.AUDIT_ADMIN]: '审计管理员',
  [ADMIN_ROLE.SUPPORT_ADMIN]: '客服管理员',
}

/** 总后台 token 期望的 audience（防止门店后台登录态混用） */
export const EXPECTED_AUDIENCE = 'admin'

/**
 * 总后台侧允许的 subject_type 允许集。
 * 后端为总后台座席签发的 subject_type 为 super_admin（见 server authn.SubjectSuperAdmin，
 * middleware 亦强制 aud=admin 必须 super_admin）。此处采用允许集：
 *  - 仅接受总部身份（至少 super_admin），后端如新增总部 subject_type 在此追加；
 *  - 门店（store_admin / cashier）与小程序（member / staff）身份不在集合内，一律拒绝。
 */
export const ADMIN_SUBJECT_TYPES = ['super_admin'] as const

/** subject_type 是否属于总后台允许集 */
export function isAdminSubjectType(subjectType: string): boolean {
  return (ADMIN_SUBJECT_TYPES as readonly string[]).includes(subjectType)
}
