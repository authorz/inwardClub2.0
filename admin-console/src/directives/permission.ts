/**
 * v-permission 指令：无权限时移除元素。
 *
 * 用法：<button v-permission="'admin.refund.approve'">退款</button>
 * 适用于非按钮元素或不便用 PermissionButton 的场景。
 * 仅做前端显隐，最终授权由服务端强制。
 */
import type { Directive, DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'
import type { PermissionCode } from '@/constants/permissions'

function check(el: HTMLElement, binding: DirectiveBinding<PermissionCode | PermissionCode[]>): void {
  const permissionStore = usePermissionStore()
  const value = binding.value
  const codes = Array.isArray(value) ? value : [value]
  const allowed = codes.length === 0 || permissionStore.hasAny(codes)
  if (!allowed) {
    el.style.display = 'none'
  } else {
    // 恢复显示（响应式变化时）
    if (el.style.display === 'none') el.style.display = ''
  }
}

export const permissionDirective: Directive<HTMLElement, PermissionCode | PermissionCode[]> = {
  mounted: check,
  updated: check,
}
