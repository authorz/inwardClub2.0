<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { NButton, NForm, NFormItem, NInputNumber, NSpin, NText } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import AuditRiskAlert from '@/components/AuditRiskAlert.vue'
import { systemService } from '@/api/services'
import { runAudited } from '@/composables/useAuditedAction'
import { toastError } from '@/utils/feedback'

const loading = ref(false)
const saving = ref(false)
const version = ref(0)
const form = reactive({
  pointsDivisor: 5,
  belowBasePointsDivisor: 2,
  coinPointsDivisor: 2000,
})

const pointRatioText = computed(
  () => `每 ${form.pointsDivisor || 0} 原始积分折算为 1 到账积分`,
)
const belowBaseRatioText = computed(
  () => `存入积分低于基数时，每 ${form.belowBasePointsDivisor || 0} 原始积分折算为 1 到账积分`,
)
const coinRatioText = computed(
  () => `每 ${form.coinPointsDivisor || 0} 个金币计算基数积分兑换 1 金币`,
)

onMounted(load)

async function load(): Promise<void> {
  loading.value = true
  try {
    const settings = await systemService.getPointReviewSettings()
    form.pointsDivisor = settings.pointsDivisor
    form.belowBasePointsDivisor = settings.belowBasePointsDivisor
    form.coinPointsDivisor = settings.coinPointsDivisor
    version.value = settings.version
  } catch (e) {
    toastError((e as { message?: string }).message ?? '读取积分审核配置失败')
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  if (!Number.isInteger(form.pointsDivisor) || form.pointsDivisor <= 0) {
    return toastError('积分比例必须是大于 0 的整数')
  }
  if (!Number.isInteger(form.belowBasePointsDivisor) || form.belowBasePointsDivisor <= 0) {
    return toastError('低于基数折算比例必须是大于 0 的整数')
  }
  if (!Number.isInteger(form.coinPointsDivisor) || form.coinPointsDivisor <= 0) {
    return toastError('金币兑换比例必须是大于 0 的整数')
  }
  saving.value = true
  try {
    await runAudited({
      title: '保存积分审核配置',
      content: '新比例仅影响保存后审核的申请，历史审核记录继续使用原比例快照。确认保存？',
      highRisk: true,
      positiveText: '确认保存',
      execute: async () => {
        const settings = await systemService.updatePointReviewSettings({
          pointsDivisor: form.pointsDivisor,
          belowBasePointsDivisor: form.belowBasePointsDivisor,
          coinPointsDivisor: form.coinPointsDivisor,
        })
        version.value = settings.version
        return settings
      },
      successText: '积分审核配置已保存',
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
      title="积分审核配置"
      description="配置员工审核存积分时使用的积分折算与金币奖励比例"
      :breadcrumb="['系统设置', '积分审核配置']"
    />
    <AuditRiskAlert title="修改比例会影响后续审核产生的会员资产" />
    <div class="settings-panel">
      <NSpin :show="loading">
        <NForm
          label-placement="left"
          label-width="150"
        >
          <NFormItem label="标准及超出部分比例">
            <div class="field-stack">
              <NInputNumber
                v-model:value="form.pointsDivisor"
                :min="1"
                :precision="0"
                class="number-input"
              />
              <NText depth="3">
                {{ pointRatioText }}。用于非营业时段、无基数和高于基数的超出部分，默认值为 5。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label="低于基数折算比例">
            <div class="field-stack">
              <NInputNumber
                v-model:value="form.belowBasePointsDivisor"
                :min="1"
                :precision="0"
                class="number-input"
              />
              <NText depth="3">
                {{ belowBaseRatioText }}，不足 1 积分的部分向下取整，默认值为 2。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label="金币兑换比例">
            <div class="field-stack">
              <NInputNumber
                v-model:value="form.coinPointsDivisor"
                :min="1"
                :precision="0"
                class="number-input"
              />
              <NText depth="3">
                {{ coinRatioText }}，不足 1 金币的部分向下取整。1.0 默认值为 2000。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label="当前规则版本">
            <NText>{{ version || '—' }}</NText>
          </NFormItem>
          <NFormItem label=" ">
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
    </div>
  </section>
</template>

<style scoped>
.settings-panel {
  max-width: 760px;
  padding: var(--ic-space-lg);
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
}
.field-stack {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-xs);
}
.number-input {
  width: 240px;
}
</style>
