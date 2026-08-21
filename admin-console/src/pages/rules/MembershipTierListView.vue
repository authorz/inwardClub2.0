<script setup lang="ts">
import { h, reactive, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NInputNumber, NSelect, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { membershipTierService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
import type { MembershipTier, TierBenefitConfig } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'

const couponTypes = [
  { label: '赛事门票券', value: 'event_ticket' },
  { label: '小吃券', value: 'snack' },
  { label: '酒水券', value: 'alcohol' },
  { label: '饮料券', value: 'beverage' },
  { label: '餐食券', value: 'meal' },
]
const periods = [
  { label: '达级时', value: 'once' },
  { label: '每日', value: 'daily' },
  { label: '每周', value: 'weekly' },
  { label: '每月', value: 'monthly' },
]
const triggers = [
  { label: '达到等级', value: 'tier_achieved' },
  { label: '低消达标', value: 'low_spend' },
  { label: '首次下单', value: 'first_order' },
  { label: '到店', value: 'visit' },
  { label: '工作日赛事', value: 'weekday_event' },
  { label: '周赛', value: 'weekly_event' },
  { label: '月赛', value: 'monthly_event' },
]
const couponTypeLabel = Object.fromEntries(couponTypes.map((item) => [item.value, item.label]))

const listRef = ref<ResourceListInstance | null>(null)
const fields: FilterField[] = [{ key: 'keyword', label: '等级名称', type: 'input' }]

function benefitSummary(row: MembershipTier): string {
  const config = row.benefitConfig || { points: [], coupons: [], descriptions: [] }
  const points = (config.points || []).reduce((sum, item) => sum + Number(item.amount || 0), 0)
  const coupons = (config.coupons || []).reduce((sum, item) => sum + Number(item.quantity || 0), 0)
  return [points ? `${points} 积分` : '', coupons ? `${coupons} 张券` : '', ...(config.descriptions || [])]
    .filter(Boolean)
    .join(' · ') || '未配置'
}

const columns = [
  textColumn<MembershipTier>('等级名称', 'name'),
  textColumn<MembershipTier>('等级', 'level', { width: 90 }),
  textColumn<MembershipTier>('成长值门槛', 'threshold', { width: 130 }),
  renderColumn<MembershipTier>('权益', 'benefits', (row) => benefitSummary(row), 320),
  statusColumn<MembershipTier>('状态', 'status', RESOURCE_STATUS_OPTIONS),
  actionsColumn<MembershipTier>((row) => h(NSpace, {}, () => [
    h(PermissionButton, { permission: PERMISSIONS.RULE_PUBLISH, onClick: () => openEdit(row) }, () => '编辑'),
    h(PermissionButton, { permission: PERMISSIONS.RULE_PUBLISH, type: 'error', onClick: () => disable(row) }, () => '禁用'),
  ]), 160),
]

const drawerShow = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const descriptionsText = ref('')
const form = reactive({
  name: '',
  level: 1,
  threshold: 0,
  benefitConfig: { points: [], coupons: [], descriptions: [] } as TierBenefitConfig,
})

function emptyBenefits(): TierBenefitConfig {
  return { points: [], coupons: [], descriptions: [] }
}

function openCreate(): void {
  editingId.value = null
  Object.assign(form, { name: '', level: 1, threshold: 0, benefitConfig: emptyBenefits() })
  descriptionsText.value = ''
  drawerShow.value = true
}

function openEdit(row: MembershipTier): void {
  editingId.value = String(row.id)
  const benefitConfig = JSON.parse(JSON.stringify(row.benefitConfig || emptyBenefits())) as TierBenefitConfig
  Object.assign(form, { name: row.name, level: row.level, threshold: row.threshold ?? 0, benefitConfig })
  descriptionsText.value = (benefitConfig.descriptions || []).join('\n')
  drawerShow.value = true
}

function addPointBenefit(): void {
  form.benefitConfig.points.push({ amount: 1000, period: 'once', trigger: 'tier_achieved' })
}

function addCouponBenefit(): void {
  form.benefitConfig.coupons.push({ couponType: 'alcohol', quantity: 1, period: 'once', trigger: 'tier_achieved' })
}

function summaryText(config: TierBenefitConfig): string {
  const pointText = config.points.map((item) => `${periods.find((x) => x.value === item.period)?.label || ''}${triggers.find((x) => x.value === item.trigger)?.label || ''}赠送${item.amount}积分`)
  const couponText = config.coupons.map((item) => `${periods.find((x) => x.value === item.period)?.label || ''}${triggers.find((x) => x.value === item.trigger)?.label || ''}赠送${couponTypeLabel[item.couponType] || '券'}${item.quantity}张`)
  return [...pointText, ...couponText, ...config.descriptions].join('；')
}

async function submit(): Promise<void> {
  if (!form.name.trim()) return toastError('请填写等级名称')
  const descriptions = descriptionsText.value.split('\n').map((item) => item.trim()).filter(Boolean)
  const benefitConfig = { ...form.benefitConfig, descriptions }
  const payload = { ...form, name: form.name.trim(), benefitConfig, benefits: summaryText(benefitConfig) }
  submitting.value = true
  try {
    if (editingId.value) await membershipTierService.update(editingId.value, payload)
    else await membershipTierService.create(payload)
    toastSuccess('VIP 等级与权益已保存')
    drawerShow.value = false
    listRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally {
    submitting.value = false
  }
}

async function disable(row: MembershipTier): Promise<void> {
  const ok = await runAudited({
    title: '禁用会员等级', content: `确认禁用「${row.name}」？`, highRisk: true,
    execute: () => membershipTierService.action(API_PATHS.membershipTiers.disable(String(row.id))),
    successText: '已禁用',
  })
  if (ok) listRef.value?.reload()
}

const toolbarActions = [{
  key: 'create', label: '新增等级', type: 'primary' as const,
  permission: PERMISSIONS.RULE_PUBLISH, onClick: openCreate,
}]
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="VIP 等级"
      description="配置成长值门槛、积分、券与文字权益；资产福利在会员达级后自动发放"
      :breadcrumb="['权益规则', 'VIP 等级']"
      :fields="fields"
      :columns="columns"
      :fetcher="membershipTierService.list"
      :toolbar-actions="toolbarActions"
    />
    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑等级' : '新增等级'"
      :submitting="submitting"
      :width="760"
      @submit="submit"
    >
      <NForm label-placement="top">
        <div class="tier-basic">
          <NFormItem
            label="等级名称"
            required
          >
            <NInput v-model:value="form.name" />
          </NFormItem>
          <NFormItem label="等级序号">
            <NInputNumber
              v-model:value="form.level"
              :min="1"
            />
          </NFormItem>
          <NFormItem label="成长值门槛">
            <NInputNumber
              v-model:value="form.threshold"
              :min="0"
            />
          </NFormItem>
        </div>

        <div class="benefit-section">
          <div class="benefit-head">
            <div><h3>积分福利</h3><p>可以配置多条不同周期或场景的积分规则。</p></div><NButton
              size="small"
              @click="addPointBenefit"
            >
              添加积分
            </NButton>
          </div>
          <div
            v-for="(item, index) in form.benefitConfig.points"
            :key="`point-${index}`"
            class="benefit-row"
          >
            <NInputNumber
              v-model:value="item.amount"
              :min="1"
              placeholder="积分数量"
            />
            <NSelect
              v-model:value="item.period"
              :options="periods"
            />
            <NSelect
              v-model:value="item.trigger"
              :options="triggers"
            />
            <NButton
              text
              type="error"
              @click="form.benefitConfig.points.splice(index, 1)"
            >
              移除
            </NButton>
          </div>
          <p
            v-if="!form.benefitConfig.points.length"
            class="empty-line"
          >
            未配置积分福利
          </p>
        </div>

        <div class="benefit-section">
          <div class="benefit-head">
            <div><h3>赠送券</h3><p>选择固定券类型，可添加多种；一张券兑换一件商品或一张门票。</p></div><NButton
              size="small"
              @click="addCouponBenefit"
            >
              添加券
            </NButton>
          </div>
          <div
            v-for="(item, index) in form.benefitConfig.coupons"
            :key="`coupon-${index}`"
            class="benefit-row benefit-row--coupon"
          >
            <NSelect
              v-model:value="item.couponType"
              :options="couponTypes"
            />
            <NInputNumber
              v-model:value="item.quantity"
              :min="1"
              :max="99"
              placeholder="数量"
            />
            <NSelect
              v-model:value="item.period"
              :options="periods"
            />
            <NSelect
              v-model:value="item.trigger"
              :options="triggers"
            />
            <NButton
              text
              type="error"
              @click="form.benefitConfig.coupons.splice(index, 1)"
            >
              移除
            </NButton>
          </div>
          <p
            v-if="!form.benefitConfig.coupons.length"
            class="empty-line"
          >
            未配置券福利
          </p>
        </div>

        <NFormItem
          label="实物及服务权益（仅文字展示，每行一项）"
          class="benefit-section"
        >
          <NInput
            v-model:value="descriptionsText"
            type="textarea"
            :rows="5"
            placeholder="例如：每月赠送礼品区任选1份"
          />
        </NFormItem>
      </NForm>
      <p class="form-note">
        自动发放的积分流水说明统一为“VIP等级福利”；券自发放起 30 天有效。周期与触发场景同时用于小程序展示。
      </p>
    </FormDrawer>
  </div>
</template>

<style scoped>
.tier-basic { display: grid; grid-template-columns: minmax(0, 1fr) 140px 180px; gap: 16px; }
.benefit-section { margin-top: 20px; padding-top: 20px; border-top: 1px solid var(--ic-color-border); }
.benefit-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; margin-bottom: 12px; }
.benefit-head h3 { margin: 0; font-size: 15px; color: var(--ic-color-text-primary); }
.benefit-head p, .empty-line, .form-note { margin: 5px 0 0; font-size: var(--ic-font-xs); color: var(--ic-color-text-tertiary); }
.benefit-row { display: grid; grid-template-columns: 150px 120px minmax(160px, 1fr) 48px; gap: 10px; align-items: center; padding: 10px 0; border-top: 1px solid var(--ic-color-border); }
.benefit-row--coupon { grid-template-columns: minmax(140px, 1fr) 100px 110px minmax(150px, 1fr) 48px; }
@media (max-width: 720px) {
  .tier-basic, .benefit-row, .benefit-row--coupon { grid-template-columns: 1fr; }
}
</style>
