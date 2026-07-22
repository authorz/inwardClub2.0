<script setup lang="ts">
/**
 * 权限按钮（公共组件）。
 * 无权限时默认隐藏（也可通过 disabledWhenNoPermission 改为禁用+提示）。
 * 前端仅做显隐，真正授权由服务端强制。
 */
import { computed } from 'vue'
import { NButton, NTooltip } from 'naive-ui'
import { usePermissionStore } from '@/stores/permission'
import type { PermissionCode } from '@/constants/permissions'

const props = withDefaults(
  defineProps<{
    permission?: PermissionCode
    type?: 'default' | 'primary' | 'error' | 'warning' | 'info' | 'success'
    size?: 'tiny' | 'small' | 'medium' | 'large'
    disabled?: boolean
    /** 无权限时保留按钮但禁用并提示，而不是隐藏 */
    disabledWhenNoPermission?: boolean
  }>(),
  {
    type: 'default',
    size: 'small',
    disabled: false,
    disabledWhenNoPermission: false,
  },
)

const emit = defineEmits<{ click: [] }>()
const permissionStore = usePermissionStore()

const allowed = computed(() => permissionStore.has(props.permission))
const visible = computed(() => allowed.value || props.disabledWhenNoPermission)
const isDisabled = computed(() => props.disabled || !allowed.value)
</script>

<template>
  <template v-if="visible">
    <NTooltip
      v-if="!allowed"
      trigger="hover"
    >
      <template #trigger>
        <NButton
          :type="type"
          :size="size"
          :disabled="isDisabled"
        >
          <slot />
        </NButton>
      </template>
      无权限：需要 {{ permission }}
    </NTooltip>
    <NButton
      v-else
      :type="type"
      :size="size"
      :disabled="isDisabled"
      @click="emit('click')"
    >
      <slot />
    </NButton>
  </template>
</template>
