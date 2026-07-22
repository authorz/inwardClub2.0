/**
 * 全局反馈能力（message / dialog / notification）的单一实例。
 *
 * 使用 Naive UI 的 discrete API，使 http 拦截器、composable 等非组件上下文
 * 也能统一弹出提示，避免各页面重复注入 useMessage/useDialog。
 */

import { createDiscreteApi, type ConfigProviderProps } from 'naive-ui'
import { computed } from 'vue'
import { themeOverrides } from '@/theme'

const configProviderProps = computed<ConfigProviderProps>(() => ({
  themeOverrides,
}))

const { message, dialog, notification } = createDiscreteApi(
  ['message', 'dialog', 'notification'],
  { configProviderProps },
)

export const feedback = { message, dialog, notification }
