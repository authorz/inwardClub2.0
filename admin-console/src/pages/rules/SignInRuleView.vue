<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NButton, NInputNumber, NSpin, NTag } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import AuditRiskAlert from '@/components/AuditRiskAlert.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { ruleDefinitionService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import type { RuleDefinition } from '@/api/models'
import { toastError } from '@/utils/feedback'

const DEFAULT_REWARDS = [100, 200, 300, 400, 500, 600, 700]

const loading = ref(false)
const saving = ref(false)
const rule = ref<RuleDefinition | null>(null)
const rewards = ref<number[]>(DEFAULT_REWARDS.slice())

const statusLabel = computed(() => {
  if (!rule.value) return '尚未创建'
  if (rule.value.status === 'published' && rule.value.enabled) return '已生效'
  if (rule.value.status === 'draft') return '草稿'
  return '未启用'
})

onMounted(load)

function parseRewards(config: unknown): number[] {
  if (!config || typeof config !== 'object') return DEFAULT_REWARDS.slice()
  const value = config as { dailyRewards?: unknown; dailyPoints?: unknown }
  const source = Array.isArray(value.dailyRewards)
    ? value.dailyRewards
    : Array.isArray(value.dailyPoints)
      ? value.dailyPoints
      : []
  const parsed = source.map(Number).filter((item) => Number.isInteger(item) && item > 0)
  return parsed.length ? parsed : DEFAULT_REWARDS.slice()
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const result = await ruleDefinitionService.list({ keyword: 'sign_in', pageSize: 100 })
    const candidates = result.items.filter((item) => item.ruleKey === 'sign_in')
    rule.value =
      candidates.find((item) => item.status === 'published' && item.enabled) ??
      candidates[0] ??
      null
    rewards.value = rule.value ? parseRewards(rule.value.configJson) : DEFAULT_REWARDS.slice()
  } catch (e) {
    toastError((e as { message?: string }).message ?? '签到规则加载失败')
  } finally {
    loading.value = false
  }
}

function updateReward(index: number, value: number | null): void {
  rewards.value[index] = value ?? 0
}

function addDay(): void {
  rewards.value.push(rewards.value[rewards.value.length - 1] || 100)
}

function removeDay(): void {
  if (rewards.value.length > 1) rewards.value.pop()
}

function validate(): boolean {
  if (rewards.value.some((item) => !Number.isInteger(item) || item <= 0)) {
    toastError('每天的签到积分必须是大于 0 的整数')
    return false
  }
  return true
}

async function persist(): Promise<void> {
  const lastReward = rewards.value[rewards.value.length - 1]
  const configJson = {
    dailyRewards: rewards.value,
    capDay: rewards.value.length,
    tailReward: lastReward,
    enabled: true,
  }

  if (rule.value) {
    const updated = await ruleDefinitionService.update(rule.value.id, {
      configJson,
      enabled: true,
    })
    rule.value = updated
    if (updated.status !== 'published') {
      rule.value = await ruleDefinitionService.action<RuleDefinition>(
        API_PATHS.ruleDefinitions.publish(updated.id),
        undefined,
        true,
      )
    }
    return
  }

  const created = await ruleDefinitionService.create({
    ruleKey: 'sign_in',
    scopeType: 'global',
    configJson,
    enabled: true,
  })
  rule.value = await ruleDefinitionService.action<RuleDefinition>(
    API_PATHS.ruleDefinitions.publish(created.id),
    undefined,
    true,
  )
}

async function save(): Promise<void> {
  if (!validate()) return
  saving.value = true
  try {
    const ok = await runAudited({
      title: '保存签到规则',
      content: `确认保存连续签到规则？第 ${rewards.value.length} 天及以后每天奖励 ${rewards.value[rewards.value.length - 1]} 积分。`,
      highRisk: true,
      positiveText: '确认保存',
      execute: persist,
      successText: '签到规则已生效',
    })
    if (ok) await load()
  } catch (e) {
    toastError((e as { message?: string }).message ?? '签到规则保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section>
    <PageHeader
      title="签到规则"
      description="配置会员连续签到的每日积分；最后一天的奖励将持续用于后续连续签到"
      :breadcrumb="['VIP / 权益规则', '签到规则']"
    >
      <template #actions>
        <NTag :type="rule?.status === 'published' && rule?.enabled ? 'success' : 'default'">
          {{ statusLabel }}
        </NTag>
      </template>
    </PageHeader>

    <AuditRiskAlert title="签到规则会直接影响会员积分" />

    <NSpin :show="loading">
      <div class="rule-panel">
        <div class="rule-panel__head">
          <div>
            <h2>连续签到阶梯</h2>
            <p>断签后从第 1 天重新计算；最后一档自动作为长期连续签到奖励。</p>
          </div>
          <div class="rule-panel__controls">
            <NButton
              secondary
              @click="addDay"
            >
              增加一天
            </NButton>
            <NButton
              secondary
              :disabled="rewards.length <= 1"
              @click="removeDay"
            >
              减少一天
            </NButton>
          </div>
        </div>

        <div class="reward-list">
          <div
            v-for="(reward, index) in rewards"
            :key="index"
            class="reward-row"
          >
            <div class="reward-row__day">
              {{ index === rewards.length - 1 ? `连续第 ${index + 1} 天及以后` : `连续第 ${index + 1} 天` }}
            </div>
            <div class="reward-row__value">
              <NInputNumber
                :value="reward"
                :min="1"
                :precision="0"
                :show-button="false"
                @update:value="updateReward(index, $event)"
              />
              <span>积分</span>
            </div>
          </div>
        </div>

        <div class="rule-panel__footer">
          <p>
            当前规则：连续第 {{ rewards.length }} 天及以后，每天奖励
            <strong>{{ rewards[rewards.length - 1] }}</strong> 积分。
          </p>
          <PermissionButton
            :permission="PERMISSIONS.RULE_PUBLISH"
            type="primary"
            :disabled="saving"
            @click="save"
          >
            {{ saving ? '保存中…' : '保存并生效' }}
          </PermissionButton>
        </div>
      </div>
    </NSpin>
  </section>
</template>

<style scoped>
.rule-panel {
  max-width: 820px;
  padding: var(--ic-space-lg);
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
}
.rule-panel__head,
.rule-panel__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-lg);
}
.rule-panel__head h2 {
  margin: 0;
  font-size: var(--ic-font-lg);
  font-weight: 600;
  color: var(--ic-color-text);
}
.rule-panel__head p,
.rule-panel__footer p {
  margin: var(--ic-space-xs) 0 0;
  font-size: var(--ic-font-sm);
  color: var(--ic-color-text-secondary);
}
.rule-panel__controls {
  display: flex;
  gap: var(--ic-space-sm);
  flex: none;
}
.reward-list {
  margin-top: var(--ic-space-lg);
  border-top: 1px solid var(--ic-color-border);
}
.reward-row {
  min-height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-lg);
  border-bottom: 1px solid var(--ic-color-border);
}
.reward-row__day {
  font-size: var(--ic-font-md);
  color: var(--ic-color-text);
}
.reward-row__value {
  display: flex;
  align-items: center;
  gap: var(--ic-space-sm);
  color: var(--ic-color-text-secondary);
}
.reward-row__value :deep(.n-input-number) {
  width: 160px;
}
.rule-panel__footer {
  margin-top: var(--ic-space-lg);
}
.rule-panel__footer p {
  margin: 0;
}
.rule-panel__footer strong {
  color: var(--ic-color-text);
}
</style>
