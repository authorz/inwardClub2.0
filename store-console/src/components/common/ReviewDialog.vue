<script setup lang="ts">
/**
 * 通用审核弹窗（通过 / 驳回 + 备注）。
 * 积分审核等审批流复用；驳回强制要求填写原因。
 */
import { ref, watch } from 'vue'
import { NButton, NInput, NModal } from 'naive-ui'
import { feedback } from '@/utils/feedback'

const props = withDefaults(
  defineProps<{
    show: boolean
    title?: string
    loading?: boolean
    /** 驳回是否必须填写原因。 */
    requireRejectReason?: boolean
  }>(),
  { title: '审核', loading: false, requireRejectReason: true },
)

const emit = defineEmits<{
  'update:show': [value: boolean]
  approve: [reason: string]
  reject: [reason: string]
}>()

const reason = ref('')

watch(
  () => props.show,
  (open) => {
    if (open) reason.value = ''
  },
)

function onApprove() {
  emit('approve', reason.value.trim())
}

function onReject() {
  if (props.requireRejectReason && !reason.value.trim()) {
    feedback.message.warning('驳回必须填写原因')
    return
  }
  emit('reject', reason.value.trim())
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="title"
    style="width: 460px"
    :mask-closable="!loading"
    @update:show="emit('update:show', $event)"
  >
    <div class="review-body">
      <slot />
      <NInput
        v-model:value="reason"
        type="textarea"
        placeholder="审核备注（驳回必填）"
        :rows="3"
        maxlength="200"
        show-count
      />
    </div>
    <template #footer>
      <div class="review-footer">
        <NButton
          :disabled="loading"
          @click="emit('update:show', false)"
        >
          取消
        </NButton>
        <NButton
          type="error"
          ghost
          :loading="loading"
          @click="onReject"
        >
          驳回
        </NButton>
        <NButton
          type="primary"
          :loading="loading"
          @click="onApprove"
        >
          通过
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.review-body {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-4);
}
.review-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--ic-space-2);
}
</style>
