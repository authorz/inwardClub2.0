<script setup lang="ts">
/**
 * 表单抽屉（公共组件）。
 * 统一新增 / 编辑表单的容器：标题、内容插槽、底部保存/取消、提交 loading。
 * 高风险表单可通过 highRisk 展示审计提示条。
 */
import { NButton, NDrawer, NDrawerContent, NSpace } from 'naive-ui'
import AuditRiskAlert from './AuditRiskAlert.vue'

withDefaults(
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
  <NDrawer
    :show="show"
    :width="width"
    :mask-closable="false"
    @update:show="(v) => emit('update:show', v)"
  >
    <NDrawerContent
      :title="title"
      closable
    >
      <AuditRiskAlert
        v-if="highRisk"
        :cross-store="crossStore"
      />
      <slot />
      <template #footer>
        <NSpace>
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
    </NDrawerContent>
  </NDrawer>
</template>
