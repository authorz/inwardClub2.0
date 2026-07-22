/**
 * 统一的二次确认封装。
 * 危险操作（退款、核销、批量下架、库存调整、发券、禁用）必须经过二次确认。
 */

import { feedback } from '@/utils/feedback'

export interface ConfirmOptions {
  title?: string
  content: string
  positiveText?: string
  negativeText?: string
  danger?: boolean
}

/** 返回一个 Promise，确认 resolve(true)，取消 resolve(false)。 */
export function confirm(options: ConfirmOptions): Promise<boolean> {
  return new Promise((resolve) => {
    feedback.dialog[options.danger ? 'warning' : 'info']({
      title: options.title ?? '操作确认',
      content: options.content,
      positiveText: options.positiveText ?? '确认',
      negativeText: options.negativeText ?? '取消',
      onPositiveClick: () => resolve(true),
      onNegativeClick: () => resolve(false),
      onClose: () => resolve(false),
      onMaskClick: () => resolve(false),
    })
  })
}
