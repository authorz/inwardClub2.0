<script setup lang="ts">
/**
 * 资产图片展示。门店后台只提交 assetId，这里仅用于展示。
 * 若给出 assetUrl 直接用；否则用 assetId 拼接资产公共域名。
 */
import { computed } from 'vue'
import { NImage } from 'naive-ui'
import { appConfig } from '@/config'

const props = withDefaults(
  defineProps<{
    url?: string | null
    assetId?: number | string | null
    size?: number
    round?: boolean
  }>(),
  { size: 40, round: false },
)

const src = computed(() => {
  if (props.url) return props.url
  if (props.assetId != null && appConfig.assetPublicDomain) {
    return `${appConfig.assetPublicDomain}/${props.assetId}`
  }
  return ''
})
</script>

<template>
  <NImage
    v-if="src"
    :src="src"
    :width="size"
    :height="size"
    object-fit="cover"
    :style="{ borderRadius: round ? '50%' : 'var(--ic-radius-sm)' }"
  />
  <span
    v-else
    class="asset-fallback"
    :style="{ width: `${size}px`, height: `${size}px`, borderRadius: round ? '50%' : '4px' }"
  />
</template>

<style scoped>
.asset-fallback {
  display: inline-block;
  background: var(--ic-color-surface-muted);
  border: var(--ic-divider);
}
</style>
