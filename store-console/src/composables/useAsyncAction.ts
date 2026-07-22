/**
 * 异步写操作编排：统一 loading、成功/失败提示、可选二次确认。
 *
 * 所有按钮触发的写操作（状态流转、核销、审核、退款、库存调整等）统一走这里，
 * 避免每处各写 try/catch + message + loading。
 */

import { ref } from 'vue'
import { ApiError } from '@/api/error'
import { feedback } from '@/utils/feedback'
import { confirm, type ConfirmOptions } from './useConfirm'

export interface RunOptions<T> {
  /** 执行前的二次确认；不传则不确认。 */
  confirm?: ConfirmOptions
  /** 成功提示文案；不传则不提示。 */
  successMessage?: string
  /** 成功回调（如刷新列表）。 */
  onSuccess?: (result: T) => void
}

export function useAsyncAction() {
  const running = ref(false)

  async function run<T>(action: () => Promise<T>, options: RunOptions<T> = {}): Promise<T | null> {
    if (running.value) return null
    if (options.confirm) {
      const ok = await confirm(options.confirm)
      if (!ok) return null
    }
    running.value = true
    try {
      const result = await action()
      if (options.successMessage) feedback.message.success(options.successMessage)
      options.onSuccess?.(result)
      return result
    } catch (err) {
      const message = err instanceof ApiError ? err.message : '操作失败'
      feedback.message.error(message)
      return null
    } finally {
      running.value = false
    }
  }

  return { running, run }
}
