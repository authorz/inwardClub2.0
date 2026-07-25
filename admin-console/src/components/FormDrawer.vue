<script setup lang="ts">
/**
 * 表单弹窗（公共组件）。
 * 统一新增 / 编辑表单的容器：标题、内容插槽、底部保存/取消、提交 loading。
 * 高风险表单可通过 highRisk 展示审计提示条。
 * 居中弹出（NModal preset="card"）；对外接口（props/emits/slots）保持不变，
 * 页面调用方无需改动。组件名与文件名沿用 FormDrawer 以保持导入稳定。
 */
import { NButton, NModal, NSpace } from 'naive-ui'
import AuditRiskAlert from './AuditRiskAlert.vue'

const props = withDefaults(
  defineProps<{
    show: boolean
    title: string
    width?: number
    submitting?: boolean
    highRisk?: boolean
    crossStore?: boolean
    submitText?: string
  }>(),
  {
    width: 520,
    submitting: false,
    highRisk: false,
    crossStore: false,
    submitText: '保存',
  },
)

const emit = defineEmits<{
  'update:show': [value: boolean]
  submit: []
  cancel: []
}>()

function close(): void {
  emit('update:show', false)
  emit('cancel')
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="title"
    closable
    :mask-closable="false"
    :style="{ width: `${props.width}px`, maxWidth: '92vw' }"
    :content-style="{ maxHeight: '72vh', overflow: 'auto' }"
    @update:show="(v) => emit('update:show', v)"
  >
    <AuditRiskAlert
      v-if="highRisk"
      :cross-store="crossStore"
    />
    <slot />
    <template #footer>
      <NSpace justify="end">
        <NButton
          :disabled="submitting"
          @click="close"
        >
          取消
        </NButton>
        <NButton
          type="primary"
          :loading="submitting"
          @click="emit('submit')"
        >
          {{ submitText }}
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>
