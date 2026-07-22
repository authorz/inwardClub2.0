<script setup lang="ts">
/**
 * 审计风险提示条（公共组件）。
 * 用于所有涉及资产、钱包、退款、规则、跨店写操作的表单/抽屉/确认弹窗，
 * 明确告知操作将被记入审计日志。
 */
import { NAlert } from 'naive-ui'

withDefaults(
  defineProps<{
    /** 是否跨店写操作（需要填写目标门店与原因） */
    crossStore?: boolean
    title?: string
  }>(),
  {
    crossStore: false,
    title: '高风险操作提示',
  },
)
</script>

<template>
  <NAlert
    type="warning"
    :title="title"
    :bordered="true"
    class="audit-risk-alert"
  >
    <div>此操作涉及资产 / 钱包 / 退款 / 规则等敏感能力，将写入审计日志（含操作者、角色、对象与前后差异）。</div>
    <div
      v-if="crossStore"
      class="audit-risk-alert__cross"
    >
      这是跨店写操作，请确认目标门店 / 投放范围并填写操作原因后再提交。
    </div>
    <slot />
  </NAlert>
</template>

<style scoped>
.audit-risk-alert {
  margin-bottom: var(--ic-space-md);
}
.audit-risk-alert__cross {
  margin-top: var(--ic-space-xs);
  font-weight: 600;
}
</style>
