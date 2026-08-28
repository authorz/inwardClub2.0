<script setup lang="ts">
/**
 * 会员管理：会员列表、详情查看与人工钱包调账。
 * 人工调账为高风险写操作，服务端带 Idempotency-Key，需二次确认。
 */
import { computed, h, reactive, ref } from 'vue'
import {
  NAvatar,
  NButton,
  NDatePicker,
  NInput,
  NInputNumber,
  NModal,
  NRadioButton,
  NRadioGroup,
  NSelect,
  NSpace,
  NSpin,
  NTabPane,
  NTabs,
  NTag,
} from 'naive-ui'
import type { DataTableColumns, DataTableSortOrder, DataTableSortState } from 'naive-ui'
import { couponService, memberService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { PERM } from '@/constants/permissions'
import { actionColumn, dateColumn, statusColumn, textColumn } from '@/utils/columns'
import { formatDateTime } from '@/utils/format'
import { ApiError } from '@/api/error'
import { feedback } from '@/utils/feedback'
import { DataTable, PageHeader, PermissionButton } from '@/components/common'
import type {
  CouponCategory,
  Member,
  MemberCouponEntitlement,
  WalletLedgerEntry,
} from '@/types/models'
import {
  ACTIVE_STATUS,
  toOptions,
  WALLET_ASSET_TYPE,
  WALLET_REASON_LABELS,
} from '@/constants/enums'

type MemberSortField = 'pointsBalance' | 'coinsBalance' | 'vipLevel'
type AdjustmentDirection = 'credit' | 'debit'
const sortBy = ref<MemberSortField | ''>('')
const sortOrder = ref<'asc' | 'desc'>('desc')
const balanceFormatter = new Intl.NumberFormat('zh-CN')

const list = useAsyncList<Member>(
  (params) =>
    memberService.list({
      ...params,
      ...(sortBy.value ? { sortBy: sortBy.value, sortOrder: sortOrder.value } : {}),
    }),
  { initialFilters: { keyword: '' } },
)
const registrationRange = ref<[number, number] | null>(null)

const detailShow = ref(false)
const detailLoading = ref(false)
const currentMember = ref<Member | null>(null)
const detailTab = ref<'coupons' | 'ledger' | 'adjust'>('coupons')
const couponMemberId = ref<string | number | null>(null)

const memberCoupons = useAsyncList<MemberCouponEntitlement>(
  (params) => memberService.couponEntitlements(couponMemberId.value!, params),
  { immediate: false },
)

const ledger = useAsyncList<WalletLedgerEntry>(
  (params) => memberService.walletLedger(params),
  { immediate: false },
)

const action = useAsyncAction()
const couponAction = useAsyncAction()

const couponShow = ref(false)
const couponLoading = ref(false)
const couponTarget = ref<Member | null>(null)
const couponTemplateOptions = ref<Array<{ label: string; value: number; validityDays: number }>>([])
const couponForm = reactive<{ templateId: number | null; expiresAt: number | null; reason: string }>({
  templateId: null,
  expiresAt: null,
  reason: '',
})
const COUPON_TYPE_LABELS: Record<string, string> = {
  event_ticket: '赛事券',
  admission_ticket: '门票券',
  snack: '小吃券',
  alcohol: '酒水券',
  beverage: '饮料券',
  drink: '饮品或啤酒券',
  meal: '餐食券',
  gift: '礼品券',
}
const COUPON_STATUS = {
  active: { value: 'active', label: '未使用', tone: 'success' as const },
  used: { value: 'used', label: '已使用', tone: 'default' as const },
  expired: { value: 'expired', label: '已过期', tone: 'warning' as const },
  void: { value: 'void', label: '已作废', tone: 'error' as const },
}

const adjustAssetTypeOptions = toOptions(WALLET_ASSET_TYPE)
  .filter(({ value }) => value !== WALLET_ASSET_TYPE.cash_balance.value)
  .map(({ label, value }) => ({ label, value }))
const ASSET_LABELS = Object.fromEntries(
  toOptions(WALLET_ASSET_TYPE).map(({ label, value }) => [value, label]),
)
const adjustForm = reactive<{
  assetType: string
  direction: AdjustmentDirection
  amount: number | null
  reason: string
}>({
  assetType: WALLET_ASSET_TYPE.coins.value,
  direction: 'credit',
  amount: null,
  reason: '',
})

function applyMemberFilters(): void {
  if (registrationRange.value) {
    list.filters.createdFrom = new Date(registrationRange.value[0]).toISOString()
    list.filters.createdTo = new Date(registrationRange.value[1]).toISOString()
  } else {
    delete list.filters.createdFrom
    delete list.filters.createdTo
  }
  list.applyFilters({})
}

function resetMemberFilters(): void {
  registrationRange.value = null
  list.reset()
}

function columnSortOrder(field: MemberSortField): DataTableSortOrder {
  if (sortBy.value !== field) return false
  return sortOrder.value === 'asc' ? 'ascend' : 'descend'
}

function handleSorter(sorter: DataTableSortState | DataTableSortState[] | null): void {
  const current = Array.isArray(sorter) ? sorter[0] : sorter
  const field = current?.columnKey
  if (
    !current?.order ||
    (field !== 'pointsBalance' && field !== 'coinsBalance' && field !== 'vipLevel')
  ) {
    sortBy.value = ''
    sortOrder.value = 'desc'
  } else {
    sortBy.value = field
    sortOrder.value = current.order === 'ascend' ? 'asc' : 'desc'
  }
  list.page.value = 1
  list.refresh()
}

async function openMember(row: Member, tab: 'coupons' | 'ledger' | 'adjust') {
  currentMember.value = row
  detailTab.value = tab
  detailShow.value = true
  detailLoading.value = true
  adjustForm.assetType = WALLET_ASSET_TYPE.coins.value
  adjustForm.direction = 'credit'
  adjustForm.amount = null
  adjustForm.reason = ''
  try {
    currentMember.value = await memberService.detail(row.id)
  } catch (err) {
    if (!(err instanceof ApiError)) throw err
  } finally {
    detailLoading.value = false
  }
  couponMemberId.value = row.id
  memberCoupons.applyFilters({})
  ledger.filters.memberId = row.id
  ledger.applyFilters({})
}

function openDetail(row: Member): void {
  void openMember(row, 'coupons')
}

function openAdjust(row: Member): void {
  void openMember(row, 'adjust')
}

function resetCouponForm(): void {
  couponForm.templateId = null
  couponForm.expiresAt = Date.now() + 30 * 24 * 60 * 60 * 1000
  couponForm.reason = ''
}

function handleCouponKindChange(templateId: number | null): void {
  couponForm.templateId = templateId
  const option = couponTemplateOptions.value.find((item) => item.value === templateId)
  couponForm.expiresAt = Date.now() + (option?.validityDays || 30) * 24 * 60 * 60 * 1000
}

async function openCouponGrant(): Promise<void> {
  if (!currentMember.value) return
  couponTarget.value = currentMember.value
  resetCouponForm()
  couponLoading.value = true
  try {
    const result = await couponService.categories({ page: 1, pageSize: 100 })
    const kinds = result.rows.filter((item: CouponCategory) => item.status === 'active')
    couponTemplateOptions.value = kinds.map((item) => ({
      label: `${item.name} · ${COUPON_TYPE_LABELS[item.businessType] ?? item.businessType} · 默认 ${item.defaultValidityDays} 天`,
      value: Number(item.canonicalTemplateId),
      validityDays: item.defaultValidityDays,
    }))
    if (!couponTemplateOptions.value.length) {
      feedback.message.warning('暂无启用中的券种，请联系总后台配置')
      return
    }
    couponShow.value = true
  } catch (error) {
    feedback.message.error(error instanceof ApiError ? error.message : '读取券种失败')
  } finally {
    couponLoading.value = false
  }
}

function submitCouponGrant(): void {
  const member = couponTarget.value
  const reason = couponForm.reason.trim()
  if (!member || couponForm.templateId == null) {
    feedback.message.error('请选择需要补发的优惠券')
    return
  }
  if (!couponForm.expiresAt || couponForm.expiresAt <= Date.now()) {
    feedback.message.error('有效期必须晚于当前时间')
    return
  }
  if (!reason) {
    feedback.message.error('请填写补发原因')
    return
  }
  const templateLabel = couponTemplateOptions.value.find(
    (item) => item.value === couponForm.templateId,
  )?.label ?? `优惠券 #${couponForm.templateId}`
  const expiresAt = new Date(couponForm.expiresAt).toISOString()
  void couponAction.run(
    () => memberService.grantCoupon(member.id, {
      templateId: couponForm.templateId!,
      expiresAt,
      reason,
    }),
    {
      confirm: {
        content: `确认向会员“${member.nickname || member.id}”补发本店券“${templateLabel}”？有效期至 ${formatDateTime(expiresAt)}，操作将写入审计日志。`,
        danger: true,
      },
      successMessage: '本店优惠券已补发',
      onSuccess: () => {
        couponShow.value = false
        couponTarget.value = null
        resetCouponForm()
        memberCoupons.refresh()
      },
    },
  )
}

function submitAdjust() {
  const member = currentMember.value
  const amount = adjustForm.amount
  const reason = adjustForm.reason.trim()
  if (!member) return
  if (amount == null || !Number.isInteger(amount) || amount <= 0) {
    feedback.message.error('请输入大于 0 的整数数量')
    return
  }
  if (!reason) {
    feedback.message.error('请填写调账原因')
    return
  }
  const assetLabel = ASSET_LABELS[adjustForm.assetType] ?? adjustForm.assetType
  const directionLabel = adjustForm.direction === 'credit' ? '增加' : '减少'
  void action.run(
    () =>
      memberService.adjustWallet(member.id, {
        assetType: adjustForm.assetType,
        direction: adjustForm.direction,
        amount,
        reason,
      }),
    {
      confirm: {
        content: `确认将会员“${member.nickname || member.id}”的${assetLabel}${directionLabel} ${balanceFormatter.format(amount)}？该操作将写入会员钱包流水。`,
        danger: true,
      },
      successMessage: '调账已提交',
      onSuccess: async () => {
        adjustForm.direction = 'credit'
        adjustForm.amount = null
        adjustForm.reason = ''
        if (currentMember.value) currentMember.value = await memberService.detail(currentMember.value.id)
        ledger.refresh()
      },
    },
  )
}

const columns = computed<DataTableColumns<Member>>(() => [
  textColumn<Member>('ID', (r) => r.id, { width: 80 }),
  {
    title: '头像',
    key: 'avatarUrl',
    width: 64,
    render: (row) =>
      (() => {
        const fallback = () => row.nickname?.trim().slice(0, 1) || String(row.id).slice(-1)
        return h(
          NAvatar,
          { size: 32, round: true, src: row.avatarUrl || undefined, objectFit: 'cover' },
          row.avatarUrl ? { fallback } : { default: fallback },
        )
      })(),
  },
  textColumn<Member>('昵称', (r) => r.nickname, { minWidth: 120 }),
  textColumn<Member>('手机号', (r) => r.phone, { width: 140 }),
  {
    title: '当前积分',
    key: 'pointsBalance',
    width: 120,
    sorter: true,
    sortOrder: columnSortOrder('pointsBalance'),
    render: (row) => balanceFormatter.format(row.pointsBalance ?? 0),
  },
  {
    title: '金币',
    key: 'coinsBalance',
    width: 120,
    sorter: true,
    sortOrder: columnSortOrder('coinsBalance'),
    render: (row) => balanceFormatter.format(row.coinsBalance ?? 0),
  },
  {
    title: 'VIP 等级',
    key: 'vipLevel',
    width: 150,
    sorter: true,
    sortOrder: columnSortOrder('vipLevel'),
    render: (row) =>
      row.vipLevel
        ? h(
            NTag,
            { size: 'small', bordered: false },
            { default: () => `VIP${row.vipLevel} · ${row.vipTierName || '会员'}` },
          )
        : '-',
  },
  statusColumn<Member>('状态', ACTIVE_STATUS, (r) => r.status, { width: 90 }),
  dateColumn<Member>('注册时间', (r) => r.createdAt, { width: 170 }),
  actionColumn<Member>(
    (row) =>
      h(NSpace, { wrap: false }, () => [
        h(
          PermissionButton,
          {
            permissions: [PERM.memberRead, PERM.memberReadLimited],
            onClick: () => openDetail(row),
          },
          { default: () => '详情' },
        ),
        h(
          PermissionButton,
          {
            permissions: [PERM.memberWalletAdjustRequest],
            type: 'primary',
            onClick: () => openAdjust(row),
          },
          { default: () => '人工调账' },
        ),
      ]),
    '操作',
    180,
  ),
])

const ledgerColumns = computed<DataTableColumns<WalletLedgerEntry>>(() => [
  textColumn<WalletLedgerEntry>('类型', (r) => ASSET_LABELS[r.assetType] ?? r.assetType),
  textColumn<WalletLedgerEntry>(
    '变动',
    (r) => `${r.direction === 'debit' ? '-' : '+'}${r.amount}`,
    { align: 'right' },
  ),
  textColumn<WalletLedgerEntry>('变动后余额', (r) => r.balanceAfter, { align: 'right' }),
  textColumn<WalletLedgerEntry>(
    '原因',
    (r) => WALLET_REASON_LABELS[r.reason || ''] ?? r.reason,
  ),
  dateColumn<WalletLedgerEntry>('时间', (r) => r.createdAt, { width: 150 }),
])

const couponColumns = computed<DataTableColumns<MemberCouponEntitlement>>(() => [
  textColumn<MemberCouponEntitlement>('券名称', (r) => r.templateName, { minWidth: 150 }),
  textColumn<MemberCouponEntitlement>('类型', (r) => COUPON_TYPE_LABELS[r.couponType] ?? r.couponType, {
    width: 110,
  }),
  statusColumn<MemberCouponEntitlement>('状态', COUPON_STATUS, (r) => r.status, { width: 90 }),
  dateColumn<MemberCouponEntitlement>('有效期至', (r) => r.expiresAt, { width: 170 }),
  dateColumn<MemberCouponEntitlement>('发放时间', (r) => r.createdAt, { width: 170 }),
])
</script>

<template>
  <section class="member-list">
    <PageHeader
      title="会员列表"
      description="全局会员查询；人工调账和补发券均为高风险操作"
      :breadcrumb="['用户 / 会员', '会员列表']"
    />

    <div class="member-filter">
      <div class="member-filter__fields">
        <label class="member-filter__field">
          <span class="member-filter__label">昵称 / 手机号</span>
          <NInput
            :value="(list.filters.keyword as string) ?? ''"
            clearable
            placeholder="支持昵称、手机号模糊搜索"
            style="width: 280px"
            @update:value="list.filters.keyword = $event"
            @keyup.enter="applyMemberFilters"
          />
        </label>
        <label class="member-filter__field">
          <span class="member-filter__label">注册时间</span>
          <NDatePicker
            v-model:value="registrationRange"
            type="daterange"
            clearable
            style="width: 280px"
          />
        </label>
      </div>
      <NSpace class="member-filter__actions">
        <NButton
          type="primary"
          size="small"
          :loading="list.loading.value"
          @click="applyMemberFilters"
        >
          查询
        </NButton>
        <NButton
          size="small"
          @click="resetMemberFilters"
        >
          重置
        </NButton>
      </NSpace>
    </div>

    <div class="member-table">
      <DataTable
        :columns="columns"
        :data="list.rows.value"
        :loading="list.loading.value"
        :page="list.page.value"
        :page-size="list.pageSize.value"
        :total="list.total.value"
        :scroll-x="1360"
        empty-text="暂无会员"
        @update:page="list.setPage"
        @update:page-size="list.setPageSize"
        @update:sorter="handleSorter"
      />
    </div>

    <NModal
      v-model:show="detailShow"
      preset="card"
      title="会员详情"
      style="width: 760px; max-width: 92vw"
    >
      <NSpin :show="detailLoading">
        <div
          v-if="currentMember"
          class="member-detail__profile"
        >
          <NAvatar
            v-if="currentMember.avatarUrl"
            :size="72"
            round
            :src="currentMember.avatarUrl"
            object-fit="cover"
          >
            <template #fallback>
              {{ currentMember.nickname?.trim().slice(0, 1) || String(currentMember.id).slice(-1) }}
            </template>
          </NAvatar>
          <NAvatar
            v-else
            :size="72"
            round
          >
            {{ currentMember.nickname?.trim().slice(0, 1) || String(currentMember.id).slice(-1) }}
          </NAvatar>
          <div class="member-detail__identity">
            <strong>{{ currentMember.nickname || '—' }}</strong>
            <span>{{ currentMember.phone || '—' }}</span>
          </div>
        </div>
        <div class="member-detail__summary">
          <div>
            <span class="ic-muted">用户 ID：</span>{{ currentMember?.id ?? '-' }}
          </div>
          <div>
            <span class="ic-muted">注册时间：</span>{{ formatDateTime(currentMember?.createdAt) }}
          </div>
          <div>
            <span class="ic-muted">当前积分：</span>{{ currentMember?.pointsBalance ?? 0 }}
          </div>
          <div>
            <span class="ic-muted">金币：</span>{{ currentMember?.coinsBalance ?? 0 }}
          </div>
          <div>
            <span class="ic-muted">VIP 等级：</span>
            {{ currentMember?.vipLevel ? `VIP${currentMember.vipLevel} · ${currentMember.vipTierName || '会员'}` : '-' }}
          </div>
          <div>
            <span class="ic-muted">状态：</span>{{ ACTIVE_STATUS[currentMember?.status as keyof typeof ACTIVE_STATUS]?.label ?? currentMember?.status ?? '-' }}
          </div>
          <div
            v-for="acc in currentMember?.wallet ?? []"
            :key="acc.assetType"
          >
            <span class="ic-muted">{{ ASSET_LABELS[acc.assetType] ?? acc.assetType }}：</span>{{ acc.availableAmount }}
          </div>
        </div>

        <NTabs
          v-model:value="detailTab"
          type="line"
        >
          <NTabPane
            name="coupons"
            tab="用户优惠券"
          >
            <div class="member-detail__coupon-toolbar">
              <span>仅显示当前门店发放的优惠券。</span>
              <PermissionButton
                :permissions="[PERM.couponWrite]"
                type="primary"
                :loading="couponLoading"
                @click="openCouponGrant"
              >
                补发券
              </PermissionButton>
            </div>
            <DataTable
              :columns="couponColumns"
              :data="memberCoupons.rows.value"
              :loading="memberCoupons.loading.value"
              :page="memberCoupons.page.value"
              :page-size="memberCoupons.pageSize.value"
              :total="memberCoupons.total.value"
              :row-key="(row) => row.entitlementId"
              :scroll-x="720"
              empty-text="该用户暂无本店优惠券"
              @update:page="memberCoupons.setPage"
              @update:page-size="memberCoupons.setPageSize"
            />
          </NTabPane>

          <NTabPane
            name="ledger"
            tab="钱包流水"
          >
            <DataTable
              :columns="ledgerColumns"
              :data="ledger.rows.value"
              :loading="ledger.loading.value"
              :page="ledger.page.value"
              :page-size="ledger.pageSize.value"
              :total="ledger.total.value"
              :row-key="(row) => row.recordKey"
              empty-text="暂无流水"
              @update:page="ledger.setPage"
              @update:page-size="ledger.setPageSize"
            />
          </NTabPane>

          <NTabPane
            name="adjust"
            tab="人工调账"
          >
            <div class="member-detail__adjust">
              <label>
                <span class="ic-muted">调账类型</span>
                <NSelect
                  v-model:value="adjustForm.assetType"
                  :options="adjustAssetTypeOptions"
                />
              </label>
              <label>
                <span class="ic-muted">调整方式</span>
                <NRadioGroup
                  v-model:value="adjustForm.direction"
                  name="wallet-adjustment-direction"
                  class="adjust-direction"
                >
                  <NRadioButton
                    value="credit"
                    class="adjust-direction__option"
                  >
                    增加
                  </NRadioButton>
                  <NRadioButton
                    value="debit"
                    class="adjust-direction__option"
                  >
                    减少
                  </NRadioButton>
                </NRadioGroup>
              </label>
              <label>
                <span class="ic-muted">调整数量</span>
                <NInputNumber
                  v-model:value="adjustForm.amount"
                  :min="1"
                  :precision="0"
                  clearable
                  placeholder="请输入正整数"
                  style="width: 100%"
                />
              </label>
              <label>
                <span class="ic-muted">调账原因</span>
                <NInput
                  v-model:value="adjustForm.reason"
                  type="textarea"
                  placeholder="必填，用于审计"
                />
              </label>
              <PermissionButton
                :permissions="[PERM.memberWalletAdjustRequest]"
                type="primary"
                :loading="action.running.value"
                :disabled="adjustForm.amount == null || adjustForm.amount <= 0 || !adjustForm.reason.trim()"
                @click="submitAdjust"
              >
                提交调账
              </PermissionButton>
            </div>
          </NTabPane>
        </NTabs>
      </NSpin>

      <template #footer>
        <div class="member-detail__footer">
          <NButton @click="detailShow = false">
            关闭
          </NButton>
        </div>
      </template>
    </NModal>

    <NModal
      v-model:show="couponShow"
      preset="card"
      :title="`补发本店优惠券 · ${couponTarget?.nickname || couponTarget?.id || ''}`"
      :mask-closable="false"
      style="width: 520px; max-width: 92vw"
    >
      <div class="coupon-grant">
        <p class="coupon-grant__notice">
          选择总后台启用的券种，发放后仅限当前门店使用。
        </p>
        <label>
          <span>券种</span>
          <NSelect
            v-model:value="couponForm.templateId"
            :options="couponTemplateOptions"
            filterable
            placeholder="选择启用中的券种"
            @update:value="handleCouponKindChange"
          />
        </label>
        <label>
          <span>有效期至</span>
          <NDatePicker
            v-model:value="couponForm.expiresAt"
            type="datetime"
            clearable
            style="width: 100%"
          />
        </label>
        <label>
          <span>补发原因</span>
          <NInput
            v-model:value="couponForm.reason"
            type="textarea"
            :autosize="{ minRows: 3, maxRows: 5 }"
            maxlength="200"
            show-count
            placeholder="必填，将写入审计日志"
          />
        </label>
      </div>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="couponShow = false">
            取消
          </NButton>
          <PermissionButton
            :permissions="[PERM.couponWrite]"
            type="primary"
            :loading="couponAction.running.value"
            @click="submitCouponGrant"
          >
            确认补发
          </PermissionButton>
        </NSpace>
      </template>
    </NModal>
  </section>
</template>

<style scoped>
.member-list {
  max-width: 1400px;
}
.member-filter {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--ic-space-4);
  padding: var(--ic-space-4);
  margin-bottom: var(--ic-space-4);
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
  flex-wrap: wrap;
}
.member-filter__fields {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ic-space-4);
}
.member-filter__field {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-1);
}
.member-filter__label {
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-secondary);
}
.member-filter__actions {
  flex-shrink: 0;
}
.member-table {
  padding: var(--ic-space-2);
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
}
.member-detail__profile {
  display: flex;
  align-items: center;
  gap: var(--ic-space-4);
  padding-bottom: var(--ic-space-4);
  margin-bottom: var(--ic-space-4);
  border-bottom: 1px solid var(--ic-color-border);
}
.member-detail__identity {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-1);
}
.member-detail__identity strong {
  font-size: var(--ic-font-md);
}
.member-detail__identity span {
  color: var(--ic-color-text-secondary);
}
.member-detail__summary {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--ic-space-2);
  margin-bottom: var(--ic-space-4);
  font-size: var(--ic-font-sm);
}
.member-detail__adjust {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-4);
  max-width: 360px;
  padding-top: var(--ic-space-3);
}
.member-detail__adjust label {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-2);
  font-size: var(--ic-font-sm);
}
.adjust-direction {
  display: flex;
  width: 100%;
}
.adjust-direction__option {
  flex: 1;
  text-align: center;
}
.member-detail__coupon-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-4);
  padding: var(--ic-space-3) 0;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}
.member-detail__footer {
  display: flex;
  justify-content: flex-end;
}
.coupon-grant {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-4);
}
.coupon-grant__notice {
  margin: 0;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}
.coupon-grant label {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-2);
  font-size: var(--ic-font-sm);
}
</style>
