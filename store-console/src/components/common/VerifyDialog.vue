<script setup lang="ts">
/**
 * 通用核销弹窗：输入/扫描核销码并确认核销。
 * 活动票核销、票券核销复用。实际扫码由门店设备输入，弹窗接受手动录入的码。
 */
import { ref, watch } from 'vue'
import { NButton, NInput, NModal } from 'naive-ui'
import { feedback } from '@/utils/feedback'

const props = withDefaults(
  defineProps<{
    show: boolean
    title?: string
    loading?: boolean
    placeholder?: string
    /** 预填的码（从列表行触发核销时带入）。 */
    presetCode?: string
  }>(),
  { title: '核销', loading: false, placeholder: '扫描或输入核销码' },
)

const emit = defineEmits<{
  'update:show': [value: boolean]
  confirm: [code: string]
}>()

const code = ref('')

watch(
  () => props.show,
  (open) => {
    if (open) code.value = props.presetCode ?? ''
  },
)

function onConfirm() {
  const value = code.value.trim()
  if (!value) {
    feedback.message.warning('请输入核销码')
    return
  }
  emit('confirm', value)
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="title"
    style="width: 420px"
    :mask-closable="!loading"
    @update:show="emit('update:show', $event)"
  >
    <div class="verify-body">
      <slot />
      <NInput
        v-model:value="code"
        :placeholder="placeholder"
        size="large"
        autofocus
        @keyup.enter="onConfirm"
      />
      <p class="verify-hint ic-muted">
        核销为高风险操作，确认后不可撤销。
      </p>
    </div>
    <template #footer>
      <div class="verify-footer">
        <NButton
          :disabled="loading"
          @click="emit('update:show', false)"
        >
          取消
        </NButton>
        <NButton
          type="primary"
          :loading="loading"
          @click="onConfirm"
        >
          确认核销
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.verify-body {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-3);
}
.verify-hint {
  font-size: var(--ic-font-xs);
  margin: 0;
}
.verify-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--ic-space-2);
}
</style>
