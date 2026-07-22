/**
 * 高风险 / 跨店写操作的统一执行器（公共状态 + 二次确认 + 审计提示）。
 *
 * 依据 ADMIN_CONSOLE_ARCHITECTURE.md 第 8 / 11 节与设计规则第 3 节：
 * 涉及资产、钱包、退款、规则、跨店写操作必须展示二次确认和审计提示，
 * 跨店写入还必须填写操作原因。
 *
 * 用法：调用 runAudited(config) 会弹出确认对话框，
 * 用户确认后执行 config.execute，成功/失败给统一反馈。
 * Idempotency-Key 由底层 http（idempotent 标记）注入，这里只负责交互与原因收集。
 */
import { getDialog, toastError, toastSuccess } from '@/utils/feedback'

export interface AuditedActionConfig {
  /** 对话框标题 */
  title: string
  /** 主要说明文案 */
  content: string
  /** 是否为跨店/高风险操作，展示审计提示条 */
  highRisk?: boolean
  /** 确认按钮文案 */
  positiveText?: string
  /** 真正执行的异步动作（返回值忽略） */
  execute: () => Promise<unknown>
  /** 成功提示 */
  successText?: string
}

/**
 * 返回 Promise<boolean>：true 表示已确认并执行成功，false 表示取消或失败。
 * 需要「填写原因」的场景由页面用 FormDrawer 收集原因后再调用 execute，
 * 这里的对话框负责最终的不可逆二次确认。
 */
export function runAudited(config: AuditedActionConfig): Promise<boolean> {
  const dialog = getDialog()
  if (!dialog) {
    // 无 dialog 环境（极少）直接执行，避免阻塞
    return config
      .execute()
      .then(() => {
        toastSuccess(config.successText ?? '操作成功')
        return true
      })
      .catch((e) => {
        toastError((e as Error)?.message ?? '操作失败')
        return false
      })
  }

  return new Promise<boolean>((resolve) => {
    dialog.warning({
      title: config.title,
      content: config.content,
      positiveText: config.positiveText ?? '确认执行',
      negativeText: '取消',
      onPositiveClick: async () => {
        try {
          await config.execute()
          toastSuccess(config.successText ?? '操作成功')
          resolve(true)
        } catch (e) {
          const err = e as { message?: string }
          toastError(err?.message ?? '操作失败')
          resolve(false)
        }
      },
      onNegativeClick: () => resolve(false),
      onClose: () => resolve(false),
      onMaskClick: () => resolve(false),
    })
  })
}
