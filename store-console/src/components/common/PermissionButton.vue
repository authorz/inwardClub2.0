<script setup lang="ts">
/**
 * 权限按钮：无权限时隐藏或禁用。
 * 前端收敛入口，最终权限由服务端强制。
 */
import { computed } from 'vue'
import { NButton, NTooltip } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import type { PermissionCode } from '@/constants/permissions'

const props = withDefaults(
  defineProps<{
    /** 所需权限码（命中任一即可）。为空则不做限制。 */
    permissions?: PermissionCode[]
    /** 无权限时的表现：隐藏或禁用。 */
    fallback?: 'hide' | 'disable'
    type?: 'default' | 'primary' | 'error' | 'warning' | 'info' | 'success'
    size?: 'tiny' | 'small' | 'medium' | 'large'
    loading?: boolean
    disabled?: boolean
    text?: boolean
  }>(),
  { fallback: 'hide', type: 'default', size: 'small', loading: false, disabled: false, text: false },
)

const emit = defineEmits<{ click: [] }>()
const auth = useAuthStore()

const allowed = computed(() => auth.hasPermission(props.permissions))
const visible = computed(() => allowed.value || props.fallback === 'disable')
const isDisabled = computed(() => props.disabled || (!allowed.value && props.fallback === 'disable'))
</script>

<template>
  <NTooltip
    v-if="visible && !allowed"
    trigger="hover"
  >
    <template #trigger>
      <NButton
        :type="type"
        :size="size"
        :disabled="isDisabled"
        :loading="loading"
        :text="text"
      >
        <slot />
      </NButton>
    </template>
    无操作权限
  </NTooltip>
  <NButton
    v-else-if="visible"
    :type="type"
    :size="size"
    :disabled="isDisabled"
    :loading="loading"
    :text="text"
    @click="emit('click')"
  >
    <slot />
  </NButton>
</template>
