<script setup lang="ts">
/**
 * 资产图片展示。门店后台只提交 assetId，这里仅用于展示。
 * 若给出 src/url 直接用；否则用 assetId 拼接资产公共域名。
 */
import { computed } from 'vue'
import { NImage } from 'naive-ui'
import { appConfig } from '@/config'

const props = withDefaults(
  defineProps<{
    src?: string | null
    url?: string | null
    assetId?: number | string | null
    width?: number
    height?: number
    size?: number
    round?: boolean
  }>(),
  { src: null, url: null, assetId: null, width: undefined, height: undefined, size: 40, round: false },
)

const src = computed(() => {
  if (props.src) return props.src
  if (props.url) return props.url
  if (props.assetId != null && appConfig.assetPublicDomain) {
    return `${appConfig.assetPublicDomain}/${props.assetId}`
  }
  return ''
})
const imageWidth = computed(() => props.width ?? props.size)
const imageHeight = computed(() => props.height ?? props.size)
</script>

<template>
  <NImage
    v-if="src"
    :src="src"
    :width="imageWidth"
    :height="imageHeight"
    object-fit="cover"
    :style="{ borderRadius: round ? '50%' : 'var(--ic-radius-sm)' }"
  />
  <span
    v-else
    class="asset-fallback"
    :style="{ width: `${imageWidth}px`, height: `${imageHeight}px`, borderRadius: round ? '50%' : '4px' }"
  />
</template>

<style scoped>
.asset-fallback {
  display: inline-block;
  background: var(--ic-color-surface-muted);
  border: var(--ic-divider);
}
</style>
