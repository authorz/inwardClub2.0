/**
 * 全局反馈句柄（message / dialog / notification）。
 *
 * Naive UI 的 message/dialog API 只能在 setup 内通过 useMessage() 获取。
 * 为了让 http client、pinia store 等非组件环境也能统一弹提示，
 * 在应用根组件里通过 registerFeedback() 注册一次，之后各处复用同一实例。
 *
 * 这样保证「错误处理 / loading 提示 / 二次确认」都走统一实现，而不是每处各写一套。
 */
import type { DialogApiInjection } from 'naive-ui/es/dialog/src/DialogProvider'
import type { MessageApiInjection } from 'naive-ui/es/message/src/MessageProvider'

interface FeedbackHandles {
  message: MessageApiInjection | null
  dialog: DialogApiInjection | null
}

const handles: FeedbackHandles = {
  message: null,
  dialog: null,
}

export function registerFeedback(message: MessageApiInjection, dialog: DialogApiInjection): void {
  handles.message = message
  handles.dialog = dialog
}

export function getMessage(): MessageApiInjection | null {
  return handles.message
}

export function getDialog(): DialogApiInjection | null {
  return handles.dialog
}

/** 便捷错误提示：有 message 用 message，否则退化到 console */
export function toastError(text: string): void {
  if (handles.message) handles.message.error(text)
  else console.error(text)
}

export function toastSuccess(text: string): void {
  if (handles.message) handles.message.success(text)
}
