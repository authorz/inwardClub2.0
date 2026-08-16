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
  NSelect,
  NSpace,
  NSpin,
  NTabPane,
  NTabs,
  NTag,
} from 'naive-ui'
import type { DataTableColumns, DataTableSortOrder, DataTableSortState } from 'naive-ui'
import { memberService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { PERM } from '@/constants/permissions'
import { actionColumn, dateColumn, statusColumn, textColumn } from '@/utils/columns'
import { formatDateTime } from '@/utils/format'
import { ApiError } from '@/api/error'
import { DataTable, PageHeader, PermissionButton } from '@/components/common'
import type { Member, WalletLedgerEntry } from '@/types/models'
import { ACTIVE_STATUS } from '@/constants/enums'

type MemberSortField = 'pointsBalance' | 'coinsBalance' | 'vipLevel'
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
const detailTab = ref<'ledger' | 'adjust'>('ledger')

const ledger = useAsyncList<WalletLedgerEntry>(
  (params) => memberService.walletLedger(params),
  { immediate: false },
)

const action = useAsyncAction()

const assetTypeOptions = [
  { label: '积分', value: 'points' },
  { label: '金币', value: 'coins' },
  { label: '余额', value: 'cash_balance' },
  { label: '成长值', value: 'growth_value' },
]
const ASSET_LABELS: Record<string, string> = {
  cash_balance: '余额',
  points: '积分',
  coins: '金币',
  growth_value: '成长值',
}
const adjustForm = reactive<{ assetType: string; changeAmount: number | null; reason: string }>({
  assetType: 'cash_balance',
  changeAmount: null,
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

async function openMember(row: Member, tab: 'ledger' | 'adjust') {
  currentMember.value = row
  detailTab.value = tab
  detailShow.value = true
  detailLoading.value = true
  adjustForm.assetType = 'cash_balance'
  adjustForm.changeAmount = null
  adjustForm.reason = ''
  try {
    currentMember.value = await memberService.detail(row.id)
  } catch (err) {
    if (!(err instanceof ApiError)) throw err
  } finally {
    detailLoading.value = false
  }
  ledger.filters.memberId = row.id
  ledger.applyFilters({})
}

function openDetail(row: Member): void {
  void openMember(row, 'ledger')
}

function openAdjust(row: Member): void {
  void openMember(row, 'adjust')
}

function submitAdjust() {
  if (!currentMember.value || adjustForm.changeAmount == null || adjustForm.changeAmount === 0 || !adjustForm.reason)
    return
  // 服务端以 direction(credit/debit) + 正整数 amount 建模；界面用带符号数量表达增减。
  const signed = adjustForm.changeAmount
  void action.run(
    () =>
      memberService.adjustWallet(currentMember.value!.id, {
        assetType: adjustForm.assetType,
        direction: signed >= 0 ? 'credit' : 'debit',
        amount: Math.abs(signed),
        reason: adjustForm.reason,
      }),
    {
      confirm: { content: '确认提交人工调账？该操作将写入会员钱包流水', danger: true },
      successMessage: '调账已提交',
      onSuccess: async () => {
        adjustForm.changeAmount = null
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
  textColumn<WalletLedgerEntry>('原因', (r) => ({
    food_order_reward: '购买餐品赠送积分',
    food_order_cancel_clawback: '取消订单扣回赠送积分',
    food_order_cancel_rollback: '取消订单失败返还积分',
    order_payment: '订单支付', refund: '订单退款返还',
  }[r.reason || ''] ?? r.reason)),
  dateColumn<WalletLedgerEntry>('时间', (r) => r.createdAt, { width: 150 }),
])
</script>

<template>
  <section class="member-list">
    <PageHeader
      title="会员列表"
      description="全局会员查询；人工调账为高风险操作"
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
                  :options="assetTypeOptions"
                />
              </label>
              <label>
                <span class="ic-muted">变动数量（正数增加，负数扣减）</span>
                <NInputNumber
                  v-model:value="adjustForm.changeAmount"
                  placeholder="如 100 或 -50"
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
                :disabled="adjustForm.changeAmount == null || adjustForm.changeAmount === 0 || !adjustForm.reason"
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
.member-detail__footer {
  display: flex;
  justify-content: flex-end;
}
</style>
