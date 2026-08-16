<script setup lang="ts">
/**
 * 会员列表 + 人工调账（高风险）。
 * 人工调账涉及钱包，必须选择资产类型、填写原因、二次确认，携带幂等键并写入审计。
 */
import { computed, h, reactive, ref } from 'vue'
import {
  NAvatar,
  NButton,
  NForm,
  NFormItem,
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
import type { DataTableSortState, DataTableSortOrder } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import DataTable from '@/components/DataTable.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { dateTimeColumn, statusColumn, textColumn, actionsColumn, renderColumn } from '@/utils/columns'
import { ASSET_TYPE_OPTIONS, RESOURCE_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { useDataTable } from '@/composables/useDataTable'
import { memberService, readonlyLists } from '@/api/services'
import { http } from '@/api/http'
import { API_PATHS } from '@/constants/api-paths'
import { formatDateTime } from '@/utils/format'
import type { Member, MemberDetail, WalletLedgerEntry } from '@/api/models'
import type { ListQuery } from '@/api/types'
import type { FilterField, TableColumnList } from '@/components/ui-types'
import { toastError } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)
type MemberSortField = 'pointsBalance' | 'coinsBalance' | 'vipLevel'
const sortBy = ref<MemberSortField | ''>('')
const sortOrder = ref<'asc' | 'desc'>('desc')
const balanceFormatter = new Intl.NumberFormat('zh-CN')

const fields: FilterField[] = [
  {
    key: 'keyword',
    label: '昵称 / 手机号',
    type: 'input',
    placeholder: '支持昵称、手机号模糊搜索',
    width: 280,
  },
  {
    key: 'created',
    label: '注册时间',
    type: 'daterange',
    width: 280,
  },
]

const columns = computed<TableColumnList<Member>>(() => [
  textColumn<Member>('ID', 'id', { width: 80 }),
  renderColumn<Member>(
    '头像',
    'avatarUrl',
    (row) => {
      const fallback = () => row.nickname?.trim().slice(0, 1) || String(row.id).slice(-1)
      return h(
        NAvatar,
        { size: 32, round: true, src: row.avatarUrl || undefined, objectFit: 'cover' },
        row.avatarUrl ? { fallback } : { default: fallback },
      )
    },
    64,
  ),
  textColumn<Member>('昵称', 'nickname'),
  textColumn<Member>('手机号', 'phone', { width: 140 }),
  textColumn<Member>('当前积分', 'pointsBalance', {
    width: 120,
    sorter: true,
    sortOrder: columnSortOrder('pointsBalance'),
    render: (row) => balanceFormatter.format(row.pointsBalance ?? 0),
  }),
  textColumn<Member>('金币', 'coinsBalance', {
    width: 120,
    sorter: true,
    sortOrder: columnSortOrder('coinsBalance'),
    render: (row) => balanceFormatter.format(row.coinsBalance ?? 0),
  }),
  textColumn<Member>('VIP 等级', 'vipLevel', {
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
        : '—',
  }),
  statusColumn<Member>('状态', 'status', RESOURCE_STATUS_OPTIONS, 100),
  dateTimeColumn<Member>('注册时间', 'createdAt'),
  actionsColumn<Member>(
    (row) =>
      h(NSpace, {}, () => [
        h(
          PermissionButton,
          { permission: PERMISSIONS.MEMBER_READ, onClick: () => openDetail(row) },
          () => '详情',
        ),
        h(
          PermissionButton,
          {
            permission: PERMISSIONS.MEMBER_WALLET_ADJUST,
            type: 'primary',
            onClick: () => openAdjust(row),
          },
          () => '人工调账',
        ),
      ]),
    180,
  ),
])

function columnSortOrder(field: MemberSortField): DataTableSortOrder {
  if (sortBy.value !== field) return false
  return sortOrder.value === 'asc' ? 'ascend' : 'descend'
}

function fetchMembers(query: ListQuery) {
  return memberService.list({
    ...query,
    ...(sortBy.value ? { sortBy: sortBy.value, sortOrder: sortOrder.value } : {}),
  })
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
  void listRef.value?.reload()
}

// —— 会员详情（只读） ——
const detailDrawerShow = ref(false)
const detailLoading = ref(false)
const detail = ref<MemberDetail | null>(null)
const detailTab = ref<'ledger' | 'adjust'>('ledger')

const ledgerTable = useDataTable<WalletLedgerEntry>({
  fetcher: readonlyLists.walletLedger,
  immediate: false,
})

const ledgerReasonLabels: Record<string, string> = {
  point_saving: '存积分',
  point_saving_reward: '存积分审核到账',
  point_saving_coin_reward: '存积分金币奖励',
  point_withdrawal: '取积分',
  food_order_reward: '购买餐品赠送积分',
  food_order_cancel_clawback: '取消订单扣回赠送积分',
  food_order_cancel_rollback: '取消订单失败返还积分',
  sign_in: '签到奖励',
  recharge: '充值到账',
  first_recharge_reward: '用户首充获得积分',
  high_value_recharge_reward: '满额充值获得积分',
  recharge_growth: '充值成长值',
  wechat_payment_growth: '微信支付获得成长值',
  order_payment: '订单支付',
  refund: '订单退款返还',
  admin_adjustment: '管理员调账',
  low_spend_reward: '预约低消达标奖励',
}

const ledgerColumns: TableColumnList<WalletLedgerEntry> = [
  renderColumn<WalletLedgerEntry>(
    '类型',
    'assetType',
    (row) => assetTypeLabel(row.assetType),
    88,
  ),
  renderColumn<WalletLedgerEntry>(
    '变动',
    'amount',
    (row) => `${row.direction === 'debit' ? '-' : '+'}${balanceFormatter.format(row.amount)}`,
    100,
  ),
  renderColumn<WalletLedgerEntry>(
    '变动后余额',
    'balanceAfter',
    (row) => balanceFormatter.format(row.balanceAfter ?? 0),
    120,
  ),
  renderColumn<WalletLedgerEntry>(
    '原因',
    'reason',
    (row) => ledgerReasonLabels[row.reason ?? ''] ?? row.reason ?? '—',
  ),
  dateTimeColumn<WalletLedgerEntry>('时间', 'createdAt', 168),
]

function assetTypeLabel(assetType: string): string {
  return ASSET_TYPE_OPTIONS.find((o) => o.value === assetType)?.label ?? assetType
}

function statusLabel(status: string): string {
  return RESOURCE_STATUS_OPTIONS.find((o) => o.value === status)?.label ?? status
}

async function openDetail(row: Member): Promise<void> {
  detail.value = null
  detailTab.value = 'ledger'
  target.value = row
  resetAdjustForm()
  detailDrawerShow.value = true
  detailLoading.value = true
  ledgerTable.filters.memberId = row.id
  ledgerTable.pagination.page = 1
  void ledgerTable.load()
  try {
    detail.value = await http.get<MemberDetail>(API_PATHS.members.detail(row.id))
  } catch (e) {
    toastError((e as { message?: string }).message ?? '加载会员详情失败')
    detailDrawerShow.value = false
  } finally {
    detailLoading.value = false
  }
}

// —— 人工调账表单 ——
const drawerShow = ref(false)
const submitting = ref(false)
const target = ref<Member | null>(null)
const adjust = reactive<{ assetType: string | null; amount: number | null; reason: string }>({
  assetType: null,
  amount: null,
  reason: '',
})

function resetAdjustForm(): void {
  adjust.assetType = null
  adjust.amount = null
  adjust.reason = ''
}

function openAdjust(row: Member): void {
  target.value = row
  resetAdjustForm()
  drawerShow.value = true
}

async function submitAdjust(closeOnSuccess = true): Promise<void> {
  if (!target.value) return
  if (!adjust.assetType) return toastError('请选择资产类型')
  if (adjust.amount == null || adjust.amount === 0) return toastError('请输入调整数量（非零，正增负减）')
  if (!adjust.reason.trim()) return toastError('人工调账必须填写原因')

  const member = target.value
  submitting.value = true
  try {
    const ok = await runAudited({
      title: '确认人工调账',
      content: `将对会员「${member.nickname ?? member.id}」的 ${adjust.assetType} 调整 ${adjust.amount}，原因：${adjust.reason}。该操作不可逆，携带幂等键并写入审计。`,
      highRisk: true,
      positiveText: '确认调账',
      execute: () =>
        http.post(
          API_PATHS.members.walletAdjustments(member.id),
          {
            assetType: adjust.assetType,
            direction: adjust.amount! >= 0 ? 'credit' : 'debit',
            amount: Math.abs(adjust.amount!),
            reason: adjust.reason,
          },
          { idempotent: true },
        ),
      successText: '调账已提交',
    })
    if (ok) {
      if (closeOnSuccess) drawerShow.value = false
      resetAdjustForm()
      listRef.value?.reload()
      if (detailDrawerShow.value && detail.value && String(detail.value.id) === String(member.id)) {
        detail.value = await http.get<MemberDetail>(API_PATHS.members.detail(member.id))
        void ledgerTable.reload()
      }
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="会员列表"
      description="全局会员查询；人工调账为高风险操作"
      :breadcrumb="['用户 / 会员', '会员列表']"
      :fields="fields"
      :columns="columns"
      :fetcher="fetchMembers"
      @update:sorter="handleSorter"
    />
    <FormDrawer
      v-model:show="drawerShow"
      title="人工调账"
      :submitting="submitting"
      high-risk
      submit-text="确认调账"
      @submit="submitAdjust()"
    >
      <NForm label-placement="top">
        <NFormItem
          label="资产类型"
          required
        >
          <NSelect
            v-model:value="adjust.assetType"
            :options="ASSET_TYPE_OPTIONS.map((o) => ({ label: o.label, value: o.value }))"
            placeholder="选择资产类型"
          />
        </NFormItem>
        <NFormItem
          label="调整数量（正增 / 负减）"
          required
        >
          <NInputNumber
            v-model:value="adjust.amount"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem
          label="调账原因"
          required
        >
          <NInput
            v-model:value="adjust.reason"
            type="textarea"
            placeholder="请填写调账原因（将写入审计）"
          />
        </NFormItem>
      </NForm>
    </FormDrawer>
    <NModal
      v-model:show="detailDrawerShow"
      preset="card"
      title="会员详情"
      :mask-closable="false"
      style="width: 640px; max-width: 92vw"
      content-style="max-height: 72vh; overflow: auto"
    >
      <NSpin :show="detailLoading">
        <div
          v-if="detail"
          class="member-detail__profile"
        >
          <NAvatar
            v-if="detail.avatarUrl"
            :size="72"
            round
            :src="detail.avatarUrl"
            object-fit="cover"
          >
            <template #fallback>
              {{ detail.nickname?.trim().slice(0, 1) || String(detail.id).slice(-1) }}
            </template>
          </NAvatar>
          <NAvatar
            v-else
            :size="72"
            round
          >
            {{ detail.nickname?.trim().slice(0, 1) || String(detail.id).slice(-1) }}
          </NAvatar>
          <div class="member-detail__identity">
            <strong>{{ detail.nickname || '—' }}</strong>
            <span>{{ detail.phone || '—' }}</span>
          </div>
        </div>
        <template v-if="detail">
          <div class="member-detail__summary">
            <div>
              <span class="member-detail__label">用户 ID：</span>{{ detail.id }}
            </div>
            <div>
              <span class="member-detail__label">注册时间：</span>{{ formatDateTime(detail.createdAt) }}
            </div>
            <div>
              <span class="member-detail__label">当前积分：</span>{{ detail.pointsBalance }}
            </div>
            <div>
              <span class="member-detail__label">金币：</span>{{ detail.coinsBalance }}
            </div>
            <div>
              <span class="member-detail__label">VIP 等级：</span>
              {{ detail.vipLevel ? `VIP${detail.vipLevel} · ${detail.vipTierName || '会员'}` : '—' }}
            </div>
            <div>
              <span class="member-detail__label">状态：</span>{{ statusLabel(detail.status) }}
            </div>
            <div
              v-for="account in detail.wallet"
              :key="account.assetType"
            >
              <span class="member-detail__label">{{ assetTypeLabel(account.assetType) }}：</span>
              {{ account.availableAmount }}
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
                class="member-detail__ledger"
                :columns="ledgerColumns"
                :data="ledgerTable.rows.value"
                :loading="ledgerTable.loading.value"
                :page="ledgerTable.pagination.page"
                :page-size="ledgerTable.pagination.pageSize"
                :item-count="ledgerTable.pagination.itemCount"
                :row-key="(row) => row.recordKey"
                empty-text="暂无流水"
                @update:page="ledgerTable.handlePageChange"
                @update:page-size="ledgerTable.handlePageSizeChange"
              />
            </NTabPane>

            <NTabPane
              name="adjust"
              tab="人工调账"
            >
              <NForm
                class="member-detail__adjust"
                label-placement="top"
              >
                <NFormItem
                  label="资产类型"
                  required
                >
                  <NSelect
                    v-model:value="adjust.assetType"
                    :options="ASSET_TYPE_OPTIONS.map((o) => ({ label: o.label, value: o.value }))"
                    placeholder="选择资产类型"
                  />
                </NFormItem>
                <NFormItem
                  label="调整数量（正增 / 负减）"
                  required
                >
                  <NInputNumber
                    v-model:value="adjust.amount"
                    style="width: 100%"
                  />
                </NFormItem>
                <NFormItem
                  label="调账原因"
                  required
                >
                  <NInput
                    v-model:value="adjust.reason"
                    type="textarea"
                    placeholder="请填写调账原因（将写入审计）"
                  />
                </NFormItem>
                <PermissionButton
                  :permission="PERMISSIONS.MEMBER_WALLET_ADJUST"
                  type="primary"
                  :loading="submitting"
                  :disabled="adjust.amount == null || adjust.amount === 0 || !adjust.reason.trim()"
                  @click="submitAdjust(false)"
                >
                  提交调账
                </PermissionButton>
              </NForm>
            </NTabPane>
          </NTabs>
        </template>
      </NSpin>

      <template #footer>
        <div class="member-detail__footer">
          <NButton @click="detailDrawerShow = false">
            关闭
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.member-detail__profile {
  display: flex;
  align-items: center;
  gap: var(--ic-space-md);
  padding-bottom: var(--ic-space-md);
  margin-bottom: var(--ic-space-md);
  border-bottom: 1px solid var(--ic-color-border);
}
.member-detail__identity {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-xs);
}
.member-detail__identity strong {
  font-size: var(--ic-font-lg);
}
.member-detail__identity span {
  color: var(--ic-color-text-secondary);
}
.member-detail__summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--ic-space-sm) var(--ic-space-lg);
  margin-bottom: var(--ic-space-md);
  font-size: var(--ic-font-sm);
}
.member-detail__label {
  color: var(--ic-color-text-secondary);
}
.member-detail__adjust {
  max-width: 360px;
  padding-top: var(--ic-space-sm);
}
.member-detail__ledger {
  padding: 0;
  border: 0;
  border-radius: 0;
}
.member-detail__footer {
  display: flex;
  justify-content: flex-end;
}
@media (max-width: 640px) {
  .member-detail__summary {
    grid-template-columns: 1fr;
  }
}
</style>
