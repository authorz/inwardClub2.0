<script setup lang="ts">
/**
 * 图片直传控件（配合 AssetImage 预览）。
 *
 * 选文件后直传七牛，成功即回传 objectKey/path（v-model:path），不再手填资产 ID。
 * 写入业务字段的是对象路径 objectKey，非 assetId、非完整域名 URL；预览用解析后的公开地址。
 */
import { ref } from 'vue'
import { NButton, NSpace, NUpload } from 'naive-ui'
import type { UploadCustomRequestOptions } from 'naive-ui'
import AssetImage from '@/components/AssetImage.vue'
import { assetService } from '@/api/services/asset'
import { toastError } from '@/utils/feedback'

const props = withDefaults(
  defineProps<{
    /** 上传用途（对齐服务端 asset purpose，如 vip_banner） */
    purpose: string
    /** 已绑定的对象路径 objectKey（v-model:path） */
    path?: string | null
    /** 使用 assets 外键的业务表通过 v-model:asset-id 接收上传后的资产 ID。 */
    assetId?: string | number | null
    /** 编辑态已有图片的展示地址（服务端返回 bannerUrl，仅预览） */
    previewUrl?: string | null
    /** 上传成功后自动回填的公开访问地址。 */
    publicUrl?: string | null
    width?: number
    height?: number
  }>(),
  { path: null, assetId: null, previewUrl: null, publicUrl: null, width: 160, height: 80 },
)

const emit = defineEmits<{
  'update:path': [value: string | null]
  'update:assetId': [value: string | null]
  'update:publicUrl': [value: string]
}>()

const uploading = ref(false)
/** 本次上传成功后的解析地址；优先于 previewUrl 展示。 */
const uploadedUrl = ref('')

async function customRequest({
  file,
  onFinish,
  onError,
}: UploadCustomRequestOptions): Promise<void> {
  const raw = file.file
  if (!raw) {
    onError()
    return
  }
  uploading.value = true
  try {
    const result = await assetService.uploadImage(props.purpose, raw)
    emit('update:path', result.objectKey)
    emit('update:assetId', String(result.assetId))
    emit('update:publicUrl', result.publicUrl)
    uploadedUrl.value = result.publicUrl
    onFinish()
  } catch (e) {
    toastError((e as { message?: string }).message ?? '图片上传失败')
    onError()
  } finally {
    uploading.value = false
  }
}
</script>

<template>
  <NSpace
    vertical
    style="width: 100%"
  >
    <AssetImage
      :src="publicUrl || uploadedUrl || previewUrl || null"
      :width="width"
      :height="height"
    />
    <NUpload
      :custom-request="customRequest"
      :show-file-list="false"
      :max="1"
      accept="image/png,image/jpeg,image/webp"
    >
      <NButton :loading="uploading">
        {{ path || assetId || publicUrl ? '重新上传' : '上传图片' }}
      </NButton>
    </NUpload>
  </NSpace>
</template>
