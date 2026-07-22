<script setup lang="ts">
/**
 * 钱包账本（只读）。记录每一笔资产变动，供财务对账与审计。
 */
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { ASSET_TYPE_OPTIONS } from '@/constants/enums'
import { readonlyLists } from '@/api/services'
import type { WalletLedgerEntry } from '@/api/models'
import type { FilterField } from '@/components/ui-types'

const fields: FilterField[] = [
  { key: 'memberId', label: '会员 ID', type: 'input' },
  { key: 'assetType', label: '资产类型', type: 'select', options: ASSET_TYPE_OPTIONS },
]

const columns = [
  textColumn<WalletLedgerEntry>('会员 ID', 'memberId', { width: 160 }),
  statusColumn<WalletLedgerEntry>('资产类型', 'assetType', ASSET_TYPE_OPTIONS, 110),
  renderColumn<WalletLedgerEntry>(
    '变动',
    'amount',
    (row) => `${row.direction === 'debit' ? '-' : '+'}${row.amount}`,
    110,
  ),
  textColumn<WalletLedgerEntry>('变动后余额', 'balanceAfter', { width: 120 }),
  textColumn<WalletLedgerEntry>('来源', 'sourceType', { width: 120 }),
  textColumn<WalletLedgerEntry>('原因', 'reason'),
  dateTimeColumn<WalletLedgerEntry>('时间', 'createdAt'),
]
</script>

<template>
  <ResourceListView
    title="钱包账本"
    description="资产变动流水（只读），支持对账与审计"
    :breadcrumb="['用户 / 会员', '钱包账本']"
    :fields="fields"
    :columns="columns"
    :fetcher="readonlyLists.walletLedger"
  />
</template>
