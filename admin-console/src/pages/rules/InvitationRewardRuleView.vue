<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NInputNumber, NSpin, NSwitch, NTag } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import AuditRiskAlert from '@/components/AuditRiskAlert.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { ruleDefinitionService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import type { RuleDefinition } from '@/api/models'
import { toastError } from '@/utils/feedback'

interface InvitationRewardConfig {
  schemaVersion?: number
  firstLowSpendRewardCoins?: number
  firstLowSpendRewardPoints?: number
  commissionRateBasisPoints?: number
}

const loading = ref(false)
const saving = ref(false)
const rule = ref<RuleDefinition | null>(null)
const enabled = ref(false)
const firstCoins = ref(0)
const firstPoints = ref(0)
const commissionPercent = ref(0)

const statusLabel = computed(() => {
  if (!rule.value) return '尚未创建'
  if (rule.value.status === 'published' && rule.value.enabled) return '已生效'
  if (rule.value.status === 'draft') return '草稿'
  return '未启用'
})

const displayCopy = computed(() => {
  const rewards = []
  if (firstCoins.value > 0) rewards.push(`${firstCoins.value} 金币`)
  if (firstPoints.value > 0) rewards.push(`${firstPoints.value} 积分`)
  const first = rewards.length
    ? `好友绑定邀请码并首次完成门店低消后，邀请人获得 ${rewards.join(' + ')}。`
    : ''
  const commission = commissionPercent.value > 0
    ? `好友绑定后每笔微信支付（含金币充值及绑定会员的门店微信收款），邀请人获得 ${commissionPercent.value}% 的金币奖励；金币支付、支付宝和现金不计入。`
    : ''
  return [first, commission].filter(Boolean).join(' ')
})

onMounted(load)

function applyConfig(config: unknown): void {
  const value = (config && typeof config === 'object' ? config : {}) as InvitationRewardConfig
  firstCoins.value = Number(value.firstLowSpendRewardCoins ?? 0)
  firstPoints.value = Number(value.firstLowSpendRewardPoints ?? 0)
  commissionPercent.value = Number(value.commissionRateBasisPoints ?? 0) / 100
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const result = await ruleDefinitionService.list({ keyword: 'invite_reward', pageSize: 100 })
    const candidates = result.items.filter((item) => item.ruleKey === 'invite_reward')
    rule.value =
      candidates.find((item) => item.status === 'published' && item.enabled) ??
      candidates[0] ??
      null
    enabled.value = !!(rule.value?.status === 'published' && rule.value.enabled)
    applyConfig(rule.value?.configJson)
  } catch (e) {
    toastError((e as { message?: string }).message ?? '邀请奖励规则加载失败')
  } finally {
    loading.value = false
  }
}

function validate(): boolean {
  if (!Number.isInteger(firstCoins.value) || firstCoins.value < 0 || firstCoins.value > 1_000_000) {
    toastError('首次奖励金币必须是 0 至 1000000 的整数')
    return false
  }
  if (!Number.isInteger(firstPoints.value) || firstPoints.value < 0 || firstPoints.value > 100_000_000) {
    toastError('首次奖励积分必须是 0 至 100000000 的整数')
    return false
  }
  if (commissionPercent.value < 0 || commissionPercent.value > 100) {
    toastError('微信支付返佣比例必须在 0% 至 100% 之间')
    return false
  }
  if (enabled.value && firstCoins.value === 0 && firstPoints.value === 0 && commissionPercent.value === 0) {
    toastError('启用规则时至少需要配置一项奖励')
    return false
  }
  return true
}

async function persist(): Promise<void> {
  const configJson: InvitationRewardConfig = {
    schemaVersion: 1,
    firstLowSpendRewardCoins: firstCoins.value,
    firstLowSpendRewardPoints: firstPoints.value,
    commissionRateBasisPoints: Math.round(commissionPercent.value * 100),
  }
  if (rule.value) {
    rule.value = await ruleDefinitionService.update(rule.value.id, { configJson })
  } else {
    rule.value = await ruleDefinitionService.create({
      ruleKey: 'invite_reward',
      scopeType: 'global',
      configJson,
      enabled: false,
    })
  }
  if (enabled.value) {
    rule.value = await ruleDefinitionService.action<RuleDefinition>(
      API_PATHS.ruleDefinitions.publish(rule.value.id),
      undefined,
      true,
    )
  } else if (rule.value.status !== 'disabled' || rule.value.enabled) {
    rule.value = await ruleDefinitionService.action<RuleDefinition>(
      API_PATHS.ruleDefinitions.disable(rule.value.id),
      undefined,
      true,
    )
  }
}

async function save(): Promise<void> {
  if (!validate()) return
  saving.value = true
  try {
    const ok = await runAudited({
      title: '保存邀请奖励规则',
      content: enabled.value
        ? `确认启用邀请奖励？${displayCopy.value}`
        : '确认停用邀请奖励？停用后新的微信支付不再产生邀请奖励，已发放记录不受影响。',
      highRisk: true,
      positiveText: '确认保存',
      execute: persist,
      successText: enabled.value ? '邀请奖励规则已生效' : '邀请奖励规则已停用',
    })
    if (ok) await load()
  } catch (e) {
    toastError((e as { message?: string }).message ?? '邀请奖励规则保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section>
    <PageHeader
      title="邀请奖励"
      description="配置邀请人首次低消奖励和受邀好友微信支付返佣；金币支付不参与返佣"
      :breadcrumb="['权益规则', '邀请奖励']"
    >
      <template #actions>
        <NTag :type="rule?.status === 'published' && rule?.enabled ? 'success' : 'default'">
          {{ statusLabel }}
        </NTag>
      </template>
    </PageHeader>

    <AuditRiskAlert title="邀请奖励会直接改变邀请人的金币与积分余额" />

    <NSpin :show="loading">
      <div class="rule-form">
        <div class="rule-row rule-row--switch">
          <div>
            <h2>启用邀请奖励</h2>
            <p>停用后，小程序不展示奖励承诺，新的微信支付也不会产生返佣。</p>
          </div>
          <NSwitch v-model:value="enabled" />
        </div>

        <div class="rule-row">
          <div>
            <h2>首次完成低消</h2>
            <p>受邀好友绑定后首次完成门店低消时，奖励发给邀请人。</p>
          </div>
          <div class="field-group">
            <label>
              <span>赠送金币</span>
              <NInputNumber
                v-model:value="firstCoins"
                :min="0"
                :max="1000000"
                :precision="0"
                :show-button="false"
              />
            </label>
            <label>
              <span>赠送积分</span>
              <NInputNumber
                v-model:value="firstPoints"
                :min="0"
                :max="100000000"
                :precision="0"
                :show-button="false"
              />
            </label>
          </div>
        </div>

        <div class="rule-row">
          <div>
            <h2>微信支付返佣</h2>
            <p>餐品、活动、金币充值及绑定会员的门店微信收款均参与；金币支付、支付宝和现金不参与。</p>
          </div>
          <label class="rate-field">
            <NInputNumber
              v-model:value="commissionPercent"
              :min="0"
              :max="100"
              :precision="2"
              :show-button="false"
            />
            <span>%</span>
          </label>
        </div>

        <div class="rule-preview">
          <span>小程序展示文案</span>
          <p>{{ enabled ? displayCopy : '邀请奖励暂未启用' }}</p>
        </div>

        <div class="rule-actions">
          <PermissionButton
            :permission="PERMISSIONS.RULE_PUBLISH"
            type="primary"
            :disabled="saving"
            @click="save"
          >
            {{ saving ? '保存中…' : '保存设置' }}
          </PermissionButton>
        </div>
      </div>
    </NSpin>
  </section>
</template>

<style scoped>
.rule-form { max-width: 860px; }
.rule-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--ic-space-xl);
  padding: var(--ic-space-xl) 0;
  border-bottom: 1px solid var(--ic-color-border);
}
.rule-row--switch { align-items: center; border-top: 1px solid var(--ic-color-border); }
.rule-row h2 { margin: 0; color: var(--ic-color-text); font-size: var(--ic-font-lg); font-weight: 600; }
.rule-row p { max-width: 520px; margin: var(--ic-space-xs) 0 0; color: var(--ic-color-text-secondary); font-size: var(--ic-font-sm); }
.field-group { display: flex; gap: var(--ic-space-lg); flex: none; }
.field-group label { display: flex; flex-direction: column; gap: var(--ic-space-xs); color: var(--ic-color-text-secondary); font-size: var(--ic-font-sm); }
.field-group :deep(.n-input-number) { width: 170px; }
.rate-field { display: flex; align-items: center; gap: var(--ic-space-sm); flex: none; color: var(--ic-color-text-secondary); }
.rate-field :deep(.n-input-number) { width: 170px; }
.rule-preview { padding: var(--ic-space-lg) 0; }
.rule-preview span { color: var(--ic-color-text-secondary); font-size: var(--ic-font-sm); }
.rule-preview p { margin: var(--ic-space-xs) 0 0; color: var(--ic-color-text); line-height: 1.65; }
.rule-actions { display: flex; justify-content: flex-end; padding-top: var(--ic-space-md); }
@media (max-width: 760px) {
  .rule-row { flex-direction: column; gap: var(--ic-space-md); }
  .rule-row--switch { flex-direction: row; }
  .field-group { width: 100%; flex-direction: column; }
  .field-group :deep(.n-input-number), .rate-field :deep(.n-input-number) { width: 100%; }
  .rate-field { width: 100%; }
}
</style>
