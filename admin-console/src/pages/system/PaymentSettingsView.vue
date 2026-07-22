<script setup lang="ts">
/**
 * 系统设置 · 支付配置（高风险）。
 * 服务端 GET /admin/payment-channel-settings 返回渠道开关列表
 * [{ channel, displayName, enabled }]，PUT 提交 { channels: [{ channel, enabled }] }。
 * 修改支付渠道配置需二次确认、幂等键与审计。
 */
import { onMounted, ref } from 'vue'
import { NButton, NForm, NFormItem, NSpin, NSwitch } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import AuditRiskAlert from '@/components/AuditRiskAlert.vue'
import { systemService } from '@/api/services'
import { runAudited } from '@/composables/useAuditedAction'
import type { PaymentChannelSetting } from '@/api/models'
import type { NormalizedError } from '@/api/types'
import { toastError } from '@/utils/feedback'

const loading = ref(false)
const saving = ref(false)
const loadError = ref(false)
const channels = ref<PaymentChannelSetting[]>([])

onMounted(load)

async function load(): Promise<void> {
  loading.value = true
  loadError.value = false
  try {
    channels.value = await systemService.getPaymentChannelSettings()
  } catch (e) {
    loadError.value = (e as NormalizedError).status !== 401
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  saving.value = true
  try {
    await runAudited({
      title: '保存支付配置',
      content: '确认修改支付渠道配置？该操作携带幂等键并写入审计日志。',
      highRisk: true,
      positiveText: '确认保存',
      execute: () =>
        systemService.updatePaymentChannelSettings(
          channels.value.map((c) => ({ channel: c.channel, enabled: c.enabled })),
        ),
      successText: '支付配置已保存',
    })
  } catch (e) {
    toastError((e as { message?: string }).message ?? '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section>
    <PageHeader
      title="系统设置 · 支付配置"
      description="支付渠道开关；修改写入审计"
      :breadcrumb="['系统设置', '支付配置']"
    />
    <AuditRiskAlert title="支付配置为高风险项" />
    <div class="settings-panel">
      <NSpin :show="loading">
        <NForm
          label-placement="left"
          label-width="160"
        >
          <NFormItem
            v-for="channel in channels"
            :key="channel.channel"
            :label="channel.displayName"
          >
            <NSwitch v-model:value="channel.enabled" />
          </NFormItem>
          <NFormItem
            v-if="channels.length"
            label=" "
          >
            <NButton
              type="primary"
              :loading="saving"
              @click="save"
            >
              保存配置
            </NButton>
          </NFormItem>
        </NForm>
      </NSpin>
      <p
        v-if="loadError"
        class="settings-note"
      >
        配置接口尚未就绪（/admin/payment-channel-settings）。
      </p>
      <p
        v-else-if="!loading && !channels.length"
        class="settings-note"
      >
        暂无可配置的支付渠道。
      </p>
    </div>
  </section>
</template>

<style scoped>
.settings-panel {
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
  padding: var(--ic-space-lg);
  max-width: 720px;
}
.settings-note {
  margin-top: var(--ic-space-md);
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
</style>
