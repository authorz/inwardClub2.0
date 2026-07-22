<script setup lang="ts">
/**
 * 资产图片展示（公共组件）。
 * 总后台只处理 assetId，不接受任意 URL。
 * 展示时用 VITE_ASSET_PUBLIC_DOMAIN 拼出可访问地址；未配置或无 assetId 时显示占位。
 */
import { computed } from 'vue'
import { NImage } from 'naive-ui'

const props = withDefaults(
  defineProps<{
    assetId?: string | null
    /** 已解析的图片地址；提供时优先于 assetId（用于服务端直接返回 URL 的场景，如 VIP 海报 bannerUrl） */
    src?: string | null
    width?: number
    height?: number
  }>(),
  { width: 48, height: 48 },
)

const publicDomain = import.meta.env.VITE_ASSET_PUBLIC_DOMAIN || ''

const resolvedSrc = computed(() => {
  if (props.src) return props.src
  if (!props.assetId) return ''
  if (!publicDomain) return ''
  return `${publicDomain.replace(/\/$/, '')}/${props.assetId}`
})
</script>

<template>
  <NImage
    v-if="resolvedSrc"
    :src="resolvedSrc"
    :width="width"
    :height="height"
    object-fit="cover"
    class="asset-image"
  />
  <div
    v-else
    class="asset-image asset-image--placeholder"
    :style="{ width: width + 'px', height: height + 'px' }"
  >
    {{ assetId ? 'asset' : '—' }}
  </div>
</template>

<style scoped>
.asset-image {
  border-radius: var(--ic-radius-sm);
  border: 1px solid var(--ic-color-border);
}
.asset-image--placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
  background: var(--ic-color-surface-muted);
}
</style>
