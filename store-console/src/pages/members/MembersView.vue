<script setup lang="ts">
/**
 * 会员管理：会员列表、详情查看与人工钱包调账。
 * 人工调账为高风险写操作，服务端带 Idempotency-Key，需二次确认。
 */
import { computed, h, reactive, ref } from 'vue'
import { NButton, NInput, NInputNumber, NModal, NSelect, NSpin, NTabPane, NTabs } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { memberService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { PERM } from '@/constants/permissions'
import { actionColumn, dateColumn, phoneColumn, textColumn } from '@/utils/columns'
import { ApiError } from '@/api/error'
import { DataTable, PageHeader, PermissionButton, StatusFilterBar } from '@/components/common'
import type { Member, WalletLedgerEntry } from '@/types/models'

const list = useAsyncList<Member>((params) => memberService.list(params), {
  initialFilters: { keyword: '' },
})

const detailShow = ref(false)
const detailLoading = ref(false)
const currentMember = ref<Member | null>(null)

const ledger = useAsyncList<WalletLedgerEntry>(
  (params) => memberService.walletLedger(params),
  { immediate: false },
)

const action = useAsyncAction()

const assetTypeOptions = [
  { label: '余额', value: 'cash_balance' },
  { label: '积分', value: 'points' },
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

async function openDetail(row: Member) {
  currentMember.value = row
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
  textColumn<Member>('昵称', (r) => r.nickname),
  phoneColumn<Member>('手机号', (r) => r.phone),
  textColumn<Member>('积分', (r) => r.pointsBalance, { width: 90, align: 'right' }),
  dateColumn<Member>('注册时间', (r) => r.createdAt, { width: 150 }),
  actionColumn<Member>(
    (row) =>
      h(
        PermissionButton,
        {
          permissions: [PERM.memberRead, PERM.memberReadLimited],
          type: 'primary',
          text: true,
          onClick: () => openDetail(row),
        },
        { default: () => '详情' },
      ),
    '操作',
    100,
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
  textColumn<WalletLedgerEntry>('原因', (r) => r.reason),
  dateColumn<WalletLedgerEntry>('时间', (r) => r.createdAt, { width: 150 }),
])
</script>

<template>
  <div>
    <PageHeader
      title="会员管理"
      description="本店会员列表与钱包流水"
    />

    <StatusFilterBar
      :searchable="true"
      :keyword="(list.filters.keyword as string) ?? ''"
      :loading="list.loading.value"
      search-placeholder="搜索昵称 / 手机号"
      @update:keyword="list.filters.keyword = $event"
      @apply="list.applyFilters({})"
      @reset="list.reset()"
    />

    <DataTable
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      empty-text="暂无会员"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />

    <NModal
      v-model:show="detailShow"
      preset="card"
      title="会员详情"
      style="width: 640px"
    >
      <NSpin :show="detailLoading">
        <div class="member-detail__summary">
          <div>
            <span class="ic-muted">昵称：</span>{{ currentMember?.nickname ?? '-' }}
          </div>
          <div>
            <span class="ic-muted">手机号：</span>{{ currentMember?.phone ?? '-' }}
          </div>
          <div>
            <span class="ic-muted">积分：</span>{{ currentMember?.pointsBalance ?? 0 }}
          </div>
          <div
            v-for="acc in currentMember?.wallet ?? []"
            :key="acc.assetType"
          >
            <span class="ic-muted">{{ ASSET_LABELS[acc.assetType] ?? acc.assetType }}：</span>{{ acc.availableAmount }}
          </div>
        </div>

        <NTabs
          type="line"
          default-value="ledger"
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
  </div>
</template>

<style scoped>
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
