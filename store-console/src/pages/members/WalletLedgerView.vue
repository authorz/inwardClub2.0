<script setup lang="ts">
/**
 * 本店会员资产流水。会员身份是平台级数据，流水由服务端按登录门店强制限域；
 * 页面不提供门店筛选，也不向接口传递 storeId。
 */
import { computed, h, ref } from 'vue'
import {
  NAvatar,
  NButton,
  NDatePicker,
  NInput,
  NSelect,
  NSpace,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { DataTable, PageHeader } from '@/components/common'
import { memberService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import {
  toOptions,
  WALLET_ASSET_TYPE,
  WALLET_DIRECTION,
  WALLET_LEDGER_STATUS,
  WALLET_REASON_LABELS,
  WALLET_SOURCE_TYPE,
} from '@/constants/enums'
import { dateColumn, statusColumn, textColumn } from '@/utils/columns'
import type { WalletLedgerEntry } from '@/types/models'

const list = useAsyncList<WalletLedgerEntry>(
  (params) => memberService.walletLedger(params),
  {
    initialFilters: {
      id: '',
      memberNickname: '',
      memberPhone: '',
      assetType: '',
      direction: '',
      sourceType: '',
      status: '',
      reason: '',
    },
  },
)

const createdRange = ref<[number, number] | null>(null)
const selectOptions = (options: ReturnType<typeof toOptions>) =>
  options.map(({ label, value }) => ({ label, value }))
const assetTypeOptions = selectOptions(toOptions(WALLET_ASSET_TYPE))
const directionOptions = selectOptions(toOptions(WALLET_DIRECTION))
const sourceTypeOptions = selectOptions(toOptions(WALLET_SOURCE_TYPE))
const statusOptions = selectOptions(toOptions(WALLET_LEDGER_STATUS))

function applyFilters(): void {
  if (createdRange.value) {
    list.filters.createdFrom = new Date(createdRange.value[0]).toISOString()
    list.filters.createdTo = new Date(createdRange.value[1]).toISOString()
  } else {
    delete list.filters.createdFrom
    delete list.filters.createdTo
  }
  list.applyFilters({})
}

function resetFilters(): void {
  createdRange.value = null
  list.reset()
}

const columns = computed<DataTableColumns<WalletLedgerEntry>>(() => [
  textColumn<WalletLedgerEntry>('ID', (row) => row.id, { width: 64 }),
  {
    title: '会员信息',
    key: 'member',
    width: 180,
    render: (row) => {
      const nickname = row.memberNickname?.trim() || `会员 ${row.memberId}`
      const phone = row.memberPhone?.trim() || '暂无手机号'
      const fallback = () => nickname.slice(0, 1)
      return h('div', { class: 'ledger-member' }, [
        h(
          NAvatar,
          {
            size: 32,
            round: true,
            src: row.memberAvatarUrl || undefined,
            objectFit: 'cover',
          },
          row.memberAvatarUrl ? { fallback } : { default: fallback },
        ),
        h('div', { class: 'ledger-member__identity' }, [
          h('strong', { title: nickname }, nickname),
          h('span', phone),
        ]),
      ])
    },
  },
  textColumn<WalletLedgerEntry>('门店', (row) => row.storeName, { width: 120 }),
  statusColumn<WalletLedgerEntry>('资产', WALLET_ASSET_TYPE, (row) => row.assetType, { width: 86 }),
  statusColumn<WalletLedgerEntry>('方向', WALLET_DIRECTION, (row) => row.direction, { width: 86 }),
  textColumn<WalletLedgerEntry>('变动数量', (row) => row.amount, { width: 100, align: 'right' }),
  textColumn<WalletLedgerEntry>('变动后余额', (row) => row.balanceAfter, { width: 112, align: 'right' }),
  statusColumn<WalletLedgerEntry>('处理状态', WALLET_LEDGER_STATUS, (row) => row.status, { width: 100 }),
  textColumn<WalletLedgerEntry>(
    '业务来源',
    (row) => WALLET_SOURCE_TYPE[row.sourceType ?? '']?.label ?? row.sourceType,
    { width: 130 },
  ),
  textColumn<WalletLedgerEntry>('关联订单号', (row) => row.relatedOrderNo, { width: 180 }),
  {
    title: '原因',
    key: 'reason',
    width: 160,
    render: (row) => {
      const reason = WALLET_REASON_LABELS[row.reason ?? ''] ?? row.reason ?? '-'
      return h('span', { class: 'ledger-ellipsis', title: reason }, reason)
    },
  },
  dateColumn<WalletLedgerEntry>('变动时间', (row) => row.createdAt, { width: 170 }),
])
</script>

<template>
  <section class="wallet-ledger">
    <PageHeader
      title="资产流水"
      description="查看本店产生的会员积分、金币、余额、成长值以及存取积分申请记录"
      :breadcrumb="['会员管理', '资产流水']"
    />

    <div class="ledger-filter">
      <div class="ledger-filter__grid">
        <label class="ledger-filter__field">
          <span>流水 ID</span>
          <NInput
            :value="(list.filters.id as string) ?? ''"
            clearable
            placeholder="支持模糊搜索"
            @update:value="list.filters.id = $event"
            @keyup.enter="applyFilters"
          />
        </label>
        <label class="ledger-filter__field">
          <span>会员昵称</span>
          <NInput
            :value="(list.filters.memberNickname as string) ?? ''"
            clearable
            placeholder="支持模糊搜索"
            @update:value="list.filters.memberNickname = $event"
            @keyup.enter="applyFilters"
          />
        </label>
        <label class="ledger-filter__field">
          <span>会员手机号</span>
          <NInput
            :value="(list.filters.memberPhone as string) ?? ''"
            clearable
            placeholder="支持模糊搜索"
            @update:value="list.filters.memberPhone = $event"
            @keyup.enter="applyFilters"
          />
        </label>
        <label class="ledger-filter__field">
          <span>资产类型</span>
          <NSelect
            :value="(list.filters.assetType as string) || null"
            :options="assetTypeOptions"
            clearable
            placeholder="全部"
            @update:value="list.filters.assetType = $event ?? ''"
          />
        </label>
        <label class="ledger-filter__field">
          <span>变动方向</span>
          <NSelect
            :value="(list.filters.direction as string) || null"
            :options="directionOptions"
            clearable
            placeholder="全部"
            @update:value="list.filters.direction = $event ?? ''"
          />
        </label>
        <label class="ledger-filter__field">
          <span>业务来源</span>
          <NSelect
            :value="(list.filters.sourceType as string) || null"
            :options="sourceTypeOptions"
            clearable
            filterable
            placeholder="全部"
            @update:value="list.filters.sourceType = $event ?? ''"
          />
        </label>
        <label class="ledger-filter__field">
          <span>处理状态</span>
          <NSelect
            :value="(list.filters.status as string) || null"
            :options="statusOptions"
            clearable
            placeholder="全部"
            @update:value="list.filters.status = $event ?? ''"
          />
        </label>
        <label class="ledger-filter__field">
          <span>原因</span>
          <NInput
            :value="(list.filters.reason as string) ?? ''"
            clearable
            placeholder="支持模糊搜索"
            @update:value="list.filters.reason = $event"
            @keyup.enter="applyFilters"
          />
        </label>
        <label class="ledger-filter__field ledger-filter__field--date">
          <span>变动时间</span>
          <NDatePicker
            v-model:value="createdRange"
            type="daterange"
            clearable
          />
        </label>
      </div>
      <NSpace class="ledger-filter__actions">
        <NButton
          type="primary"
          size="small"
          :loading="list.loading.value"
          @click="applyFilters"
        >
          查询
        </NButton>
        <NButton
          size="small"
          @click="resetFilters"
        >
          重置
        </NButton>
      </NSpace>
    </div>

    <div class="ledger-table">
      <DataTable
        :columns="columns"
        :data="list.rows.value"
        :loading="list.loading.value"
        :page="list.page.value"
        :page-size="list.pageSize.value"
        :total="list.total.value"
        :row-key="(row) => row.recordKey"
        :scroll-x="1540"
        empty-text="本店暂无资产流水"
        @update:page="list.setPage"
        @update:page-size="list.setPageSize"
      />
    </div>
  </section>
</template>

<style scoped>
.wallet-ledger {
  max-width: 1600px;
}

.ledger-filter {
  padding: 0 0 var(--ic-space-5);
  margin-bottom: var(--ic-space-4);
  border-bottom: var(--ic-divider);
}

.ledger-filter__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(180px, 1fr));
  gap: var(--ic-space-4);
}

.ledger-filter__field {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--ic-space-1);
}

.ledger-filter__field > span {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.ledger-filter__field--date {
  grid-column: span 2;
}

.ledger-filter__actions {
  margin-top: var(--ic-space-4);
}

.ledger-table {
  overflow: hidden;
  background: var(--ic-color-surface);
  border-top: var(--ic-divider);
  border-bottom: var(--ic-divider);
}

.ledger-member {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: var(--ic-space-2);
  padding: 2px 0;
}

.ledger-member__identity {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.ledger-member__identity strong,
.ledger-member__identity span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ledger-member__identity strong {
  color: var(--ic-color-text);
  font-size: var(--ic-font-sm);
}

.ledger-member__identity span {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
  font-variant-numeric: tabular-nums;
}

.ledger-ellipsis {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1200px) {
  .ledger-filter__grid {
    grid-template-columns: repeat(3, minmax(180px, 1fr));
  }
}

@media (max-width: 900px) {
  .ledger-filter__grid {
    grid-template-columns: repeat(2, minmax(160px, 1fr));
  }
}

@media (max-width: 640px) {
  .ledger-filter__grid {
    grid-template-columns: 1fr;
  }

  .ledger-filter__field--date {
    grid-column: span 1;
  }
}
</style>
