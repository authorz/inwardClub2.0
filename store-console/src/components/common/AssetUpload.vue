<script setup lang="ts">
import { ref } from 'vue'
import { NButton, NSpace, NUpload } from 'naive-ui'
import type { UploadCustomRequestOptions } from 'naive-ui'
import AssetImage from './AssetImage.vue'
import { assetService } from '@/api/services/assets'
import { feedback } from '@/utils/feedback'

const props = withDefaults(defineProps<{
  purpose: string
  assetId?: string | number | null
  previewUrl?: string | null
  width?: number
  height?: number
  compact?: boolean
}>(), { assetId: null, previewUrl: null, width: 160, height: 80, compact: false })

const emit = defineEmits<{
  'update:assetId': [value: string | null]
  'update:previewUrl': [value: string]
}>()
const uploading = ref(false)

async function upload({ file, onFinish, onError }: UploadCustomRequestOptions): Promise<void> {
  if (!file.file) return onError()
  uploading.value = true
  try {
    const result = await assetService.uploadImage(props.purpose, file.file)
    emit('update:assetId', String(result.assetId))
    emit('update:previewUrl', result.publicUrl)
    onFinish()
  } catch (error) {
    feedback.message.error((error as { message?: string }).message ?? '图片上传失败')
    onError()
  } finally {
    uploading.value = false
  }
}
</script>

<template>
  <NSpace
    :vertical="!compact"
    :align="compact ? 'center' : undefined"
    style="width: 100%"
  >
    <AssetImage
      :src="previewUrl || null"
      :width="width"
      :height="height"
    />
    <NUpload
      :custom-request="upload"
      :show-file-list="false"
      :max="1"
      accept="image/png,image/jpeg,image/webp"
    >
      <NButton :loading="uploading">
        {{ assetId ? '重新上传' : '上传图片' }}
      </NButton>
    </NUpload>
  </NSpace>
</template>
